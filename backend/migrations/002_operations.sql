CREATE TABLE IF NOT EXISTS price_observations (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  listing_id BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  observed_at TIMESTAMPTZ NOT NULL,
  price_toman BIGINT NOT NULL CHECK (price_toman > 0),
  UNIQUE (listing_id, observed_at)
);

ALTER TABLE listings ADD COLUMN IF NOT EXISTS first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS price_observations_listing_idx
  ON price_observations (listing_id, observed_at DESC);

CREATE TABLE IF NOT EXISTS import_runs (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  source_name TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL,
  finished_at TIMESTAMPTZ NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('success', 'failed')),
  imported_count INTEGER NOT NULL DEFAULT 0,
  error_message TEXT
);

CREATE INDEX IF NOT EXISTS import_runs_source_idx
  ON import_runs (source_name, started_at DESC);

CREATE INDEX IF NOT EXISTS listings_identity_idx
  ON listings (make, model, trim, production_year, year_calendar, city);

CREATE TABLE IF NOT EXISTS duplicate_candidates (
  first_listing_id BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  second_listing_id BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  reason TEXT NOT NULL,
  detected_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (first_listing_id, second_listing_id),
  CHECK (first_listing_id < second_listing_id)
);

CREATE INDEX IF NOT EXISTS duplicate_candidates_second_idx
  ON duplicate_candidates (second_listing_id);

CREATE TABLE IF NOT EXISTS saved_alerts (
  token TEXT PRIMARY KEY,
  query_text TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  make TEXT NOT NULL DEFAULT '',
  max_price_toman BIGINT NOT NULL DEFAULT 0 CHECK (max_price_toman >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL
);
