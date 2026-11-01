package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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
		       published_at, last_seen_at, expires_at,
		       (SELECT count(*) FROM duplicate_candidates d WHERE d.first_listing_id = listings.id OR d.second_listing_id = listings.id)`+where+`
		ORDER BY published_at DESC, id DESC
		LIMIT $10 OFFSET $11`, append(args, f.Limit, (f.Page-1)*f.Limit)...)
	if err != nil {
		return application.SearchResult{}, fmt.Errorf("query listings: %w", err)
	}
	defer rows.Close()
	listings := make([]domain.Listing, 0)
	for rows.Next() {
		var l domain.Listing
		if err := rows.Scan(&l.ID, &l.SourceName, &l.ExternalID, &l.SourceURL, &l.Title, &l.Make, &l.Model, &l.Trim, &l.ProductionYear, &l.YearCalendar, &l.MileageKM, &l.PriceToman, &l.City, &l.BodyCondition, &l.PublishedAt, &l.LastSeenAt, &l.ExpiresAt, &l.PotentialDuplicateCount); err != nil {
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
	updatedIDs := make([]int64, 0, len(listings))
	for _, l := range listings {
		var oldPrice int64
		oldErr := tx.QueryRow(ctx, `SELECT price_toman FROM listings WHERE source_name=$1 AND source_listing_id=$2 FOR UPDATE`, l.SourceName, l.ExternalID).Scan(&oldPrice)
		if oldErr != nil && !errors.Is(oldErr, pgx.ErrNoRows) {
			return fmt.Errorf("read previous price: %w", oldErr)
		}
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO listings (source_name, source_listing_id, permission_ref, source_url, title, make, model, trim,
			  production_year, year_calendar, mileage_km, price_toman, city, body_condition,
			  published_at, last_seen_at, expires_at, search_text, first_seen_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$16)
			ON CONFLICT (source_name, source_listing_id) DO UPDATE SET
			  permission_ref = EXCLUDED.permission_ref, source_url = EXCLUDED.source_url,
			  title = EXCLUDED.title, make = EXCLUDED.make, model = EXCLUDED.model, trim = EXCLUDED.trim,
			  production_year = EXCLUDED.production_year, year_calendar = EXCLUDED.year_calendar,
			  mileage_km = EXCLUDED.mileage_km, price_toman = EXCLUDED.price_toman,
			  city = EXCLUDED.city, body_condition = EXCLUDED.body_condition,
			  published_at = EXCLUDED.published_at, last_seen_at = EXCLUDED.last_seen_at,
			  expires_at = EXCLUDED.expires_at, search_text = EXCLUDED.search_text
			RETURNING id`,
			l.SourceName, l.ExternalID, l.PermissionRef, l.SourceURL, l.Title, l.Make, l.Model, l.Trim,
			l.ProductionYear, l.YearCalendar, l.MileageKM, l.PriceToman, l.City, l.BodyCondition,
			l.PublishedAt, l.LastSeenAt, l.ExpiresAt, l.SearchText).Scan(&id)
		if err != nil {
			return fmt.Errorf("save listing %s: %w", l.ExternalID, err)
		}
		updatedIDs = append(updatedIDs, id)
		if errors.Is(oldErr, pgx.ErrNoRows) || oldPrice != l.PriceToman {
			_, err = tx.Exec(ctx, `INSERT INTO price_observations (listing_id, observed_at, price_toman)
				VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, id, l.LastSeenAt, l.PriceToman)
			if err != nil {
				return fmt.Errorf("record price: %w", err)
			}
		}
	}
	_, err = tx.Exec(ctx, `DELETE FROM duplicate_candidates WHERE first_listing_id = ANY($1::bigint[]) OR second_listing_id = ANY($1::bigint[])`, updatedIDs)
	if err != nil {
		return fmt.Errorf("clear duplicate candidates: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO duplicate_candidates (first_listing_id, second_listing_id, reason, detected_at)
		SELECT a.id, b.id, 'same model, city and condition; near mileage and price', now()
		FROM listings a JOIN listings b ON a.id < b.id
		  AND a.source_name <> b.source_name
		  AND a.make = b.make AND a.model = b.model AND a.trim = b.trim
		  AND a.production_year = b.production_year AND a.year_calendar = b.year_calendar
		  AND a.city = b.city AND a.body_condition = b.body_condition
		WHERE (a.id = ANY($1::bigint[]) OR b.id = ANY($1::bigint[]))
		  AND a.expires_at > now() AND b.expires_at > now()
		  AND abs(a.mileage_km - b.mileage_km) <= 500
		  AND abs(a.price_toman - b.price_toman) * 100 <= LEAST(a.price_toman, b.price_toman) * 2
		ON CONFLICT (first_listing_id, second_listing_id) DO UPDATE SET detected_at = EXCLUDED.detected_at`, updatedIDs)
	if err != nil {
		return fmt.Errorf("find duplicate candidates: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit import: %w", err)
	}
	return nil
}

