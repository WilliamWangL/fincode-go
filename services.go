package fincode

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// IBANService validates IBANs and resolves the bank behind them.
type IBANService struct {
	client *Client
}

// Validate checks format, country, length, MOD-97 checksum and BBAN structure.
func (s *IBANService) Validate(ctx context.Context, iban string) (*Result[IBANValidation], error) {
	query := url.Values{"iban": {iban}}
	return request[IBANValidation](s.client, ctx, http.MethodGet, "/iban/validate", query, nil)
}

// Lookup validates an IBAN and resolves the bank behind it.
func (s *IBANService) Lookup(ctx context.Context, iban string) (*Result[IBANLookup], error) {
	return request[IBANLookup](s.client, ctx, http.MethodGet, "/iban/"+url.PathEscape(iban), nil, nil)
}

// SwiftService looks up SWIFT/BIC codes.
type SwiftService struct {
	client *Client
}

// Get resolves an 8 or 11 character SWIFT/BIC code.
func (s *SwiftService) Get(ctx context.Context, code string) (*Result[SwiftRecord], error) {
	return request[SwiftRecord](s.client, ctx, http.MethodGet, "/swift/"+url.PathEscape(code), nil, nil)
}

// RoutingService looks up US ABA routing numbers.
type RoutingService struct {
	client *Client
}

// Get resolves a 9-digit ABA routing number.
func (s *RoutingService) Get(ctx context.Context, number string) (*Result[RoutingRecord], error) {
	return request[RoutingRecord](s.client, ctx, http.MethodGet, "/routing/"+url.PathEscape(number), nil, nil)
}

// SortCodeService looks up UK sort codes.
type SortCodeService struct {
	client *Client
}

// Get resolves a UK sort code, e.g. "60-16-13".
func (s *SortCodeService) Get(ctx context.Context, code string) (*Result[SortCodeRecord], error) {
	return request[SortCodeRecord](s.client, ctx, http.MethodGet, "/sort-code/"+url.PathEscape(code), nil, nil)
}

// BSBService looks up Australian BSB codes.
type BSBService struct {
	client *Client
}

// Get resolves a 6-digit Australian BSB.
func (s *BSBService) Get(ctx context.Context, code string) (*Result[BSBRecord], error) {
	return request[BSBRecord](s.client, ctx, http.MethodGet, "/bsb/"+url.PathEscape(code), nil, nil)
}

// IFSCService looks up Indian IFSC codes.
type IFSCService struct {
	client *Client
}

// Get resolves an 11 character IFSC, including the branch MICR when known.
func (s *IFSCService) Get(ctx context.Context, code string) (*Result[IFSCRecord], error) {
	return request[IFSCRecord](s.client, ctx, http.MethodGet, "/ifsc/"+url.PathEscape(code), nil, nil)
}

// CNAPSService looks up China CNAPS codes.
type CNAPSService struct {
	client *Client
}

// Get resolves a 12-digit CNAPS code.
func (s *CNAPSService) Get(ctx context.Context, code string) (*Result[CNAPSRecord], error) {
	return request[CNAPSRecord](s.client, ctx, http.MethodGet, "/cnaps/"+url.PathEscape(code), nil, nil)
}

// BINService looks up card BIN/IIN ranges.
type BINService struct {
	client *Client
}

// Get resolves a card BIN/IIN to brand, type and issuer.
func (s *BINService) Get(ctx context.Context, bin string) (*Result[BINRecord], error) {
	return request[BINRecord](s.client, ctx, http.MethodGet, "/bin/"+url.PathEscape(bin), nil, nil)
}

// ListOptions are the shared pagination and filtering arguments of the directory.
type ListOptions struct {
	// Country filters by ISO 3166-1 alpha-2 code.
	Country string
	// Limit is the page size, 1-100. Zero leaves the API default (20).
	Limit int
	// Cursor is the value of Meta.NextCursor from the previous page.
	Cursor string
}

func (o ListOptions) query(q string) url.Values {
	values := url.Values{}
	if q != "" {
		values.Set("q", q)
	}
	if o.Country != "" {
		values.Set("country", o.Country)
	}
	if o.Limit > 0 {
		values.Set("limit", strconv.Itoa(o.Limit))
	}
	if o.Cursor != "" {
		values.Set("cursor", o.Cursor)
	}
	return values
}

// BranchOptions filters the branch listing of one institution.
type BranchOptions struct {
	ListOptions
	// Q matches the branch name or address.
	Q string
	// City filters by city name.
	City string
}

// BankService is the bank directory.
type BankService struct {
	client *Client
}

// List returns one page of institutions.
func (s *BankService) List(ctx context.Context, opts ListOptions) (*Result[[]BankListItem], error) {
	return request[[]BankListItem](s.client, ctx, http.MethodGet, "/banks", opts.query(""), nil)
}

// Search matches the English name, local name and short name.
func (s *BankService) Search(ctx context.Context, q string, opts ListOptions) (*Result[[]BankListItem], error) {
	return request[[]BankListItem](s.client, ctx, http.MethodGet, "/banks/search", opts.query(q), nil)
}

// Get returns one institution by id.
func (s *BankService) Get(ctx context.Context, id string) (*Result[BankDetail], error) {
	return request[BankDetail](s.client, ctx, http.MethodGet, "/banks/"+url.PathEscape(id), nil, nil)
}

// BySlug returns one institution by its URL slug.
func (s *BankService) BySlug(ctx context.Context, slug string) (*Result[BankDetail], error) {
	return request[BankDetail](s.client, ctx, http.MethodGet, "/banks/by-slug/"+url.PathEscape(slug), nil, nil)
}

// Branches returns one page of branches of an institution.
func (s *BankService) Branches(ctx context.Context, id string, opts BranchOptions) (*Result[[]Branch], error) {
	values := opts.ListOptions.query("")
	if opts.Q != "" {
		values.Set("q", opts.Q)
	}
	if opts.City != "" {
		values.Set("city", opts.City)
	}
	path := "/banks/" + url.PathEscape(id) + "/branches"
	return request[[]Branch](s.client, ctx, http.MethodGet, path, values, nil)
}

// CountryService exposes country reference data and IBAN formats.
type CountryService struct {
	client *Client
}

// List returns every supported country.
func (s *CountryService) List(ctx context.Context) (*Result[[]Country], error) {
	return request[[]Country](s.client, ctx, http.MethodGet, "/countries", nil, nil)
}

// Get returns one country by ISO 3166-1 alpha-2 code.
func (s *CountryService) Get(ctx context.Context, code string) (*Result[Country], error) {
	return request[Country](s.client, ctx, http.MethodGet, "/countries/"+url.PathEscape(code), nil, nil)
}

// IBANFormat returns the national IBAN structure of a country.
func (s *CountryService) IBANFormat(ctx context.Context, code string) (*Result[IBANFormat], error) {
	path := "/countries/" + url.PathEscape(code) + "/iban-format"
	return request[IBANFormat](s.client, ctx, http.MethodGet, path, nil, nil)
}

// LookupService detects the identifier type from free-form input.
type LookupService struct {
	client *Client
}

// Get resolves any identifier type with one call.
func (s *LookupService) Get(ctx context.Context, q string) (*Result[LookupResult], error) {
	return request[LookupResult](s.client, ctx, http.MethodGet, "/lookup", url.Values{"q": {q}}, nil)
}
