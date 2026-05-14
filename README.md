# gob-vendor

A Go web scraper for [HonduCompras](http://sicc.honducompras.gob.hn/HC/procesos/busquedahistorico.aspx), the public procurement portal operated by Honduras' [ONCAE](https://www.oncae.gob.hn/) (Oficina Normativa de Contratación y Adquisiciones del Estado).

The site publishes tenders, calls for offers, and awards across every public-sector buyer in Honduras. `gob-vendor` collects those listings into a searchable dataset so vendors can spot opportunities to bid on government contracts.

## What it does

- Paginates the historical-search results page on `sicc.honducompras.gob.hn`
- Extracts procurement entries (entity, modality, stage, currency, dates, amounts)
- Fetches the detail page for each entry
- Persists the results to one of two backends: ZincSearch (for full-text search) or `.json` files (XLSX export stub)

## How it works

The target site is an ASP.NET Web Forms application, so pagination is driven by posting back `__VIEWSTATE`, `__VIEWSTATEGENERATOR`, `__EVENTVALIDATION`, `__EVENTTARGET`, and `__EVENTARGUMENT` between requests. The collector maintains that session state across goroutines while a semaphore caps pagination concurrency.

```
main.go
├── client/        Resilient HTTP client (heimdall) with optional proxy rotation,
│                  user-agent rotation, and ISO-8859-1 → UTF-8 decoding.
├── collector/     Search orchestration: pagination, session state, detail fetch.
├── webscraper/    HTML parsing helpers built on goquery.
├── storage/       Pluggable persistence: ZincSearch or local file export.
├── env/           Typed env-var parsing helpers.
├── logger/        slog wrapper with context-scoped fields.
└── shared/        Cross-package entities.
```

## Requirements

- Go 1.26+
- (optional) A running [ZincSearch](https://github.com/zincsearch/zincsearch) instance if you want full-text indexing

## Configuration

Copy the example env file and fill in values:

```bash
cp .env.example .env
```

| Variable | Default | Purpose |
|---|---|---|
| `ZINCSEARCH_URL` | `""` | Base URL of the ZincSearch instance |
| `ZINCSEARCH_USERNAME` | `""` | ZincSearch admin user |
| `ZINCSEARCH_PASSWORD` | `""` | ZincSearch admin password |
| `HTTP_CLIENT_RETRIES` | `3` | Retry budget for the HTTP client |
| `HTTP_CLIENT_TIMEOUT` | `30` | HTTP timeout in seconds |
| `HTTP_CLIENT_SKIP_VERIFY` | `false` | Skip TLS verification (development only) |
| `HTTP_CLIENT_FORCE_ATTEMPT_HTTP2` | `false` | Force HTTP/2 attempts |
| `FORCE_HTTPS_REDIRECT` | `false` | Follow HTTPS redirects |
| `USE_PROXY_ROTATION` | `false` | Rotate through a list of proxies |
| `HTTP_PROXY_URL` | `""` | Semicolon-separated proxy list when rotation is enabled |

## Running

```bash
go run main.go
```

By default the collector queries:

- **Tipo de adquisición:** Suministro de Bienes y/o Servicios
- **Etapa:** Recepción de Ofertas
- **Modalidad:** Compra Menor

Change these in `main.go` by selecting different constants from `collector/models.go` (e.g. `AdquisitionTypeWorks`, `AdquisitionStageAwardedType`, `AdquisitionCategoryLicitationOrContestType`).

## Tests

```bash
go test ./...
```

## A note on rate limiting

The collector sleeps between page requests (`10s` normally, `25s` every 20 pages) to be a polite citizen of a public-sector site that isn't built for scraping load. Please keep those defaults — or raise them — rather than lowering them.

## License

MIT — see [LICENSE](./LICENSE).
