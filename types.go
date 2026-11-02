package fincode

// IdentifierRef is a compact identifier attached to a bank record.
type IdentifierRef struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// BankSummary is the compact institution embedded in identifier responses.
type BankSummary struct {
	ID        string `json:"id"`
	NameEN    string `json:"name_en"`
	NameLocal string `json:"name_local"`
	ShortName string `json:"short_name"`
	Country   string `json:"country"`
	Website   string `json:"website"`
}

// SourceInfo describes where a record came from.
type SourceInfo struct {
	Name        string `json:"name"`
	SourceType  string `json:"source_type"`
	RetrievedAt string `json:"retrieved_at"`
}

// SwiftFormat is the parsed structure of a BIC.
type SwiftFormat struct {
	InstitutionCode string  `json:"institution_code"`
	CountryCode     string  `json:"country_code"`
	LocationCode    string  `json:"location_code"`
	BranchCode      *string `json:"branch_code"`
}

// SwiftRecord is the payload of GET /v1/swift/{code}.
type SwiftRecord struct {
	SwiftCode string       `json:"swift_code"`
	BIC       string       `json:"bic"`
	Format    SwiftFormat  `json:"format"`
	Bank      *BankSummary `json:"bank"`
	City      string       `json:"city"`
	Status    string       `json:"status"`
	Source    SourceInfo   `json:"source"`
}

// RoutingRecord is the payload of GET /v1/routing/{number}.
type RoutingRecord struct {
	RoutingNumber string       `json:"routing_number"`
	ChecksumValid bool         `json:"checksum_valid"`
	Bank          *BankSummary `json:"bank"`
	City          string       `json:"city"`
	State         string       `json:"state"`
	Status        string       `json:"status"`
	Source        SourceInfo   `json:"source"`
}

// SortCodeRecord is the payload of GET /v1/sort-code/{code}.
type SortCodeRecord struct {
	SortCode  string       `json:"sort_code"`
	Formatted string       `json:"formatted"`
	Bank      *BankSummary `json:"bank"`
	City      string       `json:"city"`
	Status    string       `json:"status"`
	Source    SourceInfo   `json:"source"`
}

// BSBRecord is the payload of GET /v1/bsb/{code}.
type BSBRecord struct {
	BSB       string       `json:"bsb"`
	Formatted string       `json:"formatted"`
	Bank      *BankSummary `json:"bank"`
	Branch    string       `json:"branch"`
	City      string       `json:"city"`
	State     string       `json:"state"`
	Status    string       `json:"status"`
	Source    SourceInfo   `json:"source"`
}

// IFSCRecord is the payload of GET /v1/ifsc/{code}.
type IFSCRecord struct {
	IFSC   string       `json:"ifsc"`
	Bank   *BankSummary `json:"bank"`
	Branch string       `json:"branch"`
	City   string       `json:"city"`
	MICR   *string      `json:"micr"`
	Status string       `json:"status"`
	Source SourceInfo   `json:"source"`
}

// CNAPSRecord is the payload of GET /v1/cnaps/{code}.
type CNAPSRecord struct {
	CNAPS    string       `json:"cnaps"`
	BankCode string       `json:"bank_code"`
	Bank     *BankSummary `json:"bank"`
	Branch   string       `json:"branch"`
	City     string       `json:"city"`
	Status   string       `json:"status"`
	Source   SourceInfo   `json:"source"`
}

// BINRecord is the payload of GET /v1/bin/{bin}.
type BINRecord struct {
	BIN         string   `json:"bin"`
	CountryISO2 *string  `json:"country_iso2"`
	IsEU        *bool    `json:"is_eu"`
	IsEEA       *bool    `json:"is_eea"`
	IsSEPA      *bool    `json:"is_sepa"`
	Country     string   `json:"country"`
	Brand       string   `json:"brand"`
	Type        string   `json:"type"`
	Categories  []string `json:"categories"`
	Issuer      string   `json:"issuer"`
	IsValid     *bool    `json:"is_valid"`
}

// IBANChecks is the per-rule outcome of an IBAN validation.
type IBANChecks struct {
	Charset   bool `json:"charset"`
	Country   bool `json:"country"`
	Length    bool `json:"length"`
	Checksum  bool `json:"checksum"`
	Structure bool `json:"structure"`
}

