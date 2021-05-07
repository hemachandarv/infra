package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/go-units"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Config struct {
	Host    string `json:"host"`
	User    string `json:"user"`
	Token   string `json:"token"`
	Expires int64  `json:"expires"`
}

func ReadConfig() (config *Config, err error) {
	config = &Config{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	contents, err := ioutil.ReadFile(filepath.Join(homeDir, ".infra", "config"))
	if os.IsNotExist(err) {
		return config, nil
	}

	if err != nil {
		return
	}

	if err = json.Unmarshal(contents, &config); err != nil {
		return
	}

	return
}

func WriteConfig(config *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Join(homeDir, ".infra"), os.ModePerm); err != nil {
		return err
	}

	contents, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(homeDir, ".infra", "config"), []byte(contents), 0644); err != nil {
		return err
	}

	return nil
}

func PrintTable(header []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t\t")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(data)
	table.Render()
}

func NewUnixHttpClient(path string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", path)
			},
		},
	}
}

func normalizeHost(host string) string {
	if host == "" {
		return "http://unix"
	}

	return host
}

type BasicAuthTransport struct {
	Username string
	Password string
}

func (bat BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", bat.Username, bat.Password)))))
	return http.DefaultTransport.RoundTrip(req)
}

func (bat *BasicAuthTransport) Client() *http.Client {
	return &http.Client{Transport: bat}
}

