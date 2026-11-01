package csvfeed

import (
	"strings"
	"testing"
	"time"
)

func TestParseRejectsDuplicateIDsBeforeImport(t *testing.T) {
	header := strings.Join(required, ",") + "\n"
	row := "id-1,https://example.com/1,پژو ۲۰۷,پژو,۲۰۷,پانوراما,۱۴۰۲,jalali,۳۱۰۰۰,۸۲۰۰۰۰۰۰۰,تهران,بدون رنگ,2026-10-29T10:00:00Z\n"
	_, err := Parse(strings.NewReader(header+row+row), "dealer", "agreement-1", time.Now())
	if err == nil || !strings.Contains(err.Error(), "duplicate external ID") {
		t.Fatalf("expected duplicate external ID error, got %v", err)
	}
	listings, err := Parse(strings.NewReader(header+row), "dealer", "agreement-1", time.Now())
	if err != nil {
		t.Fatalf("valid Persian numeral row failed: %v", err)
	}
	if listings[0].ProductionYear != 1402 || listings[0].MileageKM != 31000 || listings[0].PriceToman != 820000000 {
		t.Fatalf("numeric fields parsed incorrectly: %+v", listings[0])
	}
}
