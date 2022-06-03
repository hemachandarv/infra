package cmd

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/cli/browser"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/goware/urlx"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/uid"
)

type loginCmdOptions struct {
	Server         string
	AccessKey      string
	Provider       string
	SkipTLSVerify  bool
	NonInteractive bool
}

type loginMethod int8

const (
	localLogin loginMethod = iota
	accessKeyLogin
	oidcLogin
)

const cliLoginRedirectURL = "http://localhost:8301"

func newLoginCmd(cli *CLI) *cobra.Command {
	var options loginCmdOptions

	cmd := &cobra.Command{
		Use:   "login [SERVER]",
		Short: "Login to Infra",
		Example: `# By default, login will prompt for all required information.
$ infra login

# Login to a specific server
$ infra login infraexampleserver.com

# Login with a specific identity provider
$ infra login --provider okta

# Login with an access key
$ infra login --key 1M4CWy9wF5.fAKeKEy5sMLH9ZZzAur0ZIjy`,
		Args:  MaxArgs(1),
		Group: "Core commands:",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				options.Server = args[0]
			}

			return login(cli, options)
		},
	}

	cmd.Flags().StringVar(&options.AccessKey, "key", "", "Login with an access key")
	cmd.Flags().StringVar(&options.Provider, "provider", "", "Login with an identity provider")
	cmd.Flags().BoolVar(&options.SkipTLSVerify, "skip-tls-verify", false, "Skip verifying server TLS certificates")
	addNonInteractiveFlag(cmd.Flags(), &options.NonInteractive)
	return cmd
}

func login(cli *CLI, options loginCmdOptions) error {
	var err error

	if options.Server == "" {
		options.Server, err = promptServer(cli, options)
		if err != nil {
			return err
		}
	}

	client, err := newAPIClient(cli, options)
	if err != nil {
		return err
	}

	loginReq := &api.LoginRequest{}

	// if signup is required, use it to create an admin account
	// and use those credentials for subsequent requests
	signupEnabled, err := client.SignupEnabled()
	if err != nil {
		return err
	}

	if signupEnabled.Enabled {
		loginReq.PasswordCredentials, err = runSignupForLogin(cli, client)
		if err != nil {
			return err
		}

		return loginToInfra(cli, client, loginReq)
	}

	switch {
	case options.AccessKey != "":
		loginReq.AccessKey = options.AccessKey
	case options.Provider != "":
		loginReq.OIDC, err = loginToProviderN(client, options.Provider)
		if err != nil {
			return err
		}
	default:
		if options.NonInteractive {
			return fmt.Errorf("Non-interactive login requires key, instead run: 'infra login SERVER --non-interactive --key KEY")
		}
		loginMethod, provider, err := promptLoginOptions(cli, client)
		if err != nil {
			return err
		}

		switch loginMethod {
		case accessKeyLogin:
			loginReq.AccessKey, err = promptAccessKeyLogin(cli)
			if err != nil {
				return err
			}
		case localLogin:
			loginReq.PasswordCredentials, err = promptLocalLogin(cli)
			if err != nil {
				return err
			}
		case oidcLogin:
			loginReq.OIDC, err = loginToProvider(provider)
			if err != nil {
				return err
			}
		}
	}

	return loginToInfra(cli, client, loginReq)
}

