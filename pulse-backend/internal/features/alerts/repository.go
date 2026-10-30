package alerts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"pulse-backend/internal/platform/database"
)

type Alert struct {
	ID             int64      `db:"id" json:"id"`
	DeviceID       string     `db:"device_id" json:"device_id"`
	Metric         string     `db:"metric" json:"metric"`
	Kind           string     `db:"kind" json:"kind"` // "low" or "high"
	ThresholdValue float64    `db:"threshold_value" json:"threshold_value"`
	TriggerValue   float64    `db:"trigger_value" json:"trigger_value"`
	LatestValue    float64    `db:"latest_value" json:"latest_value"`
	OpenedAt       time.Time  `db:"opened_at" json:"opened_at"`
	ResolvedAt     *time.Time `db:"resolved_at" json:"resolved_at"`
	AcknowledgedAt *time.Time `db:"acknowledged_at" json:"acknowledged_at"`
	Status         string     `db:"-" json:"status"` // "open" or "resolved"
}

func (a *Alert) setStatus() {
	a.Status = "open"
	if a.ResolvedAt != nil {
		a.Status = "resolved"
	}
}

type ListFilter struct {
	UserID   int64
	DeviceID string // optional
	Status   string // "open", "resolved" or "" for all
	Limit    int
}

type Repository interface {
	Thresholds(ctx context.Context, deviceID string) ([]Threshold, error)
	SaveThresholds(ctx context.Context, deviceID string, ts []Threshold) error

	OpenAlerts(ctx context.Context, deviceID string) (map[string]*Alert, error)
	Insert(ctx context.Context, a Alert) (Alert, error)
	UpdateOpen(ctx context.Context, id int64, kind string, threshold, last float64) error
	Resolve(ctx context.Context, id int64, at time.Time) error
	Get(ctx context.Context, id int64) (Alert, error)
	List(ctx context.Context, f ListFilter) ([]Alert, error)
	CountOpen(ctx context.Context, deviceID string) (int, error)
	Acknowledge(ctx context.Context, userID, alertID int64, at time.Time) (bool, error)
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewMySQLRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

const alertColumns = `a.id, a.device_id, a.metric, a.kind, a.threshold_value, a.trigger_value,
	a.latest_value, a.opened_at, a.resolved_at, a.acknowledged_at`

func (r *mysqlRepository) Thresholds(ctx context.Context, deviceID string) ([]Threshold, error) {
	out := []Threshold{}
	err := r.db.SelectContext(ctx, &out,
		`SELECT metric, min_value, max_value, enabled FROM thresholds WHERE device_id = ?`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("alerts repo: thresholds: %w", err)
	}
	return out, nil
}

func (r *mysqlRepository) SaveThresholds(ctx context.Context, deviceID string, ts []Threshold) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("alerts repo: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, t := range ts {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO thresholds (device_id, metric, min_value, max_value, enabled)
			VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE min_value = VALUES(min_value), max_value = VALUES(max_value), enabled = VALUES(enabled)`,
			deviceID, t.Metric, t.Min, t.Max, t.Enabled)
		if err != nil {
			return fmt.Errorf("alerts repo: save threshold %s: %w", t.Metric, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("alerts repo: commit: %w", err)
	}
	return nil
}

func (r *mysqlRepository) OpenAlerts(ctx context.Context, deviceID string) (map[string]*Alert, error) {
	var list []Alert
	err := r.db.SelectContext(ctx, &list,
		`SELECT `+alertColumns+` FROM alerts a WHERE a.device_id = ? AND a.resolved_at IS NULL`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("alerts repo: open alerts: %w", err)
	}
	out := make(map[string]*Alert, len(list))
	for i := range list {
		list[i].setStatus()
		out[list[i].Metric] = &list[i]
	}
	return out, nil
}

// Insert opens an alert. If one is already open for the same device and
// metric (a concurrent ingest got there first), the unique key rejects the
// insert and errAlreadyOpen is returned.
func (r *mysqlRepository) Insert(ctx context.Context, a Alert) (Alert, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO alerts (device_id, metric, kind, threshold_value, trigger_value, latest_value, opened_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.DeviceID, a.Metric, a.Kind, a.ThresholdValue, a.TriggerValue, a.LatestValue, a.OpenedAt)
	if err != nil {
		if database.IsDuplicateKey(err) {
			return Alert{}, errAlreadyOpen
		}
		return Alert{}, fmt.Errorf("alerts repo: insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Alert{}, fmt.Errorf("alerts repo: last insert id: %w", err)
	}
	return r.Get(ctx, id)
}

var errAlreadyOpen = errors.New("alerts: an alert is already open for this metric")

func (r *mysqlRepository) UpdateOpen(ctx context.Context, id int64, kind string, threshold, last float64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE alerts SET kind = ?, threshold_value = ?, latest_value = ? WHERE id = ? AND resolved_at IS NULL`,
		kind, threshold, last, id)
	if err != nil {
		return fmt.Errorf("alerts repo: update: %w", err)
	}
	return nil
}

func (r *mysqlRepository) Resolve(ctx context.Context, id int64, at time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE alerts SET resolved_at = ? WHERE id = ? AND resolved_at IS NULL`, at, id)
	if err != nil {
		return fmt.Errorf("alerts repo: resolve: %w", err)
	}
	return nil
}

func (r *mysqlRepository) Get(ctx context.Context, id int64) (Alert, error) {
	var a Alert
	if err := r.db.GetContext(ctx, &a, `SELECT `+alertColumns+` FROM alerts a WHERE a.id = ?`, id); err != nil {
		return Alert{}, fmt.Errorf("alerts repo: get: %w", err)
	}
	a.setStatus()
	return a, nil
}

func (r *mysqlRepository) List(ctx context.Context, f ListFilter) ([]Alert, error) {
	where := []string{"d.user_id = ?"}
	args := []any{f.UserID}
	if f.DeviceID != "" {
		where = append(where, "a.device_id = ?")
		args = append(args, f.DeviceID)
	}
	switch f.Status {
	case "open":
		where = append(where, "a.resolved_at IS NULL")
	case "resolved":
		where = append(where, "a.resolved_at IS NOT NULL")
	}
	args = append(args, f.Limit)

	out := []Alert{}
	err := r.db.SelectContext(ctx, &out, `
		SELECT `+alertColumns+`
		FROM alerts a JOIN devices d ON d.device_id = a.device_id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY (a.resolved_at IS NULL) DESC, a.opened_at DESC
		LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("alerts repo: list: %w", err)
	}
	for i := range out {
		out[i].setStatus()
	}
	return out, nil
}

func (r *mysqlRepository) CountOpen(ctx context.Context, deviceID string) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM alerts WHERE device_id = ? AND resolved_at IS NULL`, deviceID)
	if err != nil {
		return 0, fmt.Errorf("alerts repo: count open: %w", err)
	}
	return n, nil
}

// Acknowledge marks an alert as seen. It returns false when the alert does
// not exist or belongs to another user.
func (r *mysqlRepository) Acknowledge(ctx context.Context, userID, alertID int64, at time.Time) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `
		SELECT 1 FROM alerts a JOIN devices d ON d.device_id = a.device_id
		WHERE a.id = ? AND d.user_id = ?`, alertID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("alerts repo: acknowledge lookup: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE alerts SET acknowledged_at = COALESCE(acknowledged_at, ?) WHERE id = ?`, at, alertID)
	if err != nil {
		return false, fmt.Errorf("alerts repo: acknowledge: %w", err)
	}
	return true, nil
}
