package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"rahcar/backend/internal/adapters/httpapi"
	"rahcar/backend/internal/adapters/nemuda"
	"rahcar/backend/internal/adapters/postgres"
	"rahcar/backend/internal/application"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	origin := os.Getenv("CORS_ORIGIN")
	if origin == "" {
		origin = "http://localhost:5173"
	}
	server := &http.Server{
		Addr: ":" + port, Handler: httpapi.Handler(application.NewListings(postgres.NewListings(pool)), nemuda.NewClient(), origin, pool.Ping),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("API listening on :%s", port)
	log.Fatal(server.ListenAndServe())
}