func loginToInfra(cli *CLI, client *api.Client, loginReq *api.LoginRequest) error {
	loginRes, err := client.Login(loginReq)
	if err != nil {
		if api.ErrorStatusCode(err) == http.StatusUnauthorized || api.ErrorStatusCode(err) == http.StatusNotFound {
			switch {
			case loginReq.AccessKey != "":
				return &LoginError{Message: "your access key may be invalid"}
			case loginReq.PasswordCredentials != nil:
				return &LoginError{Message: "your username or password may be invalid"}
			case loginReq.OIDC != nil:
				return &LoginError{Message: "please contact an administrator and check identity provider configurations"}
			}
		}

		return err
	}

	if loginRes.PasswordUpdateRequired {
		fmt.Fprintf(cli.Stderr, "  Your password has expired. Please update your password (min. length 8).\n")

		password, err := promptSetPassword(cli, loginReq.PasswordCredentials.Password)
		if err != nil {
			return err
		}

		client.AccessKey = loginRes.AccessKey
		if _, err := client.UpdateUser(&api.UpdateUserRequest{ID: loginRes.UserID, Password: password}); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "  Updated password.\n")
	}

	if err := updateInfraConfig(client, loginReq, loginRes); err != nil {
		return err
	}

	// Client needs to be refreshed from here onwards, based on the newly saved infra configuration.
	client, err = defaultAPIClient()
	if err != nil {
		return err
	}

	if err := updateKubeconfig(client, loginRes.UserID); err != nil {
		return err
	}

	fmt.Fprintf(cli.Stderr, "  Logged in as %s\n", termenv.String(loginRes.Name).Bold().String())

	backgroundAgentRunning, err := configAgentRunning()
	if err != nil {
		// do not block login, just proceed, potentially without the agent
		logging.S.Errorf("unable to check background agent: %v", err)
	}

	if !backgroundAgentRunning {
		// the agent is started in a separate command so that it continues after the login command has finished
		if err := execAgent(); err != nil {
			// user still has a valid session, so do not fail
			logging.S.Errorf("Unable to start agent, destinations will not be updated automatically: %w", err)
		}
	}

	return nil
}

// Updates all configs with the current logged in session
func updateInfraConfig(client *api.Client, loginReq *api.LoginRequest, loginRes *api.LoginResponse) error {
	clientHostConfig := ClientHostConfig{
		Current:       true,
		PolymorphicID: uid.NewIdentityPolymorphicID(loginRes.UserID),
		Name:          loginRes.Name,
		AccessKey:     loginRes.AccessKey,
		Expires:       loginRes.Expires,
	}

	t, ok := client.HTTP.Transport.(*http.Transport)
	if !ok {
		return fmt.Errorf("Could not update config due to an internal error")
	}
	clientHostConfig.SkipTLSVerify = t.TLSClientConfig.InsecureSkipVerify

	if loginReq.OIDC != nil {
		clientHostConfig.ProviderID = loginReq.OIDC.ProviderID
	}

	u, err := urlx.Parse(client.URL)
	if err != nil {
		return err
	}
	clientHostConfig.Host = u.Host

	if err := saveHostConfig(clientHostConfig); err != nil {
		return err
	}

	return nil
}

func oidcflow(host string, clientId string) (string, error) {
	// find out what the authorization endpoint is
	provider, err := oidc.NewProvider(context.Background(), fmt.Sprintf("https://%s", host))
	if err != nil {
		return "", fmt.Errorf("get provider oidc info: %w", err)
	}

	// claims are the attributes of the user we want to know from the identity provider
	var claims struct {
		ScopesSupported []string `json:"scopes_supported"`
	}

	if err := provider.Claims(&claims); err != nil {
		return "", fmt.Errorf("parsing claims: %w", err)
	}

	scopes := []string{"openid", "email"} // openid and email are required scopes for login to work

	// we want to be able to use these scopes to access groups, but they are not needed
	wantScope := map[string]bool{
		"groups":         true,
		"offline_access": true,
	}

	for _, scope := range claims.ScopesSupported {
		if wantScope[scope] {
			scopes = append(scopes, scope)
		}
	}

	// the state makes sure we are getting the correct response for our request
	state, err := generate.CryptoRandom(12)
	if err != nil {
		return "", err
	}

	authorizeURL := fmt.Sprintf("%s?redirect_uri=http://localhost:8301&client_id=%s&response_type=code&scope=%s&state=%s", provider.Endpoint().AuthURL, clientId, strings.Join(scopes, "+"), state)

	// the local server receives the response from the identity provider and sends it along to the infra server
	ls, err := newLocalServer()
	if err != nil {
		return "", err
	}

	err = browser.OpenURL(authorizeURL)
	if err != nil {
		return "", err
	}

	code, recvstate, err := ls.wait(time.Minute * 5)
	if err != nil {
		return "", err
	}

	if state != recvstate {
		//lint:ignore ST1005, user facing error
		return "", fmt.Errorf("Login aborted, provider state did not match the expected state")
	}

	return code, nil
}

// Given the provider name, directs user to its OIDC login page, then saves the auth code (to later login to infra)
func loginToProviderN(client *api.Client, providerName string) (*api.LoginRequestOIDC, error) {
	provider, err := GetProviderByName(client, providerName)
	if err != nil {
		return nil, err
	}
	return loginToProvider(provider)
}

