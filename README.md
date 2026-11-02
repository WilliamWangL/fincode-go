# fincode-go

Official Go client for the [FinCode](https://fincode.ecommica.com) global financial data API —
IBAN validation and bank lookup, SWIFT/BIC, US routing numbers, UK sort codes, Australian BSB,
Indian IFSC, China CNAPS, card BIN/IIN, bank and country directories.

- Go 1.22+, **zero dependencies** (standard library only)
- Bearer authentication, per-attempt timeouts, retries with exponential backoff
- Typed payloads plus a raw escape hatch for new endpoints
- Errors carry the API code, HTTP status and request id

## Install

```bash
go get github.com/WilliamWangL/fincode-go
```

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	fincode "github.com/WilliamWangL/fincode-go"
)

func main() {
	client := fincode.New(os.Getenv("FINCODE_API_KEY"))

	result, err := client.Swift.Get(context.Background(), "ICBKCNBJ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Data.Bank.NameEN) // Industrial and Commercial Bank of China
	fmt.Println(result.Meta.RequestID)   // req_1a2b3c4d
}
```

## Configuration

```go
client := fincode.New(
	os.Getenv("FINCODE_API_KEY"),
	fincode.WithBaseURL("https://api.ecommica.com/v1"),
	fincode.WithTimeout(10*time.Second),
	fincode.WithMaxRetries(2),
	fincode.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
)
```

| Option           | Default                       | Description                                                          |
| ---------------- | ----------------------------- | -------------------------------------------------------------------- |
| `WithBaseURL`    | `https://api.ecommica.com/v1` | Base URL including the version segment.                              |
| `WithTimeout`    | `10s`                         | Per-attempt timeout.                                                 |
| `WithMaxRetries` | `2`                           | Retries after the first attempt for 429/502/503/504 and network errors. |
| `WithHTTPClient` | `&http.Client{}`              | Custom transport for proxies, TLS or tracing.                        |

## Methods

Every method returns `*Result[T]` with `Data` (typed payload) and `Meta` (`RequestID`, `NextCursor`).

| Method                                       | Endpoint                              |
| -------------------------------------------- | ------------------------------------- |
| `client.IBAN.Validate(ctx, iban)`            | `GET /v1/iban/validate?iban=`         |
| `client.IBAN.Lookup(ctx, iban)`              | `GET /v1/iban/{iban}`                 |
| `client.Swift.Get(ctx, code)`                | `GET /v1/swift/{code}`                |
| `client.Routing.Get(ctx, number)`            | `GET /v1/routing/{number}`            |
| `client.SortCode.Get(ctx, code)`             | `GET /v1/sort-code/{code}`            |
| `client.BSB.Get(ctx, code)`                  | `GET /v1/bsb/{code}`                  |
| `client.IFSC.Get(ctx, code)`                 | `GET /v1/ifsc/{code}`                 |
| `client.CNAPS.Get(ctx, code)`                | `GET /v1/cnaps/{code}`                |
| `client.BIN.Get(ctx, bin)`                   | `GET /v1/bin/{bin}`                   |
| `client.Banks.List(ctx, opts)`               | `GET /v1/banks`                       |
| `client.Banks.Search(ctx, q, opts)`          | `GET /v1/banks/search?q=`             |
| `client.Banks.Get(ctx, id)`                  | `GET /v1/banks/{id}`                  |
| `client.Banks.BySlug(ctx, slug)`             | `GET /v1/banks/by-slug/{slug}`        |
| `client.Banks.Branches(ctx, id, opts)`       | `GET /v1/banks/{id}/branches`         |
| `client.Countries.List(ctx)`                 | `GET /v1/countries`                   |
| `client.Countries.Get(ctx, code)`            | `GET /v1/countries/{code}`            |
| `client.Countries.IBANFormat(ctx, code)`     | `GET /v1/countries/{code}/iban-format` |
| `client.Lookup.Get(ctx, q)`                  | `GET /v1/lookup?q=`                   |

Anything else is one call away:

```go
raw, err := client.Get(ctx, "/banks/by-slug/chase", nil)
var bank fincode.BankDetail
err = raw.Decode(&bank)
```

## Pagination

```go
err := fincode.Paginate(ctx, client, "/banks", url.Values{"country": {"US"}}, 100,
	func(page *fincode.Result[[]fincode.BankListItem]) error {
		for _, bank := range page.Data {
			fmt.Println(bank.NameEN)
		}
		return nil
	})
```

## Errors and retries

```go
var apiErr *fincode.Error
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.Code)      // "INVALID_SWIFT"
	fmt.Println(apiErr.Status)    // 422
	fmt.Println(apiErr.RequestID) // "req_5e6f7a8b"
}
```

`429`, `502`, `503` and `504` responses — plus network failures and timeouts — are retried with
exponential backoff, honouring `Retry-After`. Other `4xx` validation errors are never retried.

## Development

```bash
gofmt -l .
go vet ./...
go test ./... -v
```

## Links

- API documentation: https://fincode.ecommica.com/docs
- SDK guide: https://fincode.ecommica.com/docs/sdks
- Issues: https://github.com/WilliamWangL/fincode-go/issues

## License

MIT
