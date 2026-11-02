// Package fincode is the official Go client for the FinCode global financial
// data API: IBAN validation and bank lookup, SWIFT/BIC, US routing numbers, UK
// sort codes, Australian BSB, Indian IFSC, China CNAPS, card BIN/IIN, bank and
// country directories.
//
//	client := fincode.New(os.Getenv("FINCODE_API_KEY"))
//	result, err := client.Swift.Get(ctx, "ICBKCNBJ")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(result.Data.Bank.NameEN)
//
// Every call resolves to a Result with the payload in Data and the request id in
// Meta. Failures return an *fincode.Error carrying the API code, HTTP status and
// request id.
package fincode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the production API base URL including the version segment.
	DefaultBaseURL = "https://api.ecommica.com/v1"
	// DefaultTimeout is the per-attempt timeout.
	DefaultTimeout = 10 * time.Second
	// DefaultMaxRetries is the number of retries after the first attempt.
	DefaultMaxRetries = 2

	userAgent        = "fincode-go/1.0.0"
	baseBackoffDelay = 250 * time.Millisecond
)

// Error is returned for API errors, timeouts and network failures.
type Error struct {
	// Code is the machine-readable error code, e.g. "INVALID_IBAN".
	Code string
	// Message is the human-readable error message.
	Message string
	// Status is the HTTP status, or 0 for timeouts and network failures.
	Status int
	// RequestID identifies the call on the server side; quote it in support tickets.
	RequestID string
	// RetryAfter is the delay advertised by the Retry-After header, when present.
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("fincode: %s (%s, status %d, request %s)", e.Message, e.Code, e.Status, e.RequestID)
	}
	return fmt.Sprintf("fincode: %s (%s, status %d)", e.Message, e.Code, e.Status)
}

// IsRetryable reports whether the failure is transient and may succeed when
// retried as-is.
func (e *Error) IsRetryable() bool {
	switch e.Status {
	case 0, 429, 502, 503, 504:
		return true
	default:
		return false
	}
}

// Meta is echoed on every response.
type Meta struct {
	RequestID  string  `json:"request_id"`
	NextCursor *string `json:"next_cursor"`
}

// Result is the response envelope: the payload plus metadata.
type Result[T any] struct {
	Data T    `json:"data"`
	Meta Meta `json:"meta"`
}

// Option customises the client.
type Option func(*Client)

// WithBaseURL overrides the API base URL, including the version segment.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithHTTPClient replaces the underlying HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithTimeout sets the per-attempt timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.httpClient.Timeout = timeout
		}
	}
}

// WithMaxRetries sets the retries after the first attempt for 429, 502, 503,
// 504 and network failures.
func WithMaxRetries(retries int) Option {
	return func(c *Client) {
		if retries >= 0 {
			c.maxRetries = retries
		}
	}
}

// Client is safe for concurrent use.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	maxRetries int
	backoff    time.Duration

	// IBAN validates IBANs and resolves the bank behind them.
	IBAN *IBANService
	// Swift looks up SWIFT/BIC codes.
	Swift *SwiftService
	// Routing looks up US ABA routing numbers.
	Routing *RoutingService
	// SortCode looks up UK sort codes.
	SortCode *SortCodeService
	// BSB looks up Australian BSB codes.
	BSB *BSBService
	// IFSC looks up Indian IFSC codes.
	IFSC *IFSCService
	// CNAPS looks up China CNAPS codes.
	CNAPS *CNAPSService
	// BIN looks up card BIN/IIN ranges.
	BIN *BINService
	// Banks is the bank directory.
	Banks *BankService
	// Countries is the country reference data.
	Countries *CountryService
	// Lookup detects the identifier type from free-form input.
	Lookup *LookupService
}

// New returns a client authenticated with an sk_ API key.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: DefaultTimeout},
		maxRetries: DefaultMaxRetries,
		backoff:    baseBackoffDelay,
	}
	for _, opt := range opts {
		opt(c)
	}
	c.IBAN = &IBANService{client: c}
	c.Swift = &SwiftService{client: c}
	c.Routing = &RoutingService{client: c}
	c.SortCode = &SortCodeService{client: c}
	c.BSB = &BSBService{client: c}
	c.IFSC = &IFSCService{client: c}
	c.CNAPS = &CNAPSService{client: c}
	c.BIN = &BINService{client: c}
	c.Banks = &BankService{client: c}
	c.Countries = &CountryService{client: c}
	c.Lookup = &LookupService{client: c}
	return c
}

// RawResult is the response envelope for endpoints without a typed helper.
type RawResult struct {
	Data json.RawMessage
	Meta Meta
}

// Decode binds Data to v.
func (r *RawResult) Decode(v any) error {
	if len(r.Data) == 0 {
		return nil
	}
	return json.Unmarshal(r.Data, v)
}

// Get performs a GET request against a custom path.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*RawResult, error) {
	return c.Do(ctx, http.MethodGet, path, query, nil)
}

