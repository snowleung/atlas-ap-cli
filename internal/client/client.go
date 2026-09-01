// Package client implements the HTTP protocol for the Atlas AP Remote
// service. The CLI layer composes this package; nothing in this file knows
// about flags, environment variables, or human output.
package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SubmitRequest carries the multipart form fields for POST /jobs.
//
// IdempotencyKey MUST be a non-empty UUID-shaped string. Callers should
// generate it with NewUUIDv4 for each attempt.
type SubmitRequest struct {
	FilePath       string
	CosmeticType   string
	BodyParts      string
	ProductName    string
	UsageMethod    string
	IdempotencyKey string
}

// SubmitResponse is the parsed body of a successful submit.
type SubmitResponse struct {
	JobID string `json:"job_id"`
}

// StatusResponse mirrors the JSON returned by GET /jobs/{id}.
type StatusResponse struct {
	JobID        string `json:"job_id"`
	Status       string `json:"status"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// HealthResponse is the liveness response returned by GET /health.
type HealthResponse struct {
	Status string `json:"status"`
}

// CancelResponse mirrors the JSON returned by POST /jobs/{id}/cancel.
type CancelResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// Client is the high-level HTTP client. The zero value is not usable;
// use New.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// New constructs a Client with a 30-second per-request timeout and the
// given base URL (without trailing slash) and bearer token. An empty token
// suppresses the Authorization header.
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Submit performs POST /jobs. It opens req.FilePath, streams its contents
// as the "file" multipart field, and applies Authorization when a token
// is configured.
func (c *Client) Submit(ctx context.Context, req SubmitRequest) (*SubmitResponse, error) {
	if req.FilePath == "" {
		return nil, fmt.Errorf("submit: file path required")
	}
	if req.IdempotencyKey == "" {
		return nil, fmt.Errorf("submit: idempotency key required")
	}

	f, err := os.Open(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("submit: open %s: %w", req.FilePath, err)
	}
	defer f.Close()

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)

	fields := map[string]string{
		"cosmetic_type":   req.CosmeticType,
		"body_parts":      req.BodyParts,
		"product_name":    req.ProductName,
		"usage_method":    req.UsageMethod,
		"idempotency_key": req.IdempotencyKey,
	}
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("submit: form field %s: %w", k, err)
		}
	}

	fw, err := mw.CreateFormFile("file", filepath.Base(req.FilePath))
	if err != nil {
		return nil, fmt.Errorf("submit: create file field: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return nil, fmt.Errorf("submit: copy file: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("submit: close multipart: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/jobs", body)
	if err != nil {
		return nil, fmt.Errorf("submit: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	c.applyAuth(httpReq)

	var resp SubmitResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadDataFile performs a multipart POST to one of the /data-files
// endpoints. It opens filePath, streams its contents as the "file" part
// using the file's basename, applies bearer auth, and decodes the
// successful JSON response into an arbitrary object. The caller owns the
// command-to-endpoint mapping; this method owns the HTTP protocol details.
func (c *Client) UploadDataFile(ctx context.Context, endpoint, filePath string) (map[string]any, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("data-file: endpoint required")
	}
	if filePath == "" {
		return nil, fmt.Errorf("data-file: file path required")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("data-file: open %s: %w", filePath, err)
	}
	defer f.Close()

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)

	fw, err := mw.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("data-file: create file field: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return nil, fmt.Errorf("data-file: copy file: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("data-file: close multipart: %w", err)
	}

	u, err := c.buildURL(endpoint)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return nil, fmt.Errorf("data-file: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	c.applyAuth(httpReq)

	var resp map[string]any
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Status performs GET /jobs/{id}.
func (c *Client) Status(ctx context.Context, jobID string) (*StatusResponse, error) {
	if jobID == "" {
		return nil, fmt.Errorf("status: job id required")
	}
	u, err := c.buildURL("/jobs/" + url.PathEscape(jobID))
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("status: build request: %w", err)
	}
	c.applyAuth(httpReq)

	var resp StatusResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Health performs GET /health.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	u, err := c.buildURL("/health")
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("health: build request: %w", err)
	}
	c.applyAuth(httpReq)

	var resp HealthResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Cancel performs POST /jobs/{id}/cancel.
func (c *Client) Cancel(ctx context.Context, jobID string) (*CancelResponse, error) {
	if jobID == "" {
		return nil, fmt.Errorf("cancel: job id required")
	}
	u, err := c.buildURL("/jobs/" + url.PathEscape(jobID) + "/cancel")
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return nil, fmt.Errorf("cancel: build request: %w", err)
	}
	c.applyAuth(httpReq)

	var resp CancelResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Download performs GET /jobs/{id}/download and returns a streaming body.
// The caller MUST Close the returned ReadCloser. A non-nil error means the
// response is nil or already closed.
func (c *Client) Download(ctx context.Context, jobID string) (io.ReadCloser, error) {
	if jobID == "" {
		return nil, fmt.Errorf("download: job id required")
	}
	u, err := c.buildURL("/jobs/" + url.PathEscape(jobID) + "/download")
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("download: build request: %w", err)
	}
	c.applyAuth(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, c.classifyTransport(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, decodeServiceError(resp)
	}
	return resp.Body, nil
}

func (c *Client) buildURL(path string) (string, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", fmt.Errorf("client: invalid base URL: %w", err)
	}
	ref, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("client: invalid path: %w", err)
	}
	return base.ResolveReference(ref).String(), nil
}

func (c *Client) applyAuth(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return c.classifyTransport(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeServiceError(resp)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("client: decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) classifyTransport(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("%w: %v", ErrTimeout, err)
	}
	return fmt.Errorf("%w: %v", ErrNetwork, err)
}

// decodeServiceError parses the {code,message} envelope from a non-2xx
// response. If the body is not JSON, it returns a ServiceError with the
// default INTERNAL_ERROR code. Headers are never exposed to the caller.
func decodeServiceError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	env := struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{}
	if err := json.Unmarshal(body, &env); err != nil || env.Code == "" {
		return &ServiceError{
			Code:       "INTERNAL_ERROR",
			Message:    "service returned non-JSON error",
			HTTPStatus: resp.StatusCode,
		}
	}
	return &ServiceError{
		Code:       env.Code,
		Message:    env.Message,
		HTTPStatus: resp.StatusCode,
	}
}

// NewUUIDv4 generates a random UUID version 4 using crypto/rand. The
// returned string is lowercase and 36 characters long, including hyphens.
func NewUUIDv4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("uuid: random read: %w", err)
	}
	// RFC 4122 version 4 / variant 1
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	hexStr := hex.EncodeToString(b[:])
	return hexStr[0:8] + "-" + hexStr[8:12] + "-" + hexStr[12:16] + "-" + hexStr[16:20] + "-" + hexStr[20:32], nil
}