// IBANValidation is the payload of GET /v1/iban/validate.
type IBANValidation struct {
	Valid          bool        `json:"valid"`
	IBAN           string      `json:"iban"`
	Country        *string     `json:"country"`
	CheckDigits    *string     `json:"check_digits"`
	BankCode       *string     `json:"bank_code"`
	AccountNumber  *string     `json:"account_number"`
	Length         int         `json:"length"`
	ExpectedLength *int        `json:"expected_length"`
	Checks         *IBANChecks `json:"checks"`
}

// IBANLookup is the payload of GET /v1/iban/{iban}: the parsed fields plus the
// resolved bank, which is nil while the bank is unknown.
type IBANLookup struct {
	IBANValidation
	Bank *SwiftRecord `json:"bank"`
}

// BankListItem is a row of GET /v1/banks and GET /v1/banks/search.
type BankListItem struct {
	ID          string          `json:"id"`
	NameEN      string          `json:"name_en"`
	NameLocal   string          `json:"name_local"`
	ShortName   string          `json:"short_name"`
	Country     string          `json:"country"`
	Identifiers []IdentifierRef `json:"identifiers"`
}

// BankDetail is the payload of GET /v1/banks/{id} and /v1/banks/by-slug/{slug}.
type BankDetail struct {
	ID          string          `json:"id"`
	NameEN      string          `json:"name_en"`
	NameLocal   string          `json:"name_local"`
	ShortName   string          `json:"short_name"`
	Country     string          `json:"country"`
	Website     string          `json:"website"`
	Status      string          `json:"status"`
	Identifiers []IdentifierRef `json:"identifiers"`
	BranchCount int64           `json:"branch_count"`
}

// Branch is a row of GET /v1/banks/{id}/branches. Code is the national branch
// code (BSB, CNAPS, ...) when the record links one, otherwise nil.
type Branch struct {
	ID           string  `json:"id"`
	BranchName   string  `json:"branch_name"`
	BranchNameEN string  `json:"branch_name_en"`
	Address      string  `json:"address"`
	City         string  `json:"city"`
	State        string  `json:"state"`
	Province     string  `json:"province"`
	PostalCode   string  `json:"postal_code"`
	Status       string  `json:"status"`
	Code         *string `json:"code"`
}

// Country is the payload of the country reference endpoints.
type Country struct {
	CountryCode         string `json:"country_code"`
	ISO3                string `json:"iso3"`
	NameEN              string `json:"name_en"`
	NameLocal           string `json:"name_local"`
	CurrencyCode        string `json:"currency_code"`
	IBANSupported       *bool  `json:"iban_supported"`
	SwiftSupported      *bool  `json:"swift_supported"`
	LocalIdentifierType string `json:"local_identifier_type"`
	Status              string `json:"status"`
}

// Position is a 1-based span inside an IBAN.
type Position struct {
	Start  int `json:"start"`
	Length int `json:"length"`
}

// IBANFormat describes the national IBAN structure of a country.
type IBANFormat struct {
	CountryCode            string   `json:"country_code"`
	IBANLength             int      `json:"iban_length"`
	Structure              string   `json:"structure"`
	BankIdentifierPosition Position `json:"bank_identifier_position"`
	AccountNumberPosition  Position `json:"account_number_position"`
	Example                string   `json:"example"`
}

// LookupBank is the institution inside a lookup result.
type LookupBank struct {
	ID      string `json:"id"`
	NameEN  string `json:"name_en"`
	Country string `json:"country"`
}

// LookupIdentifier is the resolved identifier of a lookup result.
type LookupIdentifier struct {
	Type  string      `json:"type"`
	Value string      `json:"value"`
	Bank  *LookupBank `json:"bank"`
}

// LookupSuggestion is a close candidate when the input cannot be detected.
type LookupSuggestion struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
	Name    string `json:"name"`
	Country string `json:"country"`
}

// LookupResult is the payload of GET /v1/lookup.
type LookupResult struct {
	Query        string             `json:"query"`
	DetectedType string             `json:"detected_type"`
	Resolved     bool               `json:"resolved"`
	Identifier   *LookupIdentifier  `json:"identifier"`
	Suggestions  []LookupSuggestion `json:"suggestions"`
}
