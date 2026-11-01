package csvfeed

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"rahcar/backend/internal/domain"
)

var required = []string{"external_id", "source_url", "title", "make", "model", "trim", "production_year", "year_calendar", "mileage_km", "price_toman", "city", "body_condition", "published_at"}

func Parse(input io.Reader, source, permissionRef string, now time.Time) ([]domain.Listing, error) {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(permissionRef) == "" {
		return nil, fmt.Errorf("source and permission reference are required")
	}
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	columns := make(map[string]int, len(header))
	for i, h := range header {
		key := strings.TrimSpace(strings.TrimPrefix(h, "\ufeff"))
		if _, exists := columns[key]; exists {
			return nil, fmt.Errorf("duplicate column %q", key)
		}
		columns[key] = i
	}
	for _, name := range required {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("missing required column %q", name)
		}
	}
	var result []domain.Listing
	seen := map[string]bool{}
	for line := 2; ; line++ {
		values, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", line, err)
		}
		get := func(name string) string {
			i := columns[name]
			if i >= len(values) {
				return ""
			}
			return strings.TrimSpace(values[i])
		}
		year, err := strconv.Atoi(domain.Normalize(get("production_year")))
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid production year: %w", line, err)
		}
		mileage, err := strconv.Atoi(domain.Normalize(get("mileage_km")))
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid mileage: %w", line, err)
		}
		price, err := strconv.ParseInt(domain.Normalize(get("price_toman")), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid price: %w", line, err)
		}
		published, err := time.Parse(time.RFC3339, get("published_at"))
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid published_at: %w", line, err)
		}
		l := domain.Listing{
			SourceName: source, ExternalID: get("external_id"), PermissionRef: permissionRef,
			SourceURL: get("source_url"), Title: get("title"), Make: get("make"), Model: get("model"),
			Trim: get("trim"), ProductionYear: year, YearCalendar: get("year_calendar"),
			MileageKM: mileage, PriceToman: price, City: get("city"), BodyCondition: get("body_condition"),
			PublishedAt: published, LastSeenAt: now, ExpiresAt: now.Add(72 * time.Hour),
		}
		if err := l.Prepare(); err != nil {
			return nil, fmt.Errorf("row %d: %w", line, err)
		}
		if seen[l.ExternalID] {
			return nil, fmt.Errorf("row %d: duplicate external ID %q", line, l.ExternalID)
		}
		seen[l.ExternalID] = true
		result = append(result, l)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("feed contains no listings")
	}
	return result, nil
}
