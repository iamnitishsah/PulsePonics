// Package httpx holds the HTTP plumbing shared by every feature: JSON
// encoding, error responses, and middleware.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"pulse-backend/internal/platform/apperr"
)

const maxBodyBytes = 1 << 20 // 1 MiB is far more than any request here needs

type errorResponse struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WriteError maps an *apperr.Error to its status code. Any other error is
// logged and reported as a generic 500 so internals never leak to clients.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if e, ok := apperr.As(err); ok {
		WriteJSON(w, statusFor(e.Kind), errorResponse{Error: e.Message, Field: e.Field})
		return
	}
	log.Printf("%s %s: internal error: %v", r.Method, r.URL.Path, err)
	WriteJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
}

func statusFor(kind apperr.Kind) int {
	switch kind {
	case apperr.KindInvalid:
		return http.StatusBadRequest
	case apperr.KindUnauthorized:
		return http.StatusUnauthorized
	case apperr.KindForbidden:
		return http.StatusForbidden
	case apperr.KindNotFound:
		return http.StatusNotFound
	case apperr.KindConflict:
		return http.StatusConflict
	case apperr.KindTooManyRequests:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// DecodeJSON reads a JSON body (max 1 MiB) into dst.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		switch {
		case errors.As(err, &tooLarge):
			return apperr.Invalid("", "request body too large")
		case errors.Is(err, io.EOF):
			return apperr.Invalid("", "request body is empty")
		default:
			return apperr.Invalid("", "invalid JSON body")
		}
	}
	return nil
}
