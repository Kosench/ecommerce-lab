package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Kosench/ecommerce-lab/platform/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type HealthHandler struct {
	db     *pgxpool.Pool
	logger logger.Logger
}

func NewHealthHandler(db *pgxpool.Pool, log logger.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: log.With(zap.String("component", "health")),
	}
}

type healthResponse struct {
	Status  string            `json:"status"`
	Version string            `json:"version"`
	Time    string            `json:"time"`
	Checks  map[string]string `json:"checks,omitempty"`
}

func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:  "alive",
		Version: "1.0.0",
		Time:    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		h.logger.Error("database not ready",
			zap.Error(err),
		)

		resp := healthResponse{
			Status: "unhealthy",
			Time:   time.Now().Format(time.RFC3339),
			Checks: map[string]string{"database": "failed"},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := healthResponse{
		Status:  "ready",
		Version: "1.0.0",
		Time:    time.Now().Format(time.RFC3339),
		Checks:  map[string]string{"database": "ok"},
	}

	h.logger.Debug("readiness check passed",
		zap.String("database", "ok"),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
