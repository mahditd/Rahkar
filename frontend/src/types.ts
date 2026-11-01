export type Listing = {
  id: number;
  source_name: string;
  external_id: string;
  source_url: string;
  title: string;
  make: string;
  model: string;
  trim: string;
  production_year: number;
  year_calendar: "jalali" | "gregorian";
  mileage_km: number;
  price_toman: number;
  city: string;
  body_condition: string;
  published_at: string;
  last_seen_at: string;
  expires_at: string;
  potential_duplicate_count: number;
};

export type ListingResponse = { items: Listing[]; count: number };

export type PriceObservation = { observed_at: string; price_toman: number };
export type SourceStatus = {
  source_name: string;
  status: "success" | "failed";
  last_run_at: string;
  imported_count: number;
};
export type SavedAlert = {
  token: string;
  query: string;
  city: string;
  make: string;
  max_price_toman: number;
  created_at: string;
  expires_at: string;
};

export type MarketPrice = {
  name: string;
  model_year: number;
  price_toman: number;
  mileage_km: number;
  data_points: number;
  source_url: string;
};

export type MarketSnapshot = { computed_at: string; items: MarketPrice[] };