func CmdRun() error {
	config, err := ReadConfig()
	if err != nil {
		return err
	}

	host := config.Host
	token := config.Token

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	httpClient := &http.Client{}
	if config.Host == "" {
		httpClient = NewUnixHttpClient(filepath.Join(homeDir, ".infra", "infra.sock"))
	} else {
		bat := BasicAuthTransport{
			Username: token,
			Password: "",
		}
		httpClient = bat.Client()
	}

	app := &cli.App{
		Usage: "manage user & machine access to infrastructure",
		Commands: []*cli.Command{
			{
				Name:  "users",
				Usage: "Manage users",
				Subcommands: []*cli.Command{
					{
						Name:      "add",
						Usage:     "Add a new user",
						ArgsUsage: "EMAIL",
						Action: func(c *cli.Context) error {
							if c.NArg() <= 0 {
								cli.ShowCommandHelp(c, "add")
								return nil
							}

							email := c.Args().Get(0)

							form := url.Values{}
							form.Add("email", email)

							res, err := httpClient.PostForm(normalizeHost(host)+"/v1/users", form)
							if err != nil {
								log.Fatal(err)
							}

							type response struct {
								ID      string `json:"id"`
								Email   string `json:"email"`
								Created int64  `json:"created"`
								Updated int64  `json:"updated"`
								Error   string `json:"error"`
							}

							var decoded response
							if err = json.NewDecoder(res.Body).Decode(&decoded); err != nil {
								panic(err)
							}

							form = url.Values{}
							form.Add("user", decoded.ID)
							res, err = httpClient.PostForm(normalizeHost(host)+"/v1/tokens", form)
							if err != nil {
								log.Fatal(err)
							}

							if res.StatusCode != http.StatusCreated {
								log.Fatal(decoded.Error)
							}

							type tokenResponse struct {
								Token
								SecretToken string `json:"secret_token"`
								Host        string `json:"host"`
								Error       string
							}

							var decodedTokenResponse tokenResponse
							if err = json.NewDecoder(res.Body).Decode(&decodedTokenResponse); err != nil {
								panic(err)
							}

							fmt.Println()
							fmt.Println("User " + decoded.Email + " added. Please share the following command with them so they can log in:")
							fmt.Println()
							if decodedTokenResponse.Host == "" {
								fmt.Println("infra login --token " + decodedTokenResponse.SecretToken)
							} else {
								fmt.Println("infra login --token " + decodedTokenResponse.SecretToken + " " + decodedTokenResponse.Host)
							}
							fmt.Println()

							return nil
						},
					},
					{
						Name:    "list",
						Usage:   "list users",
						Aliases: []string{"ls"},
						Action: func(c *cli.Context) error {
							res, err := httpClient.Get(normalizeHost(host) + "/v1/users")
							if err != nil {
								return err
							}

							type response struct {
								Data  []User
								Error string `json:"error"`
							}

							var decoded response
							if err = json.NewDecoder(res.Body).Decode(&decoded); err != nil {
								return err
							}

							if decoded.Error != "" {
								return err
							}

							rows := [][]string{}
							for _, user := range decoded.Data {
								createdAt := time.Unix(user.Created, 0)
								rows = append(rows, []string{user.ID, "infra", user.Email, units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago"})
							}

							PrintTable([]string{"USER ID", "PROVIDER", "EMAIL", "CREATED"}, rows)

							return nil
						},
					},
					{
						Name:  "delete",
						Usage: "delete a user",
						Action: func(c *cli.Context) error {
							user := c.Args().Get(0)
							req, err := http.NewRequest(http.MethodDelete, normalizeHost(host)+"/v1/users/"+user, nil)
							if err != nil {
								log.Fatal(err)
							}

							res, err := httpClient.Do(req)
							if err != nil {
								log.Fatal(err)
							}
							defer res.Body.Close()

							type response struct {
								ID      string `json:"id"`
								Deleted bool   `json:"deleted"`
								Error   string `json:"error"`
							}

							var decoded response
							if err = json.NewDecoder(res.Body).Decode(&decoded); err != nil {
								return err
							}

							if decoded.Deleted {
								fmt.Println("User deleted")
							} else if len(decoded.Error) > 0 {
								return errors.New(decoded.Error)
							} else {
								return errors.New("could not delete user")
							}

							return nil
						},
					},
				},
			},
			{
				Name:      "login",
				Usage:     "Login to an Infra Engine",
				ArgsUsage: "HOST",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "token",
						Aliases: []string{"t"},
						Usage:   "token to authenticate with",
					},
				},
				Action: func(c *cli.Context) error {
					hostArg := c.Args().First()
					client := &http.Client{}

					if hostArg == "" {
						hostArg = host
						client = httpClient
					}

					// Get token from
					token := c.String("token")

					req, err := http.NewRequest("POST", normalizeHost(hostArg)+"/v1/tokens", nil)
					if err != nil {
						return err
					}

					req.SetBasicAuth(token, "")

					res, err := client.Do(req)
					if err != nil {
						return err
					}

					body, err := ioutil.ReadAll(res.Body)
					if err != nil {
						return err
					}

					type tokenResponse struct {
						Token struct {
							ID      string `json:"id"`
							User    string `json:"user"`
							Created int64  `json:"created"`
							Updated int64  `json:"updated"`
							Expires int64  `json:"expires"`
						}
						SecretToken string `json:"secret_token"`
						Host        string `json:"host"`
						Error       string
					}

					var response tokenResponse
					if err = json.Unmarshal(body, &response); err != nil {
						return cli.Exit(err, 1)
					}

					if res.StatusCode != http.StatusCreated {
						return cli.Exit(response.Error, 1)
					}

					if err = WriteConfig(&Config{
						Host:    host,
						Token:   response.SecretToken,
						Expires: response.Token.Expires,
						User:    response.Token.User,
					}); err != nil {
						fmt.Println(err)
						return err
					}

					// Generate new kubeconfig entry
					config := clientcmdapi.NewConfig()
					config.Clusters["infra"] = &clientcmdapi.Cluster{
						Server: host + "/v1/proxy",
					}
					config.AuthInfos["infra"] = &clientcmdapi.AuthInfo{
						Token: response.SecretToken,
					}
					config.Contexts["infra"] = &clientcmdapi.Context{
						Cluster:  "infra",
						AuthInfo: "infra",
					}

					tempFile, _ := ioutil.TempFile("", "")
					defer os.Remove(tempFile.Name())
					config.CurrentContext = "infra" // TODO: should we do this?
					if err = clientcmd.WriteToFile(*config, tempFile.Name()); err != nil {
						log.Fatal(err)
					}

					// Load default config and merge new config in
					loadingRules := clientcmd.ClientConfigLoadingRules{Precedence: []string{tempFile.Name(), clientcmd.RecommendedHomeFile}}
					mergedConfig, err := loadingRules.Load()
					if err != nil {
						log.Fatal(err)
					}

					if err = clientcmd.WriteToFile(*mergedConfig, clientcmd.RecommendedHomeFile); err != nil {
						log.Fatal(err)
					}

					fmt.Println("Kubeconfig updated.")

					return nil
				},
			},
			{
				Name:  "start",
				Usage: "Start the Infra Engine",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "db-path",
						Usage: "Path to database",
					},
				},
				Action: func(c *cli.Context) error {
					return ServerRun(&ServerOptions{
						DBPath: c.String("db-path"),
					})
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		return err
	}

	return nil
}