# RahCar

Persian car listing comparison for Iran. The first version aggregates **approved dealer feeds** and links visitors to the original listing. It does not scrape Divar, Bama, or Sheypoor.

## Structure

- `backend/`: Go API with domain, application, and adapter layers; PostgreSQL, CSV importer, scheduled feed worker
- `frontend/`: React + TypeScript web app
- `sample-data/`: clearly fictional demo feeds
- `scripts/`: local database backup helper

## Run locally

With Docker installed and running:

```powershell
docker compose up --build -d
docker compose run --rm importer
docker compose run --rm importer-b
```

Open http://localhost:5173. The API health endpoint is http://localhost:8081/api/v1/health.

The web server proxies `/api/` to the Go API, so the browser and API work together at `localhost:5173`. The default search starts in Tehran; clear the city filter to search all imported cities. The page supports search, filters, listing comparison, source status, price history, duplicate hints, and saved search alerts. Alerts are checked in the page and do not send notifications.

The market price panel uses [Nemuda's free car market API](https://nemuda.com/data). These are live national asking-price estimates for used cars, **not individual listings**. Nemuda publishes the figures under CC BY 4.0; each displayed price links to its source page and shows the computation date. The API response is cached for six hours to match its update schedule. If it is unavailable, the panel reports that separately from the listing search.

The two import commands load **fictional listings** from two sample sources. They are labelled as demo records in the interface. To remove local demo data, run `docker compose down -v` (this deletes the local PostgreSQL volume).

The API and web app hide fictional offers by default. The empty state has an explicit design-demo option, and API callers can use `include_demo=1`. Demo offers have no outbound ad link. Imported approved listings use their exact `source_url` for the “original listing” link.

## Import an approved dealer feed

CSV fields: `external_id,source_url,title,make,model,trim,production_year,year_calendar,mileage_km,price_toman,city,body_condition,published_at`. `year_calendar` is `jalali` or `gregorian`; `published_at` is RFC 3339. Prices are integer **tomans**. See `sample-data/dealer-a.csv` for an example.

Import requires an explicit source ID and a reference to the permission granted by that source. Feeds are parsed and validated before any row is saved. Re-importing the same source and external ID updates the listing. Imported listings expire after 72 hours unless refreshed. We store no seller contact information.

```powershell
cd backend
$env:DATABASE_URL='postgres://rahcar:rahcar@localhost:5434/rahcar?sslmode=disable'
go run ./cmd/importer -file ../sample-data/dealer-a.csv -source dealer-a -permission-ref agreement-123
```

`backend/migrations/001_init.sql` and `002_operations.sql` create the schema. Docker applies them when its database volume is first created. If upgrading an existing local database, apply the second migration once:

```powershell
docker compose exec -T db psql -v ON_ERROR_STOP=1 -U rahcar -d rahcar -f /migrations/002_operations.sql
```

The migration is safe to rerun. Do not use `docker compose down -v` on a database you want to keep.

## Scheduled feeds and operations

`feed-config.example.json` shows two fictional local feeds. To run one import cycle, use:

```powershell
docker compose --profile worker run --rm worker -config /config/feeds.json -once
```

To keep the worker running every 30 minutes, use `docker compose --profile worker up -d worker`. For an approved real source, replace the example config mount in `compose.yaml` with your private config. Each source needs a stable name, a documented permission reference, and an HTTPS CSV URL; `token_env` optionally names an environment variable containing a bearer token. Keep credentials out of Git. The worker records successes and failures at `/api/v1/sources`. Imports retain price changes, and possible cross-source duplicates are marked as candidates for review.

Create a local PostgreSQL dump with `./scripts/backup.ps1`. It writes a timestamped file under ignored `backups/`. A backup should be copied to separate storage for production use.

## Development

```powershell
cd backend
go test ./...
```

```powershell
cd frontend
npm install
npm run build
```

The frontend calls same-origin `/api/`; the Docker Nginx server proxies those requests to Go. For Vite development outside Docker, its proxy targets `http://localhost:8081`. No paid services or AI APIs are required.

## Production preparation

`compose.prod.yaml` separates the public stack from the local demo. It keeps PostgreSQL and the API internal and binds the web server to `127.0.0.1:8080` for a host reverse proxy to serve with HTTPS. It has no sample import jobs. Copy `.env.production.example` to a private file, replace every placeholder, and deploy with a separate Compose project name:

```powershell
docker compose -p rahkar-prod --env-file .env.production -f compose.prod.yaml config
docker compose -p rahkar-prod --env-file .env.production -f compose.prod.yaml up --build -d
```

Only start the `feeds` profile after placing an approved private JSON config at `FEED_CONFIG_PATH`. `docs/data-onboarding.md` contains the feed contract and a permission request template. Configure a domain and HTTPS reverse proxy before exposing the web app; the Compose file binds only to loopback. Keep the private env file and feed config outside Git. For PostgreSQL upgrades, apply new migrations explicitly because Docker's init scripts only run for a new data volume.

## Data source status

- [Divar Kenar search API](https://divar-ir.github.io/kenar-docs/post/search_post/) has a strict documented 100-call limit and no pagination. [Kenar asks for a support ticket before building on its finder API](https://divar-ir.github.io/kenar-docs/openapi-doc/finder/).
- [Bama's terms](https://bama.ir/terms) prohibit automated copying and require permission to republish listings.
- Sheypoor has public vehicle listings, but we have not found a documented feed with republication permission.
- [Nemuda's car market figures](https://nemuda.com/data) are free to reuse with attribution, but contain aggregate prices rather than sale listings.

The importer is designed for opt-in sources. A live launch still needs approved independent feeds, a hosting provider and domain, production secrets and HTTPS, and a tested off-host backup. The web proxy limits alert requests by client IP; a public deployment should preserve this control or add authentication. The included feeds and prices are fictional.
