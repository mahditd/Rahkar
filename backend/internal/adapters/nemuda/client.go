package nemuda

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"rahcar/backend/internal/domain"
)

const indexURL = "https://api.nemuda.com/api/market/vehicles/index"
const maxResponseBytes = 8 << 20

type Client struct {
	httpClient *http.Client
	mu         sync.Mutex
	snapshot   domain.MarketSnapshot
	fetchedAt  time.Time
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 12 * time.Second}}
}

func (c *Client) Overview(ctx context.Context) (domain.MarketSnapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.fetchedAt.IsZero() && time.Since(c.fetchedAt) < 6*time.Hour {
		return c.snapshot, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return domain.MarketSnapshot{}, err
	}
	res, err := c.httpClient.Do(req)
	if err == nil {
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			err = fmt.Errorf("Nemuda returned HTTP %d", res.StatusCode)
		} else {
			var body []byte
			body, err = io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
			if err == nil && len(body) > maxResponseBytes {
				err = errors.New("Nemuda response exceeds 8 MiB")
			}
			if err == nil {
				var snapshot domain.MarketSnapshot
				snapshot, err = parse(body)
				if err == nil {
					c.snapshot = snapshot
					c.fetchedAt = time.Now()
					return snapshot, nil
				}
			}
		}
	}
	if !c.fetchedAt.IsZero() {
		return c.snapshot, nil
	}
	return domain.MarketSnapshot{}, err
}

type year struct {
	ModelYear  int   `json:"modelYear"`
	DataPoints int   `json:"dataPoints"`
	Price      int64 `json:"price"`
	MileageKM  int   `json:"mileageKm"`
}

func parse(data []byte) (domain.MarketSnapshot, error) {
	var index struct {
		ComputedAt time.Time `json:"computedAt"`
		Categories []struct {
			Category string `json:"category"`
			Brands   []struct {
				Slug       string `json:"slug"`
				DataPoints int    `json:"dataPoints"`
				Models     []struct {
					Slug       string `json:"slug"`
					NameFa     string `json:"nameFa"`
					DataPoints int    `json:"dataPoints"`
					Years      []year `json:"years"`
				} `json:"models"`
			} `json:"brands"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return domain.MarketSnapshot{}, err
	}
	if index.ComputedAt.IsZero() {
		return domain.MarketSnapshot{}, errors.New("Nemuda response has no computation date")
	}
	result := domain.MarketSnapshot{ComputedAt: index.ComputedAt, Items: make([]domain.MarketPrice, 0, 12)}
	for _, category := range index.Categories {
		if category.Category != "car" {
			continue
		}
		sort.Slice(category.Brands, func(i, j int) bool { return category.Brands[i].DataPoints > category.Brands[j].DataPoints })
		for _, brand := range category.Brands {
			if brand.Slug == "" || len(result.Items) >= 12 {
				break
			}
			added := false
			sort.Slice(brand.Models, func(i, j int) bool { return brand.Models[i].DataPoints > brand.Models[j].DataPoints })
			for _, model := range brand.Models {
				if model.Slug == "" || model.NameFa == "" {
					continue
				}
				sort.Slice(model.Years, func(i, j int) bool { return model.Years[i].DataPoints > model.Years[j].DataPoints })
				for _, y := range model.Years {
					if y.Price <= 0 || y.DataPoints < 30 || y.ModelYear <= 0 {
						continue
					}
					result.Items = append(result.Items, domain.MarketPrice{
						Name: model.NameFa, ModelYear: y.ModelYear, PriceToman: y.Price,
						MileageKM: y.MileageKM, DataPoints: y.DataPoints,
						SourceURL: fmt.Sprintf("https://nemuda.com/car/%s/%s/%d", brand.Slug, model.Slug, y.ModelYear),
					})
					added = true
					break
				}
				if added {
					break
				}
			}
		}
	}
	if len(result.Items) == 0 {
		return domain.MarketSnapshot{}, errors.New("Nemuda response has no car prices")
	}
	return result, nil
}
