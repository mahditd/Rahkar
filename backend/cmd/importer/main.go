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
	path := flag.String("file", "", "CSV feed path")
	source := flag.String("source", "", "approved source identifier")
	permissionRef := flag.String("permission-ref", "", "reference to source permission")
	flag.Parse()
	if *path == "" || *source == "" || *permissionRef == "" {
		log.Fatal("-file, -source, and -permission-ref are required")
	}
	file, err := os.Open(*path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	rows, err := csvfeed.Parse(file, *source, *permissionRef, time.Now().UTC())
	if err != nil {
		log.Fatal(err)
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
	if err := application.NewListings(postgres.NewListings(pool)).Import(ctx, rows); err != nil {
		log.Fatal(err)
	}
	log.Printf("imported %d listings from %s", len(rows), *source)
}
