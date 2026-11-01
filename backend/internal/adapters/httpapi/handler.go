package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"rahcar/backend/internal/application"
)

func Handler(service *application.Listings, allowedOrigin string, ready func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := ready(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/v1/listings/{id}/history", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid listing ID"})
			return
		}
		points, err := service.History(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load price history"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": points})
	})
	mux.HandleFunc("GET /api/v1/sources", func(w http.ResponseWriter, r *http.Request) {
		statuses, err := service.SourceStatuses(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load source status"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": statuses})
	})
	mux.HandleFunc("POST /api/v1/alerts", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Query         string `json:"query"`
			City          string `json:"city"`
			Make          string `json:"make"`
			MaxPriceToman int64  `json:"max_price_toman"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid alert request"})
			return
		}
		alert, err := service.CreateAlert(r.Context(), application.SearchFilter{
			Query: input.Query, City: input.City, Make: input.Make, MaxPriceToman: input.MaxPriceToman,
		})
		if err != nil {
			if errors.Is(err, application.ErrInvalidAlertFilters) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid alert filters"})
			} else {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save alert"})
			}
			return
		}
		writeJSON(w, http.StatusCreated, alert)
	})
	mux.HandleFunc("GET /api/v1/alerts", func(w http.ResponseWriter, r *http.Request) {
		alert, matches, err := service.AlertMatches(r.Context(), r.Header.Get("X-Alert-Token"))
		if errors.Is(err, application.ErrAlertNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert not found"})
			return
		}
		if err != nil {
			if errors.Is(err, application.ErrInvalidAlertToken) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid alert token"})
			} else {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load alert"})
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"alert": alert, "items": matches, "count": len(matches)})
	})
	mux.HandleFunc("GET /api/v1/listings", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		maxPrice, err := parseInt64(params.Get("max_price_toman"), 0)
		if err != nil || maxPrice < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid max_price_toman"})
			return
		}
		maxMileage, err := parseInt(params.Get("max_mileage_km"), 0)
		if err != nil || maxMileage < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid max_mileage_km"})
			return
		}
		yearFrom, err := parseInt(params.Get("year_from"), 0)
		if err != nil || yearFrom < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid year_from"})
			return
		}
		yearTo, err := parseInt(params.Get("year_to"), 0)
		if err != nil || yearTo < 0 || (yearFrom > 0 && yearTo > 0 && yearTo < yearFrom) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid year_to"})
			return
		}
		page, err := parseInt(params.Get("page"), 1)
		if err != nil || page < 1 || page > 10000 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "page must be between 1 and 10000"})
			return
		}
		limit, err := parseInt(params.Get("limit"), 20)
		if err != nil || limit < 1 || limit > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be between 1 and 100"})
			return
		}
		if len(params.Get("q")) > 120 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query is too long"})
			return
		}
		if yearFrom > 0 && yearTo > 0 && (yearFrom < 1600) != (yearTo < 1600) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "year range must use one calendar"})
			return
		}
		result, err := service.Search(r.Context(), application.SearchFilter{
			Query: params.Get("q"), City: params.Get("city"), Make: params.Get("make"),
			MaxPriceToman: maxPrice, MaxMileageKM: maxMileage, Condition: params.Get("condition"),
			YearFrom: yearFrom, YearTo: yearTo, Page: page, Limit: limit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load listings"})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && strings.EqualFold(origin, allowedOrigin) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Alert-Token")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func parseInt(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func parseInt64(value string, fallback int64) (int64, error) {
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