// Given the provider, directs user to its OIDC login page, then saves the auth code (to later login to infra)
func loginToProvider(provider *api.Provider) (*api.LoginRequestOIDC, error) {
	fmt.Fprintf(os.Stderr, "  Logging in with %s...\n", termenv.String(provider.Name).Bold().String())

	code, err := oidcflow(provider.URL, provider.ClientID)
	if err != nil {
		return nil, err
	}

	return &api.LoginRequestOIDC{
		ProviderID:  provider.ID,
		RedirectURL: cliLoginRedirectURL,
		Code:        code,
	}, nil
}

func runSignupForLogin(cli *CLI, client *api.Client) (*api.LoginRequestPasswordCredentials, error) {
	fmt.Fprintln(cli.Stderr, "  Welcome to Infra. Set up your admin user:")

	var username string
	if err := survey.AskOne(
		&survey.Input{Message: "Username:"},
		&username,
		cli.surveyIO,
		survey.WithValidator(survey.Required),
	); err != nil {
		return nil, err
	}

	password, err := promptSetPassword(cli, "")
	if err != nil {
		return nil, err
	}

	_, err = client.Signup(&api.SignupRequest{Name: username, Password: password})
	if err != nil {
		return nil, err
	}

	return &api.LoginRequestPasswordCredentials{
		Name:     username,
		Password: password,
	}, nil
}