// Do is the escape hatch for endpoints without a typed helper.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any) (*RawResult, error) {
	target := c.buildURL(path, query)
	payload, err := encodeBody(body)
	if err != nil {
		return nil, &Error{Code: "INVALID_REQUEST", Message: err.Error()}
	}

	attempts := c.maxRetries + 1
	var lastErr *Error
	for attempt := 1; attempt <= attempts; attempt++ {
		raw, apiErr := c.execute(ctx, method, target, payload)
		if apiErr == nil {
			result := &RawResult{Meta: raw.Meta}
			if len(raw.Data) > 0 {
				result.Data = raw.Data
			}
			return result, nil
		}
		lastErr = apiErr
		if attempt == attempts || !apiErr.IsRetryable() || ctx.Err() != nil {
			break
		}
		if err := sleepContext(ctx, c.backoffDelay(attempt, apiErr.RetryAfter)); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func request[T any](c *Client, ctx context.Context, method, path string, query url.Values, body any) (*Result[T], error) {
	target := c.buildURL(path, query)
	payload, err := encodeBody(body)
	if err != nil {
		return nil, &Error{Code: "INVALID_REQUEST", Message: err.Error()}
	}

	attempts := c.maxRetries + 1
	var lastErr *Error
	for attempt := 1; attempt <= attempts; attempt++ {
		result, apiErr := executeTyped[T](c, ctx, method, target, payload)
		if apiErr == nil {
			return result, nil
		}
		lastErr = apiErr
		if attempt == attempts || !apiErr.IsRetryable() || ctx.Err() != nil {
			break
		}
		if err := sleepContext(ctx, c.backoffDelay(attempt, apiErr.RetryAfter)); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *Client) execute(ctx context.Context, method, target string, payload []byte) (*Result[json.RawMessage], *Error) {
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, &Error{Code: "INVALID_REQUEST", Message: err.Error()}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, networkError(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, networkError(err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var out Result[json.RawMessage]
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &out); err != nil {
				return nil, &Error{
					Code:    "INVALID_RESPONSE",
					Message: "cannot decode response: " + err.Error(),
					Status:  resp.StatusCode,
				}
			}
		}
		return &out, nil
	}
	return nil, apiError(resp, raw)
}

// executeTyped runs a request and decodes the payload into T.
func executeTyped[T any](c *Client, ctx context.Context, method, target string, payload []byte) (*Result[T], *Error) {
	raw, apiErr := c.execute(ctx, method, target, payload)
	if apiErr != nil {
		return nil, apiErr
	}
	var out Result[T]
	out.Meta = raw.Meta
	if len(raw.Data) > 0 {
		if err := json.Unmarshal(raw.Data, &out.Data); err != nil {
			return nil, &Error{Code: "INVALID_RESPONSE", Message: "cannot decode data: " + err.Error(), Status: 200}
		}
	}
	return &out, nil
}

func apiError(resp *http.Response, raw []byte) *Error {
	code := fmt.Sprintf("HTTP_%d", resp.StatusCode)
	message := http.StatusText(resp.StatusCode)
	requestID := ""
	if len(raw) > 0 {
		var envelope struct {
			Error struct {
				Code      string `json:"code"`
				Message   string `json:"message"`
				RequestID string `json:"request_id"`
			} `json:"error"`
		}
		if err := json.Unmarshal(raw, &envelope); err == nil {
			if envelope.Error.Code != "" {
				code = envelope.Error.Code
			}
			if envelope.Error.Message != "" {
				message = envelope.Error.Message
			}
			requestID = envelope.Error.RequestID
		}
	}
	return &Error{
		Code:       code,
		Message:    message,
		Status:     resp.StatusCode,
		RequestID:  requestID,
		RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
	}
}

func networkError(err error) *Error {
	code := "NETWORK_ERROR"
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		code = "TIMEOUT"
	}
	return &Error{Code: code, Message: err.Error()}
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func (c *Client) backoffDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	exponential := c.backoff << (attempt - 1)
	return exponential + time.Duration(rand.Int63n(int64(c.backoff)))
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return &Error{Code: "INTERRUPTED", Message: ctx.Err().Error()}
	case <-timer.C:
		return nil
	}
}

func (c *Client) buildURL(path string, query url.Values) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	return target
}

func encodeBody(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	return json.Marshal(body)
}

// Paginate walks a cursor paginated collection such as "/banks" or
// "/banks/{id}/branches", calling fn for every page until the cursor is
// exhausted.
func Paginate[T any](ctx context.Context, c *Client, path string, query url.Values, limit int, fn func(page *Result[[]T]) error) error {
	cursor := ""
	for {
		params := url.Values{}
		for key, values := range query {
			params[key] = values
		}
		params.Set("limit", strconv.Itoa(limit))
		if cursor != "" {
			params.Set("cursor", cursor)
		}
		page, err := request[[]T](c, ctx, http.MethodGet, path, params, nil)
		if err != nil {
			return err
		}
		if err := fn(page); err != nil {
			return err
		}
		if page.Meta.NextCursor == nil || *page.Meta.NextCursor == "" {
			return nil
		}
		cursor = *page.Meta.NextCursor
	}
}
