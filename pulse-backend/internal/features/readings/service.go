package readings

import (
	"context"
	"fmt"
	"time"
)

type Broadcaster interface {
	Broadcast(payload any)
}

type ReadingService struct {
	repo        ReadingRepository
	broadcaster Broadcaster
}

func NewService(repo ReadingRepository) *ReadingService {
	return &ReadingService{repo: repo}
}

func (s *ReadingService) SetBroadcaster(b Broadcaster) {
	s.broadcaster = b
}

func (s *ReadingService) SubmitReading(ctx context.Context, input NewReadingInput) (int64, error) {
	if err := validate(input); err != nil {
		return 0, err
	}

	if input.RecordedAt.IsZero() {
		input.RecordedAt = time.Now()
	}

	id, err := s.repo.Insert(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("service: submit reading: %w", err)
	}

	if s.broadcaster != nil {
		s.broadcaster.Broadcast(Reading{
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

func (s *ReadingService) GetHistory(ctx context.Context, deviceID string, from, to time.Time, limit, offset int) ([]Reading, error) {
	if deviceID == "" {
		return nil, ErrMissingDeviceID
	}

	if to.IsZero() {
		to = time.Now()
	}
	if from.IsZero() {
		from = to.Add(-24 * time.Hour)
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	readings, err := s.repo.GetByDevice(ctx, deviceID, from, to, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("service: get history: %w", err)
	}

	return readings, nil
}

func (s *ReadingService) GetLatest(ctx context.Context, deviceID string) (*Reading, error) {
	if deviceID == "" {
		return nil, ErrMissingDeviceID
	}

	reading, err := s.repo.GetLatest(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("service: get latest reading: %w", err)
	}

	return reading, nil
}

func validate(input NewReadingInput) error {
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

	if input.ECmScm != nil && *input.ECmScm < 0 {
		return ErrInvalidEC
	}

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
