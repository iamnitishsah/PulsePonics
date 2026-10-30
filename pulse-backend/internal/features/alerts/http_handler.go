package alerts

import (
	"context"
	"net/http"
	"strconv"

	"pulse-backend/internal/platform/apperr"
	"pulse-backend/internal/platform/authctx"
	"pulse-backend/internal/platform/httpx"
)

type DeviceAuthorizer interface {
	Authorize(ctx context.Context, userID int64, deviceID string) error
}

type Handler struct {
	service *Service
	devices DeviceAuthorizer
}

func NewHandler(service *Service, devices DeviceAuthorizer) *Handler {
	return &Handler{service: service, devices: devices}
}

func (h *Handler) Register(mux *http.ServeMux, auth func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/devices/{deviceID}/thresholds", auth(h.getThresholds))
	mux.HandleFunc("PUT /api/devices/{deviceID}/thresholds", auth(h.putThresholds))
	mux.HandleFunc("GET /api/alerts", auth(h.list))
	mux.HandleFunc("POST /api/alerts/{alertID}/acknowledge", auth(h.acknowledge))
}

type thresholdsBody struct {
	Thresholds []Threshold `json:"thresholds"`
}

func (h *Handler) getThresholds(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("deviceID")
	if err := h.devices.Authorize(r.Context(), authctx.UserID(r.Context()), deviceID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ts, err := h.service.Thresholds(r.Context(), deviceID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, thresholdsBody{Thresholds: ts})
}

func (h *Handler) putThresholds(w http.ResponseWriter, r *http.Request) {
	userID := authctx.UserID(r.Context())
	deviceID := r.PathValue("deviceID")
	if err := h.devices.Authorize(r.Context(), userID, deviceID); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var body thresholdsBody
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ts, err := h.service.UpdateThresholds(r.Context(), userID, deviceID, body.Thresholds)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, thresholdsBody{Thresholds: ts})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ListFilter{
		UserID:   authctx.UserID(r.Context()),
		DeviceID: q.Get("device_id"),
		Status:   q.Get("status"),
	}
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			httpx.WriteError(w, r, apperr.Invalid("limit", "limit must be an integer"))
			return
		}
		f.Limit = n
	}
	alerts, err := h.service.List(r.Context(), f)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, alerts)
}

func (h *Handler) acknowledge(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("alertID"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, r, apperr.NotFound("alert not found"))
		return
	}
	a, err := h.service.Acknowledge(r.Context(), authctx.UserID(r.Context()), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a)
}
