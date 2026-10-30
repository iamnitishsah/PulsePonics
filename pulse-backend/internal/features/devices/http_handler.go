package devices

import (
	"context"
	"net/http"

	"pulse-backend/internal/platform/authctx"
	"pulse-backend/internal/platform/httpx"
)

// Summary is the live state shown on the device list: the latest reading
// (nil if none yet) and how many alerts are open.
type Summary struct {
	Latest     any `json:"latest"`
	OpenAlerts int `json:"open_alerts"`
}

// SummaryProvider is implemented in main by combining the readings and
// alerts features, so this package doesn't depend on either.
type SummaryProvider interface {
	Summary(ctx context.Context, deviceID string) (Summary, error)
}

type deviceWithSummary struct {
	Device
	Summary
}

type Handler struct {
	service   *Service
	summaries SummaryProvider
}

func NewHandler(service *Service, summaries SummaryProvider) *Handler {
	return &Handler{service: service, summaries: summaries}
}

func (h *Handler) Register(mux *http.ServeMux, auth func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/devices", auth(h.list))
	mux.HandleFunc("POST /api/devices", auth(h.create))
	mux.HandleFunc("GET /api/devices/{deviceID}", auth(h.get))
	mux.HandleFunc("PATCH /api/devices/{deviceID}", auth(h.rename))
	mux.HandleFunc("DELETE /api/devices/{deviceID}", auth(h.delete))
	mux.HandleFunc("POST /api/devices/{deviceID}/api-key", auth(h.rotateKey))
}

func (h *Handler) withSummary(ctx context.Context, d Device) (deviceWithSummary, error) {
	s, err := h.summaries.Summary(ctx, d.DeviceID)
	return deviceWithSummary{Device: d, Summary: s}, err
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	devices, err := h.service.List(r.Context(), authctx.UserID(r.Context()))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]deviceWithSummary, 0, len(devices))
	for _, d := range devices {
		item, err := h.withSummary(r.Context(), d)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		out = append(out, item)
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	d, err := h.service.Get(r.Context(), authctx.UserID(r.Context()), r.PathValue("deviceID"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	item, err := h.withSummary(r.Context(), d)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in CreateInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	d, err := h.service.Create(r.Context(), authctx.UserID(r.Context()), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, d)
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	var in UpdateInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	d, err := h.service.Rename(r.Context(), authctx.UserID(r.Context()), r.PathValue("deviceID"), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), authctx.UserID(r.Context()), r.PathValue("deviceID")); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteNoContent(w)
}

func (h *Handler) rotateKey(w http.ResponseWriter, r *http.Request) {
	d, err := h.service.RotateKey(r.Context(), authctx.UserID(r.Context()), r.PathValue("deviceID"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, d)
}
