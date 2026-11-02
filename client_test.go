package fincode

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type recordedRequest struct {
	method string
	path   string
	query  string
	auth   string
	body   string
}

func newTestClient(t *testing.T, handler http.Handler, opts ...Option) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	all := append([]Option{WithBaseURL(server.URL + "/v1")}, opts...)
	return New("sk_test", all...)
}

func TestSuccessEnvelopeAndAuth(t *testing.T) {
	var recorded recordedRequest
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded = recordedRequest{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.RawQuery,
			auth:   r.Header.Get("Authorization"),
		}
		_, _ = w.Write([]byte(`{"data":{"swift_code":"ICBKCNBJ","bic":"ICBKCNBJ","bank":{"name_en":"ICBC"}},"meta":{"request_id":"req_1"}}`))
	}))

	result, err := client.Swift.Get(context.Background(), "ICBKCNBJ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.SwiftCode != "ICBKCNBJ" {
		t.Fatalf("swift_code = %q", result.Data.SwiftCode)
	}
	if result.Data.Bank == nil || result.Data.Bank.NameEN != "ICBC" {
		t.Fatalf("bank = %+v", result.Data.Bank)
	}
	if result.Meta.RequestID != "req_1" {
		t.Fatalf("request_id = %q", result.Meta.RequestID)
	}
	if result.Meta.NextCursor != nil {
		t.Fatalf("next_cursor = %v", *result.Meta.NextCursor)
	}
	if recorded.auth != "Bearer sk_test" {
		t.Fatalf("authorization = %q", recorded.auth)
	}
	if recorded.path != "/v1/swift/ICBKCNBJ" {
		t.Fatalf("path = %q", recorded.path)
	}
}

func TestQueryEncodingAndNullOmission(t *testing.T) {
	var recorded recordedRequest
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded = recordedRequest{path: r.URL.Path, query: r.URL.RawQuery}
		_, _ = w.Write([]byte(`{"data":[],"meta":{"request_id":"req_2","next_cursor":null}}`))
	}))

	if _, err := client.Banks.List(context.Background(), ListOptions{Country: "US", Limit: 5}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	values, err := url.ParseQuery(recorded.query)
	if err != nil {
		t.Fatalf("query parse: %v", err)
	}
	if values.Get("country") != "US" || values.Get("limit") != "5" {
		t.Fatalf("query = %q", recorded.query)
	}
	if values.Has("cursor") {
		t.Fatalf("cursor should be omitted, query = %q", recorded.query)
	}
	if recorded.path != "/v1/banks" {
		t.Fatalf("path = %q", recorded.path)
	}
}

func TestAPIErrorMapping(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"INVALID_SWIFT","message":"bad bic","request_id":"req_5"}}`))
	}))

	_, err := client.Swift.Get(context.Background(), "NOPE")
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if apiErr.Code != "INVALID_SWIFT" || apiErr.Status != 422 || apiErr.RequestID != "req_5" {
		t.Fatalf("error = %+v", apiErr)
	}
	if apiErr.IsRetryable() {
		t.Fatal("validation errors must not be retryable")
	}
	if !strings.Contains(err.Error(), "req_5") {
		t.Fatalf("error message should quote the request id: %v", err)
	}
}

func TestRetriesRateLimitThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"RATE_LIMITED","message":"slow","request_id":"req_2"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"swift_code":"FLAKY"},"meta":{"request_id":"req_3"}}`))
	}))

	result, err := client.Swift.Get(context.Background(), "FLAKY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Meta.RequestID != "req_3" {
		t.Fatalf("request_id = %q", result.Meta.RequestID)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestRetriesExhausted(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"code":"UNAVAILABLE","message":"later","request_id":"req_4"}}`))
	}), WithMaxRetries(2))

	_, err := client.Swift.Get(context.Background(), "BROKEN")
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.Status != 503 {
		t.Fatalf("error = %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

func TestTimeoutBecomesTimeoutError(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(`{"data":{},"meta":{"request_id":"req_6"}}`))
	}), WithTimeout(50*time.Millisecond), WithMaxRetries(0))

	_, err := client.Swift.Get(context.Background(), "SLOW")
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %v", err)
	}
	if apiErr.Code != "TIMEOUT" || apiErr.Status != 0 {
		t.Fatalf("error = %+v", apiErr)
	}
}

func TestPaginateFollowsCursor(t *testing.T) {
	var calls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("cursor") == "next" {
			_, _ = w.Write([]byte(`{"data":[{"id":"inst_2"}],"meta":{"request_id":"req_7","next_cursor":null}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"inst_1"}],"meta":{"request_id":"req_7","next_cursor":"next"}}`))
	}))

	var ids []string
	err := Paginate(context.Background(), client, "/banks", url.Values{"country": {"US"}}, 1,
		func(page *Result[[]BankListItem]) error {
			for _, bank := range page.Data {
				ids = append(ids, bank.ID)
			}
			return nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != "inst_1" || ids[1] != "inst_2" {
		t.Fatalf("ids = %v", ids)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestDoEscapeHatch(t *testing.T) {
	var body string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var payload map[string]any
		_ = decoder.Decode(&payload)
		raw, _ := json.Marshal(payload)
		body = string(raw)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"echo":` + body + `},"meta":{"request_id":"req_10"}}`))
	}))

	raw, err := client.Do(context.Background(), http.MethodPost, "/echo", nil, map[string]string{"name": "Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != `{"name":"Test"}` {
		t.Fatalf("body = %q", body)
	}
	var decoded struct {
		Echo map[string]string `json:"echo"`
	}
	if err := raw.Decode(&decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Echo["name"] != "Test" {
		t.Fatalf("echo = %+v", decoded.Echo)
	}
}

func TestPathSegmentsAreEscaped(t *testing.T) {
	var recorded recordedRequest
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded = recordedRequest{path: r.URL.Path}
		_, _ = w.Write([]byte(`{"data":{"valid":true},"meta":{"request_id":"req_8"}}`))
	}))

	if _, err := client.IBAN.Lookup(context.Background(), "DE89370400440532013000"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recorded.path != "/v1/iban/DE89370400440532013000" {
		t.Fatalf("path = %q", recorded.path)
	}
}
