package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"rahcar/backend/internal/application"
	"rahcar/backend/internal/domain"
)

type Listings struct{ pool *pgxpool.Pool }

func NewListings(pool *pgxpool.Pool) *Listings { return &Listings{pool: pool} }

func (r *Listings) Search(ctx context.Context, f application.SearchFilter) (application.SearchResult, error) {
	args := []any{f.Query, f.City, f.Make, f.MaxPriceToman, f.MaxMileageKM, f.Condition, f.YearFrom, f.YearTo, f.YearCalendar}
	const where = ` FROM listings
		WHERE expires_at > now()
		  AND ($1 = '' OR search_text ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR city = $2)
		  AND ($3 = '' OR make = $3)
		  AND ($4::bigint = 0 OR price_toman <= $4)
		  AND ($5::int = 0 OR mileage_km <= $5)
		  AND ($6 = '' OR body_condition = $6)
		  AND ($7::int = 0 OR production_year >= $7)
		  AND ($8::int = 0 OR production_year <= $8)
		  AND ($9 = '' OR year_calendar = $9)`
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*)`+where, args...).Scan(&total); err != nil {
		return application.SearchResult{}, fmt.Errorf("count listings: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, source_name, source_listing_id, source_url, title, make, model, trim,
		       production_year, year_calendar, mileage_km, price_toman, city, body_condition,
		       published_at, last_seen_at, expires_at`+where+`
		ORDER BY published_at DESC, id DESC
		LIMIT $10 OFFSET $11`, append(args, f.Limit, (f.Page-1)*f.Limit)...)
	if err != nil {
		return application.SearchResult{}, fmt.Errorf("query listings: %w", err)
	}
	defer rows.Close()
	listings := make([]domain.Listing, 0)
	for rows.Next() {
		var l domain.Listing
		if err := rows.Scan(&l.ID, &l.SourceName, &l.ExternalID, &l.SourceURL, &l.Title, &l.Make, &l.Model, &l.Trim, &l.ProductionYear, &l.YearCalendar, &l.MileageKM, &l.PriceToman, &l.City, &l.BodyCondition, &l.PublishedAt, &l.LastSeenAt, &l.ExpiresAt); err != nil {
			return application.SearchResult{}, fmt.Errorf("scan listing: %w", err)
		}
		listings = append(listings, l)
	}
	if err := rows.Err(); err != nil {
		return application.SearchResult{}, err
	}
	return application.SearchResult{Items: listings, Count: total}, nil
}

func (r *Listings) SaveBatch(ctx context.Context, listings []domain.Listing) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin import: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, l := range listings {
		_, err := tx.Exec(ctx, `
			INSERT INTO listings (source_name, source_listing_id, permission_ref, source_url, title, make, model, trim,
			  production_year, year_calendar, mileage_km, price_toman, city, body_condition,
			  published_at, last_seen_at, expires_at, search_text)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
			ON CONFLICT (source_name, source_listing_id) DO UPDATE SET
			  permission_ref = EXCLUDED.permission_ref, source_url = EXCLUDED.source_url,
			  title = EXCLUDED.title, make = EXCLUDED.make, model = EXCLUDED.model, trim = EXCLUDED.trim,
			  production_year = EXCLUDED.production_year, year_calendar = EXCLUDED.year_calendar,
			  mileage_km = EXCLUDED.mileage_km, price_toman = EXCLUDED.price_toman,
			  city = EXCLUDED.city, body_condition = EXCLUDED.body_condition,
			  published_at = EXCLUDED.published_at, last_seen_at = EXCLUDED.last_seen_at,
			  expires_at = EXCLUDED.expires_at, search_text = EXCLUDED.search_text`,
			l.SourceName, l.ExternalID, l.PermissionRef, l.SourceURL, l.Title, l.Make, l.Model, l.Trim,
			l.ProductionYear, l.YearCalendar, l.MileageKM, l.PriceToman, l.City, l.BodyCondition,
			l.PublishedAt, l.LastSeenAt, l.ExpiresAt, l.SearchText)
		if err != nil {
			return fmt.Errorf("save listing %s: %w", l.ExternalID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit import: %w", err)
	}
	return nil
}
