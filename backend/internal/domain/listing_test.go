package domain

import (
	"testing"
	"time"
)

func TestNormalizePersianVariants(t *testing.T) {
	got := Normalize("  پژو ٢٠٧  كيا  ")
	want := "پژو 207 کیا"
	if got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func TestListingRequiresPermissionAndValidPrice(t *testing.T) {
	now := time.Now()
	l := Listing{SourceName: "dealer", ExternalID: "1", SourceURL: "https://example.com/1", Title: "car", Make: "پژو", Model: "۲۰۷", ProductionYear: 1402, YearCalendar: "jalali", MileageKM: 1000, PriceToman: 1, City: "تهران", PublishedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := l.Prepare(); err == nil {
		t.Fatal("expected missing permission reference to fail")
	}
	l.PermissionRef = "agreement-1"
	l.PriceToman = -1
	if err := l.Prepare(); err == nil {
		t.Fatal("expected negative price to fail")
	}
	l.PriceToman = 100000000
	if err := l.Prepare(); err != nil {
		t.Fatalf("valid listing failed: %v", err)
	}
	if l.Model != "207" {
		t.Fatalf("model was not normalized: %q", l.Model)
	}
}
