package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"rahcar/backend/internal/adapters/csvfeed"
	"rahcar/backend/internal/adapters/postgres"
	"rahcar/backend/internal/application"
)

func main() {
	started := time.Now().UTC()
	path := flag.String("file", "", "CSV feed path")
	source := flag.String("source", "", "approved source identifier")
	permissionRef := flag.String("permission-ref", "", "reference to source permission")
	flag.Parse()
	if *path == "" || *source == "" || *permissionRef == "" {
		log.Fatal("-file, -source, and -permission-ref are required")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	repo := postgres.NewListings(pool)
	file, err := os.Open(*path)
	if err != nil {
		_ = repo.RecordImportRun(ctx, *source, started, 0, err)
		log.Fatal(err)
	}
	defer file.Close()
	rows, err := csvfeed.Parse(file, *source, *permissionRef, time.Now().UTC())
	if err != nil {
		_ = repo.RecordImportRun(ctx, *source, started, 0, err)
		log.Fatal(err)
	}
	if err := application.NewListings(repo).Import(ctx, rows); err != nil {
		_ = repo.RecordImportRun(ctx, *source, started, 0, err)
		log.Fatal(err)
	}
	if err := repo.RecordImportRun(ctx, *source, started, len(rows), nil); err != nil {
		log.Fatal(err)
	}
	log.Printf("imported %d listings from %s", len(rows), *source)
}
