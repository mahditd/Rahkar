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

The two import commands load **fictional listings** from two sample sources. They are labelled as demo records in the interface. To remove local demo data, run `docker compose down -v` (this deletes the local PostgreSQL volume).

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

## Data source status

- [Divar Kenar search API](https://divar-ir.github.io/kenar-docs/post/search_post/) has a strict documented 100-call limit and no pagination. [Kenar asks for a support ticket before building on its finder API](https://divar-ir.github.io/kenar-docs/openapi-doc/finder/).
- [Bama's terms](https://bama.ir/terms) prohibit automated copying and require permission to republish listings.
- Sheypoor has public vehicle listings, but we have not found a documented feed with republication permission.

The importer is designed for opt-in sources. A live launch still needs approved independent feeds, a hosting provider and domain, production secrets and HTTPS, and a tested off-host backup. The web proxy limits alert requests by client IP; a public deployment should preserve this control or add authentication. The included feeds and prices are fictional.
