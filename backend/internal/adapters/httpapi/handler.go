package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"rahcar/backend/internal/application"
)

func Handler(service *application.Listings, allowedOrigin string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
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
