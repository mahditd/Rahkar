CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS listings (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  source_name TEXT NOT NULL,
  source_listing_id TEXT NOT NULL,
  permission_ref TEXT NOT NULL,
  source_url TEXT NOT NULL,
  title TEXT NOT NULL,
  make TEXT NOT NULL,
  model TEXT NOT NULL,
  trim TEXT NOT NULL,
  production_year INTEGER NOT NULL,
  year_calendar TEXT NOT NULL CHECK (year_calendar IN ('jalali', 'gregorian')),
  mileage_km INTEGER NOT NULL CHECK (mileage_km >= 0),
  price_toman BIGINT NOT NULL CHECK (price_toman > 0),
  city TEXT NOT NULL,
  body_condition TEXT NOT NULL,
  published_at TIMESTAMPTZ NOT NULL,
  last_seen_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  search_text TEXT NOT NULL,
  UNIQUE (source_name, source_listing_id)
);

CREATE INDEX IF NOT EXISTS listings_city_price_idx ON listings (city, price_toman);
CREATE INDEX IF NOT EXISTS listings_search_trgm_idx ON listings USING GIN (search_text gin_trgm_ops);
CREATE INDEX IF NOT EXISTS listings_expires_idx ON listings (expires_at);
