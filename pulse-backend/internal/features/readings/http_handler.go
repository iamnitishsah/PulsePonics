package readings

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pulse-backend/internal/features/devices"
	"pulse-backend/internal/platform/apperr"
	"pulse-backend/internal/platform/authctx"
	"pulse-backend/internal/platform/httpx"
)

// DeviceAccess is the slice of the device registry the handlers need.
type DeviceAccess interface {
	Authorize(ctx context.Context, userID int64, deviceID string) error
	ByAPIKey(ctx context.Context, key string) (*devices.Device, error)
}

type Handler struct {
	service *Service
	devices DeviceAccess
}

func NewHandler(service *Service, devices DeviceAccess) *Handler {
	return &Handler{service: service, devices: devices}
}

func (h *Handler) Register(mux *http.ServeMux, auth func(http.HandlerFunc) http.HandlerFunc) {
	// Device ingestion: authenticated with the device's API key, not a user token.
	mux.HandleFunc("POST /api/readings", h.ingest)

	mux.HandleFunc("GET /api/devices/{deviceID}/readings", auth(h.history))
	mux.HandleFunc("GET /api/devices/{deviceID}/readings/latest", auth(h.latest))
	mux.HandleFunc("GET /api/devices/{deviceID}/readings/recent", auth(h.recent))
	mux.HandleFunc("GET /api/metrics", h.metrics)
}

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.Header.Get("X-Device-Key"))
	if key == "" {
		httpx.WriteError(w, r, apperr.Unauthorized("missing X-Device-Key header"))
		return
	}
	device, err := h.devices.ByAPIKey(r.Context(), key)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if device == nil {
		httpx.WriteError(w, r, apperr.Unauthorized("invalid device key"))
		return
	}

	var in Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if in.DeviceID == "" {
		in.DeviceID = device.DeviceID
	}
	if in.DeviceID != device.DeviceID {
		httpx.WriteError(w, r, apperr.Forbidden("device_id does not match the device key"))
		return
	}

	reading, err := h.service.Ingest(r.Context(), in)
	switch {
	case errors.Is(err, ErrDuplicate):
		httpx.WriteError(w, r, apperr.Conflict("a reading with this recorded_at is already stored"))
	case err != nil:
		httpx.WriteError(w, r, err)
	default:
		httpx.WriteJSON(w, http.StatusCreated, reading)
	}
}

// authorize checks the device belongs to the caller and returns its ID.
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) (string, bool) {
	deviceID := r.PathValue("deviceID")
	if err := h.devices.Authorize(r.Context(), authctx.UserID(r.Context()), deviceID); err != nil {
		httpx.WriteError(w, r, err)
		return "", false
	}
	return deviceID, true
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	deviceID, ok := h.authorize(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	var (
		query HistoryQuery
		err   error
	)
	if query.From, err = parseTime(q.Get("from"), "from"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if query.To, err = parseTime(q.Get("to"), "to"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if query.MaxPoints, err = parseInt(q.Get("max_points"), "max_points"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	history, err := h.service.History(r.Context(), deviceID, query)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, history)
}

func (h *Handler) latest(w http.ResponseWriter, r *http.Request) {
	deviceID, ok := h.authorize(w, r)
	if !ok {
		return
	}
	reading, err := h.service.Latest(r.Context(), deviceID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if reading == nil {
		httpx.WriteError(w, r, apperr.NotFound("no readings yet for this device"))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, reading)
}

func (h *Handler) recent(w http.ResponseWriter, r *http.Request) {
	deviceID, ok := h.authorize(w, r)
	if !ok {
		return
	}
	limit, err := parseInt(r.URL.Query().Get("limit"), "limit")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	readings, err := h.service.Recent(r.Context(), deviceID, limit)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, readings)
}

func (h *Handler) metrics(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, Metrics)
}

func parseTime(s, field string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, apperr.Invalid(field, field+" must be an RFC3339 time, e.g. 2026-10-24T08:20:00Z")
	}
	return t, nil
}

func parseInt(s, field string) (int, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, apperr.Invalid(field, field+" must be a non-negative integer")
	}
	return n, nil
}
