package application

import (
	"context"
	"testing"

	"rahcar/backend/internal/domain"
)

type captureRepo struct{ filter SearchFilter }

func (r *captureRepo) Search(_ context.Context, f SearchFilter) (SearchResult, error) {
	r.filter = f
	return SearchResult{Items: []domain.Listing{}, Count: 0}, nil
}
func (*captureRepo) SaveBatch(context.Context, []domain.Listing) error { return nil }

func TestSearchNormalizesPersianAndKeepsYearCalendar(t *testing.T) {
	repo := &captureRepo{}
	service := NewListings(repo)
	_, err := service.Search(context.Background(), SearchFilter{Query: "كيا ٢٠٧", City: "تهران", YearFrom: 1400})
	if err != nil {
		t.Fatal(err)
	}
	if repo.filter.Query != "کیا 207" || repo.filter.YearCalendar != "jalali" {
		t.Fatalf("unexpected filter: %+v", repo.filter)
	}
	_, err = service.Search(context.Background(), SearchFilter{YearFrom: 1400, YearTo: 2020})
	if err == nil {
		t.Fatal("expected mixed calendar range to fail")
	}
}
