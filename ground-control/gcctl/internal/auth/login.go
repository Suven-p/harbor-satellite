package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"

	authclient "github.com/container-registry/harbor-satellite/ground-control/gcctl/apiclient/generated/client/auth"
	"github.com/container-registry/harbor-satellite/ground-control/gcctl/apiclient/generated/models"
	"github.com/container-registry/harbor-satellite/ground-control/gcctl/internal/apiclient"
)

// Result is what a successful login returns to the caller.
type Result struct {
	Token     string
	ExpiresAt time.Time
}

// Login authenticates against Ground Control at serverURL using the given
// credentials and returns the issued session token.
//
// serverURL must include a scheme and host (e.g. "http://localhost:8080");
// any path component is used as the API base path.
func Login(ctx context.Context, serverURL, username, password string) (*Result, error) {
	if username == "" {
		return nil, errors.New("username is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}

	gc, err := apiclient.New(serverURL)
	if err != nil {
		return nil, err
	}

	pw := strfmt.Password(password)
	params := authclient.NewLoginParamsWithContext(ctx).WithCredentials(&models.LoginRequest{
		Username: &username,
		Password: &pw,
	})

	ok, err := gc.Auth.Login(params)
	if err != nil {
		return nil, translateLoginError(err)
	}
	if ok.Payload == nil || ok.Payload.Token == nil || ok.Payload.ExpiresAt == nil {
		return nil, errors.New("login response missing token or expires_at")
	}

	return &Result{
		Token:     *ok.Payload.Token,
		ExpiresAt: time.Time(*ok.Payload.ExpiresAt),
	}, nil
}

func translateLoginError(err error) error {
	if _, ok := errors.AsType[*authclient.LoginUnauthorized](err); ok {
		return errors.New("invalid username or password")
	}
	if badRequest, ok := errors.AsType[*authclient.LoginBadRequest](err); ok {
		return fmt.Errorf("invalid login request: %s", errorBody(badRequest.Payload))
	}
	if serverErr, ok := errors.AsType[*authclient.LoginInternalServerError](err); ok {
		return fmt.Errorf("ground control returned 500: %s", errorBody(serverErr.Payload))
	}
	return err
}

func errorBody(e *models.ErrorResponse) string {
	if e == nil || e.Error == nil {
		return "no error message in response"
	}
	return *e.Error
}
