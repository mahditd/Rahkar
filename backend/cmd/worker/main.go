package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"rahcar/backend/internal/adapters/csvfeed"
	"rahcar/backend/internal/adapters/postgres"
	"rahcar/backend/internal/application"
)

const maxFeedBytes = 8 << 20

type sourceConfig struct {
	Name          string `json:"name"`
	PermissionRef string `json:"permission_ref"`
	URL           string `json:"url"`
	TokenEnv      string `json:"token_env"`
}

type config struct {
	IntervalMinutes int            `json:"interval_minutes"`
	Sources         []sourceConfig `json:"sources"`
}

func main() {
	path := flag.String("config", "", "JSON feed configuration path")
	once := flag.Bool("once", false, "run each source once and exit")
	flag.Parse()
	if *path == "" {
		log.Fatal("-config is required")
	}
	data, err := os.ReadFile(*path)
	if err != nil {
		log.Fatal(err)
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatal(err)
	}
	if cfg.IntervalMinutes < 5 || cfg.IntervalMinutes > 1440 || len(cfg.Sources) == 0 {
		log.Fatal("interval must be 5-1440 minutes and at least one source is required")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	repo := postgres.NewListings(pool)
	service := application.NewListings(repo)
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return errors.New("feed redirect must remain HTTPS")
		}
		if len(via) >= 3 {
			return errors.New("too many feed redirects")
		}
		return nil
	}}
	for {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := repo.DeleteExpiredAlerts(cleanupCtx); err != nil {
			log.Printf("alert_cleanup_error=%v", err)
		}
		cleanupCancel()
		for _, source := range cfg.Sources {
			started := time.Now().UTC()
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			count, runErr := importSource(ctx, client, service, source)
			cancel()
			recordCtx, recordCancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := repo.RecordImportRun(recordCtx, source.Name, started, count, runErr); err != nil {
				log.Printf("source=%s record_status_error=%v", source.Name, err)
			}
			recordCancel()
			if runErr != nil {
				log.Printf("source=%s status=failed error=%v", source.Name, runErr)
			} else {
				log.Printf("source=%s status=success imported=%d", source.Name, count)
			}
		}
		if *once {
			return
		}
		time.Sleep(time.Duration(cfg.IntervalMinutes) * time.Minute)
	}
}

func importSource(ctx context.Context, client *http.Client, service *application.Listings, source sourceConfig) (int, error) {
	if source.Name == "" || source.PermissionRef == "" {
		return 0, errors.New("source name and permission reference are required")
	}
	u, err := url.Parse(source.URL)
	if err != nil {
		return 0, err
	}
	var data []byte
	switch u.Scheme {
	case "file":
		clean := path.Clean(u.Path)
		if u.Host != "" || !strings.HasPrefix(clean, "/data/") {
			return 0, errors.New("file feeds must be under /data")
		}
		data, err = os.ReadFile(clean)
	case "https":
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
		if reqErr != nil {
			return 0, reqErr
		}
		if source.TokenEnv != "" {
			token := os.Getenv(source.TokenEnv)
			if token == "" {
				return 0, fmt.Errorf("token environment variable %s is empty", source.TokenEnv)
			}
			req.Header.Set("Authorization", "Bearer "+token)
		}
		response, reqErr := client.Do(req)
		if reqErr != nil {
			return 0, reqErr
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("feed returned HTTP %d", response.StatusCode)
		}
		data, err = io.ReadAll(io.LimitReader(response.Body, maxFeedBytes+1))
	default:
		return 0, errors.New("feed URL must use HTTPS or a /data file")
	}
	if err != nil {
		return 0, err
	}
	if len(data) > maxFeedBytes {
		return 0, errors.New("feed exceeds 8 MiB limit")
	}
	rows, err := csvfeed.Parse(bytes.NewReader(data), source.Name, source.PermissionRef, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	if err := service.Import(ctx, rows); err != nil {
		return 0, err
	}
	return len(rows), nil
}
