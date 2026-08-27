package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// recordedRequest captures what the test server observed so individual
// assertions can target specific fields.
type recordedRequest struct {
	method        string
	path          string
	contentType   string
	authorization string
	formFields    map[string]string
	fileContent    []byte
}

func newRecordingServer(handler func(*recordedRequest)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &recordedRequest{
			method:        r.Method,
			path:          r.URL.Path,
			authorization: r.Header.Get("Authorization"),
			formFields:    map[string]string{},
		}

		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "multipart/form-data") {
			rec.contentType = ct
			if err := r.ParseMultipartForm(1 << 20); err == nil {
				for k, v := range r.MultipartForm.Value {
					if len(v) > 0 {
						rec.formFields[k] = v[0]
					}
				}
				if r.MultipartForm.File != nil {
					for _, files := range r.MultipartForm.File {
						for _, fh := range files {
							f, err := fh.Open()
							if err != nil {
								http.Error(w, err.Error(), 500)
								return
							}
							data, _ := io.ReadAll(f)
							rec.fileContent = data
							f.Close()
						}
					}
				}
			}
		}

		handler(rec)

		// Default response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
}

func TestSubmit_SendsMultipartToJobs(t *testing.T) {
	var rec recordedRequest
	rec.formFields = map[string]string{}
	srv := newRecordingServer(func(r *recordedRequest) {
		rec = *r
	})
	defer srv.Close()

	c := New(srv.URL, "")
	resp, err := c.Submit(context.Background(), SubmitRequest{
		FilePath:       writeTempFile(t, "hello world"),
		CosmeticType:   "驻留",
		BodyParts:      "全身",
		ProductName:    "面霜",
		UsageMethod:    "daily",
		IdempotencyKey: "11111111-1111-4111-8111-111111111111",
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if rec.method != http.MethodPost {
		t.Errorf("expected POST, got %s", rec.method)
	}
	if rec.path != "/jobs" {
		t.Errorf("expected /jobs, got %s", rec.path)
	}
	if !strings.HasPrefix(rec.contentType, "multipart/form-data") {
		t.Errorf("expected multipart Content-Type, got %q", rec.contentType)
	}
	if rec.formFields["cosmetic_type"] != "驻留" {
		t.Errorf("cosmetic_type mismatch: %q", rec.formFields["cosmetic_type"])
	}
	if rec.formFields["body_parts"] != "全身" {
		t.Errorf("body_parts mismatch: %q", rec.formFields["body_parts"])
	}
	if rec.formFields["product_name"] != "面霜" {
		t.Errorf("product_name mismatch: %q", rec.formFields["product_name"])
	}
	if rec.formFields["usage_method"] != "daily" {
		t.Errorf("usage_method mismatch: %q", rec.formFields["usage_method"])
	}
	if rec.formFields["idempotency_key"] != "11111111-1111-4111-8111-111111111111" {
		t.Errorf("idempotency_key mismatch: %q", rec.formFields["idempotency_key"])
	}
	if string(rec.fileContent) != "hello world" {
		t.Errorf("file payload mismatch: %q", string(rec.fileContent))
	}
	if resp == nil {
		t.Fatal("response is nil")
	}
}

func TestSubmit_GeneratesUUIDIdempotencyKey(t *testing.T) {
	srv := newRecordingServer(func(r *recordedRequest) {})
	defer srv.Close()

	key, err := NewUUIDv4()
	if err != nil {
		t.Fatalf("uuid failed: %v", err)
	}
	if !looksLikeUUIDv4(key) {
		t.Errorf("key %q does not look like a UUIDv4", key)
	}

	if _, err := New(srv.URL, "").Submit(context.Background(), SubmitRequest{
		FilePath:       writeTempFile(t, "x"),
		CosmeticType:   "驻留",
		BodyParts:      "全身",
		IdempotencyKey: key,
	}); err != nil {
		t.Fatalf("submit failed: %v", err)
	}
}

func TestSubmit_NoAuthHeaderWithoutToken(t *testing.T) {
	var rec recordedRequest
	rec.formFields = map[string]string{}
	srv := newRecordingServer(func(r *recordedRequest) { rec = *r })
	defer srv.Close()

	if _, err := New(srv.URL, "").Submit(context.Background(), SubmitRequest{
		FilePath:       writeTempFile(t, "x"),
		CosmeticType:   "驻留",
		BodyParts:      "全身",
		IdempotencyKey: "22222222-2222-4222-8222-222222222222",
	}); err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if rec.authorization != "" {
		t.Errorf("expected no Authorization header, got %q", rec.authorization)
	}
}

func TestSubmit_AddsBearerWhenTokenSet(t *testing.T) {
	var rec recordedRequest
	rec.formFields = map[string]string{}
	srv := newRecordingServer(func(r *recordedRequest) { rec = *r })
	defer srv.Close()

	if _, err := New(srv.URL, "my-token").Submit(context.Background(), SubmitRequest{
		FilePath:       writeTempFile(t, "x"),
		CosmeticType:   "驻留",
		BodyParts:      "全身",
		IdempotencyKey: "33333333-3333-4333-8333-333333333333",
	}); err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if rec.authorization != "Bearer my-token" {
		t.Errorf("expected Bearer my-token, got %q", rec.authorization)
	}
}

func TestStatus_GETJobByID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/jobs/job-123" {
			t.Errorf("expected /jobs/job-123, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(StatusResponse{JobID: "job-123", Status: "succeeded"})
	}))
	defer srv.Close()

	resp, err := New(srv.URL, "").Status(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if resp.JobID != "job-123" || resp.Status != "succeeded" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestCancel_PostJobCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/jobs/job-456/cancel" {
			t.Errorf("expected /jobs/job-456/cancel, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(CancelResponse{JobID: "job-456", Status: "cancelled"})
	}))
	defer srv.Close()

	resp, err := New(srv.URL, "").Cancel(context.Background(), "job-456")
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if resp.JobID != "job-456" || resp.Status != "cancelled" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestServiceError_Preserved(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"BAD_INPUT","message":"missing field"}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL, "").Status(context.Background(), "job-789")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var se *ServiceError
	if !errors.As(err, &se) {
		t.Fatalf("expected *ServiceError, got %T", err)
	}
	if se.Code != "BAD_INPUT" {
		t.Errorf("code mismatch: %q", se.Code)
	}
	if se.Message != "missing field" {
		t.Errorf("message mismatch: %q", se.Message)
	}
	if se.HTTPStatus != http.StatusBadRequest {
		t.Errorf("http status mismatch: %d", se.HTTPStatus)
	}
}