func (r *Listings) History(ctx context.Context, listingID int64) ([]domain.PriceObservation, error) {
	rows, err := r.pool.Query(ctx, `SELECT observed_at, price_toman FROM price_observations WHERE listing_id=$1 ORDER BY observed_at DESC LIMIT 100`, listingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.PriceObservation, 0)
	for rows.Next() {
		var point domain.PriceObservation
		if err := rows.Scan(&point.ObservedAt, &point.PriceToman); err != nil {
			return nil, err
		}
		result = append(result, point)
	}
	return result, rows.Err()
}

func (r *Listings) RecordImportRun(ctx context.Context, source string, started time.Time, count int, runErr error) error {
	status := "success"
	message := ""
	if runErr != nil {
		status = "failed"
		message = runErr.Error()
		if len(message) > 1000 {
			message = message[:1000]
		}
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO import_runs (source_name, started_at, finished_at, status, imported_count, error_message)
		VALUES ($1,$2,now(),$3,$4,$5)`, source, started, status, count, message)
	return err
}

func (r *Listings) DeleteExpiredAlerts(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM saved_alerts WHERE expires_at < now()`)
	return err
}

func (r *Listings) SourceStatuses(ctx context.Context) ([]domain.SourceStatus, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT ON (source_name) source_name, status, finished_at, imported_count
		FROM import_runs ORDER BY source_name, started_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.SourceStatus, 0)
	for rows.Next() {
		var status domain.SourceStatus
		if err := rows.Scan(&status.SourceName, &status.Status, &status.LastRunAt, &status.ImportedCount); err != nil {
			return nil, err
		}
		result = append(result, status)
	}
	return result, rows.Err()
}

func (r *Listings) CreateAlert(ctx context.Context, alert domain.SavedAlert) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO saved_alerts (token, query_text, city, make, max_price_toman, created_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, alert.Token, alert.Query, alert.City, alert.Make,
		alert.MaxPriceToman, alert.CreatedAt, alert.ExpiresAt)
	return err
}

func (r *Listings) AlertMatches(ctx context.Context, token string) (domain.SavedAlert, []domain.Listing, error) {
	var a domain.SavedAlert
	err := r.pool.QueryRow(ctx, `SELECT token, query_text, city, make, max_price_toman, created_at, expires_at
		FROM saved_alerts WHERE token=$1 AND expires_at>now()`, token).Scan(
		&a.Token, &a.Query, &a.City, &a.Make, &a.MaxPriceToman, &a.CreatedAt, &a.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.SavedAlert{}, nil, application.ErrAlertNotFound
	}
	if err != nil {
		return domain.SavedAlert{}, nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id, source_name, source_listing_id, source_url, title, make, model, trim,
		production_year, year_calendar, mileage_km, price_toman, city, body_condition,
		published_at, last_seen_at, expires_at
		FROM listings WHERE expires_at>now()
		  AND (first_seen_at >= $1 OR ($5::bigint>0 AND EXISTS (
		    SELECT 1 FROM price_observations p WHERE p.listing_id=listings.id
		      AND p.observed_at >= $1 AND p.price_toman <= $5)))
		  AND ($2='' OR search_text ILIKE '%' || $2 || '%')
		  AND ($3='' OR city=$3) AND ($4='' OR make=$4)
		  AND ($5::bigint=0 OR price_toman<=$5)
		ORDER BY last_seen_at DESC LIMIT 50`, a.CreatedAt, a.Query, a.City, a.Make, a.MaxPriceToman)
	if err != nil {
		return domain.SavedAlert{}, nil, err
	}
	defer rows.Close()
	items := make([]domain.Listing, 0)
	for rows.Next() {
		var l domain.Listing
		if err := rows.Scan(&l.ID, &l.SourceName, &l.ExternalID, &l.SourceURL, &l.Title, &l.Make, &l.Model, &l.Trim,
			&l.ProductionYear, &l.YearCalendar, &l.MileageKM, &l.PriceToman, &l.City, &l.BodyCondition,
			&l.PublishedAt, &l.LastSeenAt, &l.ExpiresAt); err != nil {
			return domain.SavedAlert{}, nil, err
		}
		items = append(items, l)
	}
	return a, items, rows.Err()
}
