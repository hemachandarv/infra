package registry

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/infrahq/infra/internal/generate"
	"github.com/infrahq/infra/internal/registry/mocks"
	v1 "github.com/infrahq/infra/internal/v1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func TestAuthInterceptorPublic(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/Status",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, err = authInterceptor(db)(context.Background(), "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.OK)
}

func TestAuthInterceptorDefaultUnauthenticated(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	ctx := context.Background()

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorNoAuthorizationMetadata(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "random", "metadata")

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorEmptyAuthorization(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "")

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorWrongAuthorizationFormat(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "hello")

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorWrongBearerAuthorizationFormat(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer hello")

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorInvalidToken(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer "+generate.RandString(TOKEN_LEN))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func addUser(db *gorm.DB, email string, password string, admin bool) (tokenId string, tokenSecret string, err error) {
	var token Token
	var secret string
	err = db.Transaction(func(tx *gorm.DB) error {
		var infraSource Source
		if err := tx.Where(&Source{Type: SOURCE_TYPE_INFRA}).First(&infraSource).Error; err != nil {
			return err
		}
		var user User

		err := infraSource.CreateUser(tx, &user, email, password, admin)
		if err != nil {
			return err
		}

		secret, err = NewToken(tx, user.Id, &token)
		if err != nil {
			return errors.New("could not create token")
		}

		return nil
	})
	if err != nil {
		return "", "", err
	}

	return token.Id, secret, nil
}

func TestAuthInterceptorValidToken(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListUsers",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	id, secret, err := addUser(db, "test@test.com", "passw0rd", false)
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+id+secret))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.OK)
}

func TestAuthInterceptorAdmin(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/CreateUser",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	id, secret, err := addUser(db, "test@test.com", "passw0rd", false)
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+id+secret))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorAdminPass(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/CreateUser",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	id, secret, err := addUser(db, "test@test.com", "passw0rd", true)
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+id+secret))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.OK)
}

func TestAuthInterceptorInvalidApiKey(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListPermissions",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+generate.RandString(API_KEY_LEN)))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestAuthInterceptorValidApiKey(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/ListPermissions",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	var apiKey ApiKey
	err = db.FirstOrCreate(&apiKey, &ApiKey{Name: "default"}).Error
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+apiKey.Key))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.OK)
}

func TestAuthInterceptorValidApiKeyInvalidMethod(t *testing.T) {
	unaryInfo := &grpc.UnaryServerInfo{
		FullMethod: "/v1.V1/CreateUser",
	}
	unaryHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	var apiKey ApiKey
	err = db.FirstOrCreate(&apiKey, &ApiKey{Name: "default"}).Error
	if err != nil {
		t.Fatal(err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+apiKey.Key))

	_, err = authInterceptor(db)(ctx, "req", unaryInfo, unaryHandler)
	assert.Equal(t, status.Code(err), codes.Unauthenticated)
}

func TestLoginMethodEmptyRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodNilInfraRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_INFRA,
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodEmptyInfraRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type:  v1.SourceType_INFRA,
		Infra: &v1.LoginRequest_Infra{},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodInfraEmptyPassword(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_INFRA,
		Infra: &v1.LoginRequest_Infra{
			Email: "test@test.com",
		},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodInfraEmptyEmail(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_INFRA,
		Infra: &v1.LoginRequest_Infra{
			Password: "passw0rd",
		},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodInfraSuccess(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = addUser(db, "test@test.com", "passw0rd", false)
	if err != nil {
		t.Fatal(err)
	}

	server := &V1Server{db: db}

	req := &v1.LoginRequest{
		Type: v1.SourceType_INFRA,
		Infra: &v1.LoginRequest_Infra{
			Email:    "test@test.com",
			Password: "passw0rd",
		},
	}

	res, err := server.Login(context.Background(), req)
	assert.Equal(t, status.Code(err), codes.OK)
	assert.NotEqual(t, res.Token, "")
}

func TestLoginMethodNilOktaRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_OKTA,
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodEmptyOktaRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_OKTA,
		Okta: &v1.LoginRequest_Okta{},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodOktaMissingDomainRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_OKTA,
		Okta: &v1.LoginRequest_Okta{
			Code: "code",
		},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodOktaMissingCodeRequest(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	req := &v1.LoginRequest{
		Type: v1.SourceType_OKTA,
		Okta: &v1.LoginRequest_Okta{
			Domain: "testing.okta.com",
		},
	}

	server := &V1Server{db: db}

	_, err = server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

func TestLoginMethodOkta(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	var source Source
	source.Type = "okta"
	source.OktaApiToken = "test-api-token"
	source.OktaDomain = "test.okta.com"
	source.OktaClientId = "test-client-id"
	source.OktaClientSecret = "test-client-secret"
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}

	var user User
	source.CreateUser(db, &user, "test@test.com", "", false)
	if err != nil {
		t.Fatal(err)
	}

	testOkta := new(mocks.Okta)
	testOkta.On("EmailFromCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("test@test.com", nil)

	server := &V1Server{db: db, okta: testOkta}

	req := &v1.LoginRequest{
		Type: v1.SourceType_OKTA,
		Okta: &v1.LoginRequest_Okta{
			Domain: "test.okta.com",
			Code:   "testcode",
		},
	}

	res, err := server.Login(context.Background(), req)

	assert.Equal(t, status.Code(err), codes.OK)
	assert.NotEqual(t, res.Token, "")
}

func TestSignup(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Fatal(err)
	}

	server := &V1Server{db: db}

	req := &v1.SignupRequest{
		Email:    "test@test.com",
		Password: "passw0rd",
	}

	res, err := server.Signup(context.Background(), req)
	assert.Equal(t, status.Code(err), codes.OK)
	assert.NotEqual(t, res.Token, "")

	var user User
	err = db.First(&user).Error
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, user.Admin, true)
	assert.Equal(t, user.Email, "test@test.com")
}

func TestSignupWithExistingAdmin(t *testing.T) {
	db, err := NewDB("file::memory:")
	if err != nil {
		t.Error(err)
	}

	addUser(db, "existing@user.com", "passw0rd", true)

	server := &V1Server{db: db}

	req := &v1.SignupRequest{
		Email:    "admin@test.com",
		Password: "adminpassw0rd",
	}

	res, err := server.Signup(context.Background(), req)
	assert.Equal(t, status.Code(err), codes.InvalidArgument)
	assert.Equal(t, res, nil)
}