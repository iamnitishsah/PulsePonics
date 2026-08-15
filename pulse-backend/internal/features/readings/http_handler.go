package readings

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
)

type ReadingHandler struct {
	service *ReadingService
}

func NewHTTPHandler(s *ReadingService) *ReadingHandler {
	return &ReadingHandler{service: s}
}

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

func (h *ReadingHandler) CreateReading(w http.ResponseWriter, r *http.Request) {
	var input NewReadingInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	id, err := h.service.SubmitReading(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrMissingDeviceID),
			errors.Is(err, ErrNoSensorData),
			errors.Is(err, ErrInvalidPH),
			errors.Is(err, ErrInvalidEC),
			errors.Is(err, ErrInvalidWaterTemp),
			errors.Is(err, ErrInvalidAirTemp),
			errors.Is(err, ErrInvalidHumidity):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			log.Printf("CreateReading: unexpected error: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to save reading")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (h *ReadingHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	deviceID := q.Get("device_id")

	from := parseTimeOrZero(q.Get("from"))
	to := parseTimeOrZero(q.Get("to"))
	limit := parseIntOrZero(q.Get("limit"))
	offset := parseIntOrZero(q.Get("offset"))

	readings, err := h.service.GetHistory(r.Context(), deviceID, from, to, limit, offset)
	if err != nil {
		if errors.Is(err, ErrMissingDeviceID) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch readings")
		return
	}

	writeJSON(w, http.StatusOK, readings)
}

func (h *ReadingHandler) GetLatest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	deviceID := q.Get("device_id")

	reading, err := h.service.GetLatest(r.Context(), deviceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrMissingDeviceID):
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
