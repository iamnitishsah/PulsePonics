package readings

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"pulse-backend/internal/platform/apperr"
)

// ErrUnknownDevice means nobody has registered this device_id; its
// telemetry is dropped.
var ErrUnknownDevice = errors.New("readings: device is not registered")

const (
	defaultHistoryRange = 24 * time.Hour
	maxHistoryRange     = 31 * 24 * time.Hour
	defaultMaxPoints    = 500
	minMaxPoints        = 10
	maxMaxPoints        = 2000
	defaultRecentLimit  = 20
	maxRecentLimit      = 200
)

// DeviceDirectory answers "who owns this device_id?" for ingestion.
type DeviceDirectory interface {
	OwnerOf(ctx context.Context, deviceID string) (userID int64, ok bool, err error)
}

// AlertEvaluator checks a freshly stored reading against the device's thresholds.
type AlertEvaluator interface {
	Evaluate(ctx context.Context, userID int64, r Reading)
}

// Notifier pushes realtime events to a user's open dashboards.
type Notifier interface {
	Publish(userID int64, eventType string, data any)
}

type Service struct {
	repo     Repository
	devices  DeviceDirectory
	alerts   AlertEvaluator
	notifier Notifier
	now      func() time.Time
}

func NewService(repo Repository, devices DeviceDirectory, alerts AlertEvaluator, notifier Notifier) *Service {
	return &Service{repo: repo, devices: devices, alerts: alerts, notifier: notifier, now: time.Now}
}

// Ingest validates and stores one reading, pushes it to the owner's
// dashboards and evaluates alert thresholds. MQTT and HTTP both end here.
func (s *Service) Ingest(ctx context.Context, in Input) (Reading, error) {
	if err := validate(&in, s.now()); err != nil {
		return Reading{}, err
	}
	ownerID, ok, err := s.devices.OwnerOf(ctx, in.DeviceID)
	if err != nil {
		return Reading{}, err
	}
	if !ok {
		return Reading{}, ErrUnknownDevice
	}

	reading, err := s.repo.Insert(ctx, in)
	if err != nil {
		return Reading{}, err
	}

	s.notifier.Publish(ownerID, "reading", reading)
	s.alerts.Evaluate(ctx, ownerID, reading)
	return reading, nil
}

func (s *Service) Latest(ctx context.Context, deviceID string) (*Reading, error) {
	return s.repo.Latest(ctx, deviceID)
}

func (s *Service) Recent(ctx context.Context, deviceID string, limit int) ([]Reading, error) {
	if limit <= 0 {
		limit = defaultRecentLimit
	}
	if limit > maxRecentLimit {
		return nil, apperr.Invalid("limit", fmt.Sprintf("limit must be at most %d", maxRecentLimit))
	}
	return s.repo.Recent(ctx, deviceID, limit)
}

type HistoryQuery struct {
	From, To  time.Time // zero = defaults (last 24 h)
	MaxPoints int       // 0 = default (500)
}

// History returns at most MaxPoints chart points. When the range holds more
// readings than that, they are averaged into equal time buckets.
func (s *Service) History(ctx context.Context, deviceID string, q HistoryQuery) (History, error) {
	to := q.To
	if to.IsZero() {
		to = s.now()
	}
	from := q.From
	if from.IsZero() {
		from = to.Add(-defaultHistoryRange)
	}
	from, to = from.UTC(), to.UTC()
	if !from.Before(to) {
		return History{}, apperr.Invalid("from", "from must be before to")
	}
	if to.Sub(from) > maxHistoryRange {
		return History{}, apperr.Invalid("from", "range must be at most 31 days")
	}

	maxPoints := q.MaxPoints
	if maxPoints == 0 {
		maxPoints = defaultMaxPoints
	}
	if maxPoints < minMaxPoints || maxPoints > maxMaxPoints {
		return History{}, apperr.Invalid("max_points", fmt.Sprintf("max_points must be between %d and %d", minMaxPoints, maxMaxPoints))
	}

	h := History{DeviceID: deviceID, From: from, To: to}

	count, err := s.repo.Count(ctx, deviceID, from, to)
	if err != nil {
		return History{}, err
	}
	if count <= maxPoints {
		h.Points, err = s.repo.Range(ctx, deviceID, from, to)
		return h, err
	}

	h.BucketSeconds = bucketFor(to.Sub(from), maxPoints)
	h.Points, err = s.repo.Aggregate(ctx, deviceID, from, to, h.BucketSeconds)
	return h, err
}

// niceBuckets are bucket sizes (seconds) that give round chart timestamps.
var niceBuckets = []int{10, 15, 30, 60, 120, 300, 600, 900, 1800, 3600, 7200, 10800, 21600, 43200, 86400}

// bucketFor picks the smallest nice bucket that keeps the result within
// maxPoints. Buckets are aligned to the epoch, so a range can touch one more
// bucket than range/size; dividing by maxPoints-1 leaves room for it.
func bucketFor(r time.Duration, maxPoints int) int {
	need := int(math.Ceil(r.Seconds() / float64(maxPoints-1)))
	for _, b := range niceBuckets {
		if b >= need {
			return b
		}
	}
	return need
}
