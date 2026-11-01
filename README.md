# RahCar

Persian car listing comparison for Iran. The first version aggregates **approved dealer feeds** and links visitors to the original listing. It does not scrape Divar, Bama, or Sheypoor.

## Structure

- `backend/`: Go API, clean architecture layers, PostgreSQL adapter, CSV feed importer
- `frontend/`: React + TypeScript web app
- `sample-data/`: clearly fictional demo feeds

## Run locally

With Docker installed and running:

```powershell
docker compose up --build -d
docker compose run --rm importer
docker compose run --rm importer-b
```

Open http://localhost:5173. The API health endpoint is http://localhost:8081/api/v1/health.

The two import commands load **fictional listings** from two sample sources. They are labelled as demo records in the interface. To remove local demo data, run `docker compose down -v` (this deletes the local PostgreSQL volume).

## Import an approved dealer feed

CSV fields: `external_id,source_url,title,make,model,trim,production_year,year_calendar,mileage_km,price_toman,city,body_condition,published_at`. `year_calendar` is `jalali` or `gregorian`; `published_at` is RFC 3339. Prices are integer **tomans**. See `sample-data/dealer-a.csv` for an example.

Import requires an explicit source ID and a reference to the permission granted by that source. Feeds are parsed and validated before any row is saved. Re-importing the same source and external ID updates the listing. Imported listings expire after 72 hours unless refreshed. We store no seller contact information.

```powershell
cd backend
$env:DATABASE_URL='postgres://rahcar:rahcar@localhost:5434/rahcar?sslmode=disable'
go run ./cmd/importer -file ../sample-data/dealer-a.csv -source dealer-a -permission-ref agreement-123
```

`backend/migrations/001_init.sql` creates the schema. Docker applies it when its database volume is first created. For an existing database, apply the SQL migration manually with `psql`.

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

The frontend uses `VITE_API_URL` if set; by default it calls `http://localhost:8080`. No paid services or AI APIs are required.

## Data source status

- [Divar Kenar search API](https://divar-ir.github.io/kenar-docs/post/search_post/) has a strict documented 100-call limit and no pagination. [Kenar asks for a support ticket before building on its finder API](https://divar-ir.github.io/kenar-docs/openapi-doc/finder/).
- [Bama's terms](https://bama.ir/terms) prohibit automated copying and require permission to republish listings.
- Sheypoor has public vehicle listings, but we have not found a documented feed with republication permission.

The importer is designed for opt-in sources. A live launch needs at least two approved independent feeds.
