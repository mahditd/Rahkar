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
};

export type ListingResponse = { items: Listing[]; count: number };
