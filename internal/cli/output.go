package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

// Version is the build-time version string. It is overridden via -ldflags
// "-X github.com/atlas-ap/atlas-ap-remote/internal/cli.Version=vX.Y.Z".
var Version = "dev"

// ErrorEnvelope is the JSON shape used for both successful `--json` outputs
// (with Success=true) and failure outputs (Success=false).
type ErrorEnvelope struct {
	Success    bool   `json:"success"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"http_status,omitempty"`
}

// WriteErrorJSON emits a single-line JSON error envelope followed by a
// trailing newline. Callers MUST NOT pass bearer tokens inside the message;
// this helper does not inspect or redact input but the contract guarantees
// that no token material is ever included in the structured fields.
func WriteErrorJSON(w io.Writer, code, message string, httpStatus int) error {
	env := ErrorEnvelope{
		Success:    false,
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal error envelope: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

// WriteErrorHuman writes a single-line, human-readable error to the given
// writer. The format is "<code>: <message>\n".
func WriteErrorHuman(w io.Writer, code, message string) error {
	if _, err := fmt.Fprintf(w, "%s: %s\n", code, message); err != nil {
		return err
	}
	return nil
}