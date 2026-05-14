package apiclient

import (
	"fmt"
	"net/url"

	"github.com/go-openapi/strfmt"

	gen "github.com/container-registry/harbor-satellite/ground-control/gcctl/apiclient/generated/client"
)

// New returns a Ground Control client targeted at serverURL.
//
// serverURL must include scheme and host (e.g. "http://localhost:8080"). Any
// path component is used as the API base path; an empty path falls back to
// the generated client's DefaultBasePath.
func New(serverURL string) (*gen.Groundcontrol, error) {
	cfg, err := transportConfig(serverURL)
	if err != nil {
		return nil, err
	}
	return gen.NewHTTPClientWithConfig(strfmt.Default, cfg), nil
}

func transportConfig(serverURL string) (*gen.TransportConfig, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("parse server URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("server URL must include scheme and host, got %q", serverURL)
	}
	basePath := u.Path
	if basePath == "" {
		basePath = gen.DefaultBasePath
	}
	return gen.DefaultTransportConfig().
		WithHost(u.Host).
		WithBasePath(basePath).
		WithSchemes([]string{u.Scheme}), nil
}
