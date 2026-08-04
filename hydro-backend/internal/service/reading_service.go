// internal/service/reading_service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hydro-backend/internal/domain"
	"hydro-backend/internal/repository"
)

// Sentinel errors — declared once, checked with errors.Is() by callers
// (handlers). This is the idiomatic Go way to let calling code distinguish
// "bad input" (→ HTTP 400) from "something broke" (→ HTTP 500) without
// string-matching error messages, which is fragile.
var (
	ErrMissingDeviceID  = errors.New("device_id is required")
	ErrNoSensorData     = errors.New("at least one sensor value must be provided")
	ErrInvalidPH        = errors.New("ph must be between 0 and 14")
	ErrInvalidEC        = errors.New("ec_ms_cm cannot be negative")
	ErrInvalidWaterTemp = errors.New("water_temp_c must be between -10 and 60")
	ErrInvalidAirTemp   = errors.New("air_temp_c must be between -20 and 60")
	ErrInvalidHumidity  = errors.New("humidity_pct must be between 0 and 100")
)

// Broadcaster is implemented by anything that wants to be notified when a
// new reading is saved — currently just the WebSocket Hub. Defining this
// as an interface *here* (in service, the consumer) rather than importing
// the concrete ws.Hub type avoids a dependency cycle: service would import
// ws, and ws's handler already needs to reach the service layer's data
// shape indirectly. This is the same "accept interfaces" pattern as
// ReadingRepository — service doesn't know or care that WebSockets are
// involved, only that "something wants to know about new readings".
type Broadcaster interface {
	Broadcast(reading domain.Reading)
}

// ReadingService holds business logic. It depends on the ReadingRepository
// *interface* (Step 1.5), not the concrete MySQL struct — this is the
// payoff of that interface: this service doesn't know or care that MySQL
// is involved at all.
type ReadingService struct {
	repo        repository.ReadingRepository
	broadcaster Broadcaster // may be nil — see NewReadingService
}

// NewReadingService constructs a service with no broadcaster attached.
// Existing callers (and any tests) keep working unchanged.
func NewReadingService(repo repository.ReadingRepository) *ReadingService {
	return &ReadingService{repo: repo}
}

// SetBroadcaster attaches a Broadcaster after construction. We use a
// setter here instead of requiring it in NewReadingService because the
// WebSocket Hub and the service are both created independently in main.go,
// and this keeps the constructor from needing every future "notify on
// insert" consumer as a required argument.
func (s *ReadingService) SetBroadcaster(b Broadcaster) {
	s.broadcaster = b
}

// SubmitReading validates an incoming payload and persists it. This is where
// Sprint 4's "biological safety threshold" alerting logic will eventually
// hook in — right after a successful insert, you'll add a check like
// "if reading breaches threshold, trigger notification". For Phase 1, we
// keep it to validation + insert.
func (s *ReadingService) SubmitReading(ctx context.Context, input domain.NewReadingInput) (int64, error) {
	if err := validate(input); err != nil {
		return 0, err
	}

	// If the device didn't send a timestamp (e.g. a quick curl test),
	// default to "now" rather than rejecting — server-side timestamping
	// is a reasonable fallback for a hobby/research device that may not
	// have NTP-synced time.
	if input.RecordedAt.IsZero() {
		input.RecordedAt = time.Now()
	}

	id, err := s.repo.Insert(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("service: submit reading: %w", err)
	}

	// Notify any connected WebSocket clients about this new reading. We
	// build the broadcast payload from what we already have in memory
	// (input + new id + current time) rather than re-querying the DB for
	// the row we just inserted — saves a round trip for something that's
	// purely for real-time display.
	//
	// We check for nil because broadcaster is optional (see SetBroadcaster) —
	// this keeps SubmitReading usable in tests or contexts where no
	// WebSocket hub exists at all.
	if s.broadcaster != nil {
		s.broadcaster.Broadcast(domain.Reading{
			ID:          id,
			DeviceID:    input.DeviceID,
			PH:          input.PH,
			ECmScm:      input.ECmScm,
			WaterTempC:  input.WaterTempC,
			AirTempC:    input.AirTempC,
			HumidityPct: input.HumidityPct,
			PressureHPa: input.PressureHPa,
			RecordedAt:  input.RecordedAt,
			CreatedAt:   time.Now(),
		})
	}

	return id, nil
}

// GetHistory returns readings for a device within a time range. Defaults
// are applied here (not in the handler) so this logic is testable and
// reusable regardless of which transport calls it (HTTP now, maybe a CLI
// or gRPC endpoint later).
func (s *ReadingService) GetHistory(
	ctx context.Context,
	deviceID string,
	from, to time.Time,
	limit, offset int,
) ([]domain.Reading, error) {
	if deviceID == "" {
		return nil, ErrMissingDeviceID
	}

	// Sensible defaults: if no range given, default to "last 24 hours".
	if to.IsZero() {
		to = time.Now()
	}
	if from.IsZero() {
		from = to.Add(-24 * time.Hour)
	}
	if limit <= 0 || limit > 1000 {
		limit = 100 // guard against a client requesting an unbounded/huge result set
	}

	readings, err := s.repo.GetByDevice(ctx, deviceID, from, to, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("service: get history: %w", err)
	}

	return readings, nil
}

// GetLatest returns the most recent reading for a device.
func (s *ReadingService) GetLatest(ctx context.Context, deviceID string) (*domain.Reading, error) {
	if deviceID == "" {
		return nil, ErrMissingDeviceID
	}

	reading, err := s.repo.GetLatest(ctx, deviceID)
	if err != nil {
		return nil, err // let handler check for sql.ErrNoRows specifically
	}

	return reading, nil
}

// validate applies basic sanity checks before anything touches the
// database. Kept as a standalone function (not a method) since it has no
// need for service state — a small Go convention: don't make something a
// method unless it actually uses the receiver.
func validate(input domain.NewReadingInput) error {
	if input.DeviceID == "" {
		return ErrMissingDeviceID
	}

	if input.PH == nil && input.ECmScm == nil && input.WaterTempC == nil &&
		input.AirTempC == nil && input.HumidityPct == nil && input.PressureHPa == nil {
		return ErrNoSensorData
	}

	if input.PH != nil && (*input.PH < 0 || *input.PH > 14) {
		return ErrInvalidPH
	}

	// EC (electrical conductivity) can't physically be negative — it
	// measures how well the nutrient solution conducts electricity, which
	// is a magnitude, not a signed quantity.
	if input.ECmScm != nil && *input.ECmScm < 0 {
		return ErrInvalidEC
	}

	// Wide but physically reasonable bounds — these aren't tight
	// "ideal growing range" thresholds (that's Sprint 4's alerting logic,
	// which flags *unsafe* values while still storing them). This is just
	// catching sensor glitches / bad test data, e.g. a disconnected probe
	// reporting -127°C, a known DS18B20 failure signature.
	if input.WaterTempC != nil && (*input.WaterTempC < -10 || *input.WaterTempC > 60) {
		return ErrInvalidWaterTemp
	}
	if input.AirTempC != nil && (*input.AirTempC < -20 || *input.AirTempC > 60) {
		return ErrInvalidAirTemp
	}
	if input.HumidityPct != nil && (*input.HumidityPct < 0 || *input.HumidityPct > 100) {
		return ErrInvalidHumidity
	}

	return nil
}
