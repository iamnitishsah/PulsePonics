package alerts

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"pulse-backend/internal/features/readings"
	"pulse-backend/internal/platform/apperr"
)

const (
	defaultListLimit = 50
	maxListLimit     = 200
)

type Notifier interface {
	Publish(userID int64, eventType string, data any)
}

// LatestReader lets threshold changes re-check the device's latest reading
// immediately, instead of waiting for the next one.
type LatestReader interface {
	Latest(ctx context.Context, deviceID string) (*readings.Reading, error)
}

type Service struct {
	repo     Repository
	notifier Notifier
	latest   LatestReader
	now      func() time.Time
}

func NewService(repo Repository, notifier Notifier) *Service {
	return &Service{repo: repo, notifier: notifier, now: time.Now}
}

// SetLatestReader breaks the construction cycle: the readings service needs
// this service as its evaluator, and this service reads latest readings.
func (s *Service) SetLatestReader(l LatestReader) {
	s.latest = l
}

// Thresholds returns the effective thresholds for every metric.
func (s *Service) Thresholds(ctx context.Context, deviceID string) ([]Threshold, error) {
	stored, err := s.repo.Thresholds(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	return merge(stored), nil
}

// UpdateThresholds validates and saves the given thresholds (any subset of
// metrics), then re-evaluates the latest reading so alerts reflect the new
// limits right away.
func (s *Service) UpdateThresholds(ctx context.Context, userID int64, deviceID string, in []Threshold) ([]Threshold, error) {
	if len(in) == 0 {
		return nil, apperr.Invalid("thresholds", "at least one threshold is required")
	}
	seen := map[string]bool{}
	for _, t := range in {
		if err := validateThreshold(t); err != nil {
			return nil, err
		}
		if seen[t.Metric] {
			return nil, apperr.Invalid(t.Metric, fmt.Sprintf("duplicate metric %q", t.Metric))
		}
		seen[t.Metric] = true
	}
	if err := s.repo.SaveThresholds(ctx, deviceID, in); err != nil {
		return nil, err
	}

	// Resolve alerts for metrics that were just disabled; their readings may
	// be null, so the latest-reading pass below would never touch them.
	open, err := s.repo.OpenAlerts(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	for _, t := range in {
		if a, ok := open[t.Metric]; ok && !t.Enabled {
			if err := s.resolve(ctx, userID, a, s.now()); err != nil {
				return nil, err
			}
		}
	}

	if s.latest != nil {
		if r, err := s.latest.Latest(ctx, deviceID); err != nil {
			return nil, err
		} else if r != nil {
			if err := s.evaluate(ctx, userID, *r); err != nil {
				return nil, err
			}
		}
	}
	return s.Thresholds(ctx, deviceID)
}

// Evaluate implements readings.AlertEvaluator. Alert failures are logged,
// never returned: a broken alert must not lose the reading itself.
func (s *Service) Evaluate(ctx context.Context, userID int64, r readings.Reading) {
	if err := s.evaluate(ctx, userID, r); err != nil {
		log.Printf("alerts: evaluate reading %d (%s): %v", r.ID, r.DeviceID, err)
	}
}

func (s *Service) evaluate(ctx context.Context, userID int64, r readings.Reading) error {
	thresholds, err := s.Thresholds(ctx, r.DeviceID)
	if err != nil {
		return err
	}
	open, err := s.repo.OpenAlerts(ctx, r.DeviceID)
	if err != nil {
		return err
	}

	for _, t := range thresholds {
		v := r.Value(t.Metric)
		if v == nil {
			continue // sensor sent null: keep the alert state as it is
		}
		current := open[t.Metric]
		d := decide(t, *v, current)

		switch d.action {
		case actionOpen:
			a, err := s.repo.Insert(ctx, Alert{
				DeviceID: r.DeviceID, Metric: t.Metric, Kind: d.kind,
				ThresholdValue: d.bound, TriggerValue: *v, LatestValue: *v, OpenedAt: r.RecordedAt,
			})
			if errors.Is(err, errAlreadyOpen) {
				continue
			}
			if err != nil {
				return err
			}
			s.notifier.Publish(userID, "alert", a)
		case actionUpdate:
			if err := s.repo.UpdateOpen(ctx, current.ID, d.kind, d.bound, *v); err != nil {
				return err
			}
		case actionResolve:
			if err := s.resolve(ctx, userID, current, r.RecordedAt); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) resolve(ctx context.Context, userID int64, a *Alert, at time.Time) error {
	if err := s.repo.Resolve(ctx, a.ID, at); err != nil {
		return err
	}
	resolved, err := s.repo.Get(ctx, a.ID)
	if err != nil {
		return err
	}
	s.notifier.Publish(userID, "alert", resolved)
	return nil
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]Alert, error) {
	switch f.Status {
	case "", "all":
		f.Status = ""
	case "open", "resolved":
	default:
		return nil, apperr.Invalid("status", "status must be open, resolved or all")
	}
	if f.Limit == 0 {
		f.Limit = defaultListLimit
	}
	if f.Limit < 0 || f.Limit > maxListLimit {
		return nil, apperr.Invalid("limit", fmt.Sprintf("limit must be between 1 and %d", maxListLimit))
	}
	return s.repo.List(ctx, f)
}

func (s *Service) CountOpen(ctx context.Context, deviceID string) (int, error) {
	return s.repo.CountOpen(ctx, deviceID)
}

func (s *Service) Acknowledge(ctx context.Context, userID, alertID int64) (Alert, error) {
	ok, err := s.repo.Acknowledge(ctx, userID, alertID, s.now())
	if err != nil {
		return Alert{}, err
	}
	if !ok {
		return Alert{}, apperr.NotFound("alert not found")
	}
	return s.repo.Get(ctx, alertID)
}
