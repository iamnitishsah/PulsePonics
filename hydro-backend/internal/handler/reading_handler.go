// internal/handler/reading_handler.go
package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"hydro-backend/internal/domain"
	"hydro-backend/internal/service"
)

// ReadingHandler holds only what it needs to talk to the service layer.
// Handlers should never import "repository" or talk to *sqlx.DB directly —
// that boundary keeps HTTP concerns (status codes, JSON encoding) separate
// from business logic (validation, defaults), so each is easy to reason
// about and test independently.
type ReadingHandler struct {
	service *service.ReadingService
}

func NewReadingHandler(s *service.ReadingService) *ReadingHandler {
	return &ReadingHandler{service: s}
}

// errorResponse is the consistent JSON shape for any failed request —
// having one shared error format makes the frontend's error handling
// (Sprint 4, React dashboard) simple and predictable.
type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// CreateReading handles POST /readings — this is the endpoint your ESP32
// (via an MQTT-to-HTTP bridge, or directly once you add MQTT in Phase 2)
// or a test curl command will hit to submit a new sensor reading.
func (h *ReadingHandler) CreateReading(w http.ResponseWriter, r *http.Request) {
	var input domain.NewReadingInput

	// Decode JSON body straight into the domain type. If the body is
	// malformed JSON, this fails immediately — no need for separate
	// manual validation of "is this valid JSON".
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	id, err := h.service.SubmitReading(r.Context(), input)
	if err != nil {
		// Distinguish validation errors (client's fault → 400) from
		// unexpected failures (server's fault → 500) using errors.Is
		// against the sentinel errors declared in the service layer.
		switch {
		case errors.Is(err, service.ErrMissingDeviceID),
			errors.Is(err, service.ErrNoSensorData),
			errors.Is(err, service.ErrInvalidPH),
			errors.Is(err, service.ErrInvalidEC),
			errors.Is(err, service.ErrInvalidWaterTemp),
			errors.Is(err, service.ErrInvalidAirTemp),
			errors.Is(err, service.ErrInvalidHumidity):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			// Log the real error server-side (never send raw internal
			// error details to the client — that can leak DB schema/
			// connection info — but you need to see it to debug).
			log.Printf("CreateReading: unexpected error: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to save reading")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

// GetHistory handles GET /readings?device_id=X&from=RFC3339&to=RFC3339&limit=N&offset=N
// Only device_id is required; the rest have defaults applied in the service
// layer (last 24h, limit 100).
func (h *ReadingHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	deviceID := q.Get("device_id")

	from := parseTimeOrZero(q.Get("from"))
	to := parseTimeOrZero(q.Get("to"))
	limit := parseIntOrZero(q.Get("limit"))
	offset := parseIntOrZero(q.Get("offset"))

	readings, err := h.service.GetHistory(r.Context(), deviceID, from, to, limit, offset)
	if err != nil {
		if errors.Is(err, service.ErrMissingDeviceID) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch readings")
		return
	}

	writeJSON(w, http.StatusOK, readings)
}

// GetLatest handles GET /readings/latest?device_id=X
func (h *ReadingHandler) GetLatest(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")

	reading, err := h.service.GetLatest(r.Context(), deviceID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMissingDeviceID):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, sql.ErrNoRows):
			writeError(w, http.StatusNotFound, "no readings found for this device")
		default:
			writeError(w, http.StatusInternalServerError, "failed to fetch latest reading")
		}
		return
	}

	writeJSON(w, http.StatusOK, reading)
}

// --- small local helpers, kept private to this file since they're only
// used for parsing query params here ---

func parseTimeOrZero(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseIntOrZero(s string) int {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
