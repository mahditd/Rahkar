# Listing source onboarding

RahCar needs permission from a listing owner before it can display the owner's inventory. A public page or undocumented endpoint alone does not establish permission to copy and republish its data.

## Request to send to a dealer or marketplace

> We are building RahCar, a car listing comparison site for Iran. We would like to show your available cars with their asking price, model, year, mileage, city, and a link back to the original listing. We will identify your business as the source, refresh the data on an agreed schedule, remove stale listings, and will not display seller contact details. Could you provide a CSV feed or API and written permission for this use? Please tell us your update interval, rate limit, attribution requirements, and how you want removals handled.

No message is sent automatically. Save the source's written response and give each approved feed a stable source name and a permission reference.

## Feed contract

CSV header: `external_id,source_url,title,make,model,trim,production_year,year_calendar,mileage_km,price_toman,city,body_condition,published_at`

- `external_id` stays stable across updates; a source can change price without changing ID.
- `source_url` links to the original listing. Use HTTPS.
- `price_toman` is a positive integer in tomans; `mileage_km` is a nonnegative integer.
- `year_calendar` is `jalali` or `gregorian`; `published_at` is RFC 3339.
- The feed must omit seller names, phone numbers, and other personal contact details.
- The worker expects a full snapshot on each run. Listings not refreshed expire after 72 hours; arrange a refresh interval shorter than that.
- For HTTPS feeds, the worker accepts an optional bearer token through `token_env`. The feed URL and token belong in private configuration.

The import validates the full CSV before saving it. Test an approved source with a small feed before scheduling it. Check `GET /api/v1/sources` after an import and verify links, prices, city names, and expiry behavior in the web app.

## Current source options

- [Divar Kenar search](https://divar-ir.github.io/kenar-docs/post/search_post/) is documented, but its 100-call total limit and lack of pagination make it unsuitable as the main continuous feed.
- [Bama's terms](https://bama.ir/terms) require permission for automated copying or republication.
- [Nemuda's car prices](https://nemuda.com/data) are published for free reuse under CC BY 4.0 with source, page link, and computation date. RahCar uses them as separate market estimates because they are not individual sale listings.
- A dealer-supplied feed with explicit display permission is the practical first live source. The project's fictional CSV files show the required format.
