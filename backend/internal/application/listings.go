package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"rahcar/backend/internal/domain"
)

type SearchFilter struct {
	Query         string
	IncludeDemo   bool
	City          string
	Make          string
	MaxPriceToman int64
	MaxMileageKM  int
	Condition     string
	YearFrom      int
	YearTo        int
	YearCalendar  string
	Page          int
	Limit         int
}

type SearchResult struct {
	Items []domain.Listing `json:"items"`
	Count int              `json:"count"`
}

type Repository interface {
	Search(context.Context, SearchFilter) (SearchResult, error)
	SaveBatch(context.Context, []domain.Listing) error
	History(context.Context, int64) ([]domain.PriceObservation, error)
	SourceStatuses(context.Context) ([]domain.SourceStatus, error)
	CreateAlert(context.Context, domain.SavedAlert) error
	AlertMatches(context.Context, string) (domain.SavedAlert, []domain.Listing, error)
}

type Listings struct{ repo Repository }

var ErrAlertNotFound = errors.New("alert not found")
var ErrInvalidAlertFilters = errors.New("invalid alert filters")
var ErrInvalidAlertToken = errors.New("invalid alert token")

func NewListings(repo Repository) *Listings { return &Listings{repo: repo} }

func (s *Listings) Search(ctx context.Context, f SearchFilter) (SearchResult, error) {
	f.Query = domain.Normalize(f.Query)
	f.City = domain.Normalize(f.City)
	f.Make = domain.Normalize(f.Make)
	f.Condition = strings.TrimSpace(f.Condition)
	if f.MaxPriceToman < 0 || f.MaxMileageKM < 0 || f.YearFrom < 0 || f.YearTo < 0 {
		return SearchResult{}, errors.New("filters cannot be negative")
	}
	if f.YearFrom > 0 {
		if f.YearFrom < 1600 {
			f.YearCalendar = "jalali"
		} else {
			f.YearCalendar = "gregorian"
		}
	}
	if f.YearTo > 0 {
		calendar := "gregorian"
		if f.YearTo < 1600 {
			calendar = "jalali"
		}
		if f.YearCalendar != "" && f.YearCalendar != calendar {
			return SearchResult{}, errors.New("year range must use one calendar")
		}
		f.YearCalendar = calendar
	}
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Page <= 0 || f.Page > 10000 {
		f.Page = 1
	}
	return s.repo.Search(ctx, f)
}

func (s *Listings) Import(ctx context.Context, rows []domain.Listing) error {
	if len(rows) == 0 {
		return errors.New("feed has no rows")
	}
	for i := range rows {
		if err := rows[i].Prepare(); err != nil {
			return err
		}
	}
	return s.repo.SaveBatch(ctx, rows)
}

func (s *Listings) History(ctx context.Context, listingID int64) ([]domain.PriceObservation, error) {
	if listingID <= 0 {
		return nil, errors.New("listing ID must be positive")
	}
	return s.repo.History(ctx, listingID)
}

func (s *Listings) SourceStatuses(ctx context.Context) ([]domain.SourceStatus, error) {
	return s.repo.SourceStatuses(ctx)
}

func (s *Listings) CreateAlert(ctx context.Context, f SearchFilter) (domain.SavedAlert, error) {
	if f.MaxPriceToman < 0 || len(f.Query) > 120 || len(f.City) > 100 || len(f.Make) > 100 {
		return domain.SavedAlert{}, ErrInvalidAlertFilters
	}
	secret := make([]byte, 24)
	if _, err := rand.Read(secret); err != nil {
		return domain.SavedAlert{}, err
	}
	now := time.Now().UTC()
	a := domain.SavedAlert{
		Token: base64.RawURLEncoding.EncodeToString(secret), Query: domain.Normalize(f.Query),
		City: domain.Normalize(f.City), Make: domain.Normalize(f.Make), MaxPriceToman: f.MaxPriceToman,
		CreatedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour),
	}
	if err := s.repo.CreateAlert(ctx, a); err != nil {
		return domain.SavedAlert{}, err
	}
	return a, nil
}

func (s *Listings) AlertMatches(ctx context.Context, token string) (domain.SavedAlert, []domain.Listing, error) {
	if len(token) != 32 {
		return domain.SavedAlert{}, nil, ErrInvalidAlertToken
	}
	return s.repo.AlertMatches(ctx, token)
}
