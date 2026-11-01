package domain

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

type Listing struct {
	ID                      int64     `json:"id"`
	SourceName              string    `json:"source_name"`
	ExternalID              string    `json:"external_id"`
	PermissionRef           string    `json:"-"`
	SourceURL               string    `json:"source_url"`
	Title                   string    `json:"title"`
	Make                    string    `json:"make"`
	Model                   string    `json:"model"`
	Trim                    string    `json:"trim"`
	ProductionYear          int       `json:"production_year"`
	YearCalendar            string    `json:"year_calendar"`
	MileageKM               int       `json:"mileage_km"`
	PriceToman              int64     `json:"price_toman"`
	City                    string    `json:"city"`
	BodyCondition           string    `json:"body_condition"`
	PublishedAt             time.Time `json:"published_at"`
	LastSeenAt              time.Time `json:"last_seen_at"`
	ExpiresAt               time.Time `json:"expires_at"`
	SearchText              string    `json:"-"`
	PotentialDuplicateCount int       `json:"potential_duplicate_count"`
}

type PriceObservation struct {
	ObservedAt time.Time `json:"observed_at"`
	PriceToman int64     `json:"price_toman"`
}

func (l *Listing) Prepare() error {
	l.SourceName = strings.TrimSpace(l.SourceName)
	l.ExternalID = strings.TrimSpace(l.ExternalID)
	l.PermissionRef = strings.TrimSpace(l.PermissionRef)
	l.Title = strings.TrimSpace(l.Title)
	l.Make = strings.TrimSpace(l.Make)
	l.Model = strings.TrimSpace(l.Model)
	l.Trim = strings.TrimSpace(l.Trim)
	l.City = strings.TrimSpace(l.City)
	l.BodyCondition = strings.TrimSpace(l.BodyCondition)
	l.Make = Normalize(l.Make)
	l.Model = Normalize(l.Model)
	l.Trim = Normalize(l.Trim)
	l.City = Normalize(l.City)

	if l.SourceName == "" || l.ExternalID == "" || l.PermissionRef == "" || l.Title == "" || l.Make == "" || l.Model == "" || l.City == "" {
		return errors.New("source, external ID, permission reference, title, make, model, and city are required")
	}
	u, err := url.Parse(l.SourceURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return errors.New("source URL must be an absolute HTTP(S) URL")
	}
	if l.YearCalendar != "jalali" && l.YearCalendar != "gregorian" {
		return errors.New("year calendar must be jalali or gregorian")
	}
	if (l.YearCalendar == "jalali" && (l.ProductionYear < 1300 || l.ProductionYear > 1500)) ||
		(l.YearCalendar == "gregorian" && (l.ProductionYear < 1900 || l.ProductionYear > 2100)) {
		return errors.New("production year is outside the supported range")
	}
	if l.MileageKM < 0 || l.PriceToman <= 0 {
		return errors.New("mileage must be nonnegative and price must be positive")
	}
	if l.PublishedAt.IsZero() || l.LastSeenAt.IsZero() || !l.ExpiresAt.After(l.LastSeenAt) {
		return errors.New("valid publication and expiry times are required")
	}
	l.SearchText = Normalize(strings.Join([]string{l.Title, l.Make, l.Model, l.Trim}, " "))
	return nil
}

func Normalize(s string) string {
	r := strings.NewReplacer("ي", "ی", "ى", "ی", "ك", "ک", "۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9", "٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9")
	return strings.ToLower(strings.Join(strings.Fields(r.Replace(s)), " "))
}
