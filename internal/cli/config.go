// Package cli implements argument parsing, environment-variable fallback,
// and structured output for the atlas-ap-remote command-line client.
package cli

import (
	"errors"
	"fmt"
	"strings"
)

// Config captures the resolved connection settings for a CLI invocation.
type Config struct {
	// Server is the base URL of the Atlas AP Remote service, with no
	// trailing slash.
	Server string
	// Token is the bearer token, or empty for unauthenticated requests.
	Token string
}

// ErrMissingServer is returned by ResolveConfig when neither the
// --server flag nor ATLAS_REMOTE_URL provides a server URL.
var ErrMissingServer = errors.New("missing server URL; pass --server or set ATLAS_REMOTE_URL")

// ResolveConfig determines the effective server URL and bearer token using
// explicit flags first, then environment variables. The supplied environ is
// expected to be a list of "KEY=VALUE" pairs (typically os.Environ()).
//
// The server URL has any single trailing "/" stripped. The token has only
// surrounding ASCII whitespace stripped; internal whitespace is preserved
// to avoid mutating user-provided secrets.
func ResolveConfig(serverFlag, tokenFlag string, environ []string) (Config, error) {
	server := strings.TrimRight(strings.TrimSpace(serverFlag), "/")
	if server == "" {
		server = strings.TrimRight(strings.TrimSpace(envValue(environ, "ATLAS_REMOTE_URL")), "/")
	}

	token := strings.TrimSpace(tokenFlag)
	if token == "" {
		token = strings.TrimSpace(envValue(environ, "ATLAS_REMOTE_TOKEN"))
	}

	if server == "" {
		return Config{}, fmt.Errorf("%w", ErrMissingServer)
	}

	return Config{Server: server, Token: token}, nil
}

func envValue(environ []string, key string) string {
	prefix := key + "="
	for _, kv := range environ {
		if strings.HasPrefix(kv, prefix) {
			return kv[len(prefix):]
		}
	}
	return ""
}