// Only used when logging in or switching to a new session, since user has no credentials. Otherwise, use defaultAPIClient().
func newAPIClient(cli *CLI, options loginCmdOptions) (*api.Client, error) {
	if !options.SkipTLSVerify {
		// Prompt user only if server fails the TLS verification
		if err := attemptTLSRequest(options.Server); err != nil {
			var uaErr x509.UnknownAuthorityError
			if !errors.As(err, &uaErr) {
				return nil, err
			}

			if options.NonInteractive {
				// TODO: add the --tls-ca flag
				// TODO: give a different error if the flag was set
				return nil, Error{
					Message: "The authenticity of the server could not be verified. " +
						"Use the --tls-ca flag to specify a trusted CA, or run " +
						"in interactive mode.",
				}
			}

			if err = promptVerifyTLSCert(cli, uaErr.Cert); err != nil {
				return nil, err
			}
			// TODO: save cert for future requests

		}
	}

	client, err := apiClient(options.Server, "", options.SkipTLSVerify)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func attemptTLSRequest(host string) error {
	// TODO: use apiClient here so that we set the right user agent, and can re-use
	// error handling from the client. Use the /api/version endpoint.
	reqURL, err := urlx.Parse(host)
	if err != nil {
		return fmt.Errorf("failed to parse the server hostname: %w", err)
	}
	reqURL.Scheme = "https"

	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	urlErr := &url.Error{}
	res, err := http.DefaultClient.Do(req)
	switch {
	case err == nil:
		res.Body.Close()
		return nil
	case errors.As(err, &urlErr):
		if urlErr.Timeout() {
			return fmt.Errorf("%w: %s", api.ErrTimeout, err)
		}
	}
	return err
}

func promptLocalLogin(cli *CLI) (*api.LoginRequestPasswordCredentials, error) {
	var credentials struct {
		Username string
		Password string
	}

	questionPrompt := []*survey.Question{
		{
			Name:     "Username",
			Prompt:   &survey.Input{Message: "Username:"},
			Validate: survey.Required,
		},
		{
			Name:     "Password",
			Prompt:   &survey.Password{Message: "Password:"},
			Validate: survey.Required,
		},
	}

	if err := survey.Ask(questionPrompt, &credentials, cli.surveyIO); err != nil {
		return &api.LoginRequestPasswordCredentials{}, err
	}

	return &api.LoginRequestPasswordCredentials{
		Name:     credentials.Username,
		Password: credentials.Password,
	}, nil
}

func promptAccessKeyLogin(cli *CLI) (string, error) {
	var accessKey string
	err := survey.AskOne(
		&survey.Password{Message: "Access Key:"},
		&accessKey,
		cli.surveyIO,
		survey.WithValidator(survey.Required),
	)
	return accessKey, err
}

func listProviders(client *api.Client) ([]api.Provider, error) {
	providers, err := client.ListProviders("")
	if err != nil {
		return nil, err
	}

	sort.Slice(providers.Items, func(i, j int) bool {
		return providers.Items[i].Name < providers.Items[j].Name
	})

	return providers.Items, nil
}

func promptLoginOptions(cli *CLI, client *api.Client) (loginMethod loginMethod, provider *api.Provider, err error) {
	providers, err := listProviders(client)
	if err != nil {
		return 0, nil, err
	}

	var options []string
	for _, p := range providers {
		options = append(options, fmt.Sprintf("%s (%s)", p.Name, p.URL))
	}

	options = append(options, "Login with username and password")
	options = append(options, "Login with an access key")

	var i int
	selectPrompt := &survey.Select{
		Message: "Select a login method:",
		Options: options,
	}
	err = survey.AskOne(selectPrompt, &i, cli.surveyIO)
	if errors.Is(err, terminal.InterruptErr) {
		return 0, nil, err
	}

	switch i {
	case len(options) - 1: // last option: accessKeyLogin
		return accessKeyLogin, nil, nil
	case len(options) - 2: // second last option: localLogin
		return localLogin, nil, nil
	default:
		return oidcLogin, &providers[i], nil
	}
}

func promptVerifyTLSCert(cli *CLI, cert *x509.Certificate) error {
	// TODO: improve this message.
	fmt.Fprintf(cli.Stderr, `The certificate presented by the server could not be automatically verified.

Subject: %[1]s
Issuer: %[2]s

Validity
  Not Before: %[3]v
  Not After: %[4]v

Subject Alternative Names:
  DNS Names: %[5]s
  IP Addresses: %[6]v

SHA-256 Fingerprint
  %[7]s

Compare the SHA-256 fingerprint against the one provided by your administrator
to manually verify the certificate can be trusted.
`,
		cert.Subject,
		cert.Issuer,
		cert.NotBefore.Format(time.RFC1123),
		cert.NotAfter.Format(time.RFC1123),
		strings.Join(cert.DNSNames, ", "),
		cert.IPAddresses, // TODO: format
		fingerprint(cert),
	)
	confirmPrompt := &survey.Select{
		Message: "Options:",
		Options: []string{
			"I do not trust this certificate",
			"Trust and save the certificate",
		},
	}
	var selection string
	if err := survey.AskOne(confirmPrompt, &selection, cli.surveyIO); err != nil {
		return err
	}
	switch {
	case selection == confirmPrompt.Options[0]:
		return terminal.InterruptErr
	case selection == confirmPrompt.Options[1]:
		return nil
	}
	// TODO: can this happen?
	panic("unexpected")
}

// TODO: move this to a shared place.
func fingerprint(cert *x509.Certificate) string {
	raw := sha256.Sum256(cert.Raw)
	return strings.Replace(fmt.Sprintf("% x", raw), " ", ":", -1)
}

// Returns the host address of the Infra server that user would like to log into
func promptServer(cli *CLI, options loginCmdOptions) (string, error) {
	if options.NonInteractive {
		return "", fmt.Errorf("Non-interactive login requires the [SERVER] argument")
	}

	config, err := readConfig()
	if err != nil {
		return "", err
	}

	servers := config.Hosts

	if len(servers) == 0 {
		return promptNewServer(cli)
	}

	return promptServerList(cli, servers)
}

func promptNewServer(cli *CLI) (string, error) {
	var server string
	err := survey.AskOne(
		&survey.Input{Message: "Server:"},
		&server,
		cli.surveyIO,
		survey.WithValidator(survey.Required),
	)
	return server, err
}

func promptServerList(cli *CLI, servers []ClientHostConfig) (string, error) {
	var promptOptions []string
	for _, server := range servers {
		promptOptions = append(promptOptions, server.Host)
	}

	defaultOption := "Connect to a new server"
	promptOptions = append(promptOptions, defaultOption)

	prompt := &survey.Select{
		Message: "Select a server:",
		Options: promptOptions,
	}

	filter := func(filterValue string, optValue string, optIndex int) bool {
		return strings.Contains(optValue, filterValue) || strings.EqualFold(optValue, defaultOption)
	}

	var i int
	if err := survey.AskOne(prompt, &i, survey.WithFilter(filter), cli.surveyIO); err != nil {
		return "", err
	}

	if promptOptions[i] == defaultOption {
		return promptNewServer(cli)
	}

	return servers[i].Host, nil
}
