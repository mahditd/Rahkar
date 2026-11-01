package application

import (
	"context"
	"errors"
	"strings"

	"rahcar/backend/internal/domain"
)

type SearchFilter struct {
	Query         string
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
}

type Listings struct{ repo Repository }

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