func TestServiceError_InvalidJSONMapsToNetwork(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("oops"))
	}))
	defer srv.Close()

	_, err := New(srv.URL, "").Status(context.Background(), "job-101")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrNetwork) {
		t.Fatalf("expected non-network error for invalid JSON, got %v", err)
	}
	var se *ServiceError
	if !errors.As(err, &se) {
		t.Fatalf("expected *ServiceError, got %T", err)
	}
	if se.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("http status mismatch: %d", se.HTTPStatus)
	}
	if se.Code != "INTERNAL_ERROR" {
		t.Errorf("default code should be INTERNAL_ERROR, got %q", se.Code)
	}
}

func TestNetworkError_MapsToNetwork(t *testing.T) {
	c := New("http://127.0.0.1:1", "") // closed port
	c.HTTPClient = &http.Client{Timeout: 500 * time.Millisecond}
	_, err := c.Status(context.Background(), "any")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNetwork) {
		t.Fatalf("expected ErrNetwork, got %v", err)
	}
}

func TestTimeout_MapsToTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	c.HTTPClient = &http.Client{Timeout: 50 * time.Millisecond}
	_, err := c.Status(context.Background(), "any")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
}

func TestDownload_StreamsBody(t *testing.T) {
	payload := []byte("zip-content-here")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/jobs/job-d/download" {
			t.Errorf("expected /jobs/job-d/download, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	body, err := c.Download(context.Background(), "job-d")
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	defer body.Close()
	data, _ := io.ReadAll(body)
	if string(data) != string(payload) {
		t.Errorf("payload mismatch: %q", string(data))
	}
}

func TestDownload_ServiceError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND","message":"no such job"}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL, "").Download(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	var se *ServiceError
	if !errors.As(err, &se) {
		t.Fatalf("expected *ServiceError, got %T", err)
	}
	if se.Code != "NOT_FOUND" {
		t.Errorf("code mismatch: %q", se.Code)
	}
}

// helper funcs

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "submit-*")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return f.Name()
}

func looksLikeUUIDv4(s string) bool {
	// 8-4-4-4-12 hex, version nibble 4 in 13th position
	if len(s) != 36 {
		return false
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	if s[14] != '4' {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// ensure multipart import is used (the recorder always parses multipart)
var _ = multipart.ErrMessageTooLarge