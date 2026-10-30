package readings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"pulse-backend/internal/platform/database"
)

// ErrDuplicate means a reading with the same device_id and recorded_at is
// already stored (a firmware PUBACK retry).
var ErrDuplicate = errors.New("readings: duplicate reading")

type Repository interface {
	Insert(ctx context.Context, in Input) (Reading, error)
	Latest(ctx context.Context, deviceID string) (*Reading, error)
	Recent(ctx context.Context, deviceID string, limit int) ([]Reading, error)
	Count(ctx context.Context, deviceID string, from, to time.Time) (int, error)
	Range(ctx context.Context, deviceID string, from, to time.Time) ([]Point, error)
	Aggregate(ctx context.Context, deviceID string, from, to time.Time, bucketSeconds int) ([]Point, error)
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewMySQLRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

const readingColumns = `id, device_id, ph, ec_ms_cm, water_temp_c, air_temp_c,
	humidity_pct, pressure_hpa, recorded_at, created_at`

func (r *mysqlRepository) Insert(ctx context.Context, in Input) (Reading, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO readings
			(device_id, ph, ec_ms_cm, water_temp_c, air_temp_c, humidity_pct, pressure_hpa, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.DeviceID, in.PH, in.ECmScm, in.WaterTempC, in.AirTempC, in.HumidityPct, in.PressureHPa, in.RecordedAt)
	if err != nil {
		if database.IsDuplicateKey(err) {
			return Reading{}, ErrDuplicate
		}
		return Reading{}, fmt.Errorf("readings repo: insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Reading{}, fmt.Errorf("readings repo: last insert id: %w", err)
	}
	// Read the row back so the response and the WebSocket event carry the
	// values exactly as stored (DECIMAL rounding, created_at).
	var stored Reading
	if err := r.db.GetContext(ctx, &stored, `SELECT `+readingColumns+` FROM readings WHERE id = ?`, id); err != nil {
		return Reading{}, fmt.Errorf("readings repo: read back: %w", err)
	}
	return stored, nil
}

func (r *mysqlRepository) Latest(ctx context.Context, deviceID string) (*Reading, error) {
	var reading Reading
	err := r.db.GetContext(ctx, &reading, `
		SELECT `+readingColumns+` FROM readings
		WHERE device_id = ? ORDER BY recorded_at DESC LIMIT 1`, deviceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("readings repo: latest: %w", err)
	}
	return &reading, nil
}

func (r *mysqlRepository) Recent(ctx context.Context, deviceID string, limit int) ([]Reading, error) {
	out := []Reading{}
	err := r.db.SelectContext(ctx, &out, `
		SELECT `+readingColumns+` FROM readings
		WHERE device_id = ? ORDER BY recorded_at DESC LIMIT ?`, deviceID, limit)
	if err != nil {
		return nil, fmt.Errorf("readings repo: recent: %w", err)
	}
	return out, nil
}

func (r *mysqlRepository) Count(ctx context.Context, deviceID string, from, to time.Time) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n, `
		SELECT COUNT(*) FROM readings
		WHERE device_id = ? AND recorded_at >= ? AND recorded_at <= ?`, deviceID, from, to)
	if err != nil {
		return 0, fmt.Errorf("readings repo: count: %w", err)
	}
	return n, nil
}

func (r *mysqlRepository) Range(ctx context.Context, deviceID string, from, to time.Time) ([]Point, error) {
	out := []Point{}
	err := r.db.SelectContext(ctx, &out, `
		SELECT recorded_at, ph, ec_ms_cm, water_temp_c, air_temp_c, humidity_pct, pressure_hpa
		FROM readings
		WHERE device_id = ? AND recorded_at >= ? AND recorded_at <= ?
		ORDER BY recorded_at`, deviceID, from, to)
	if err != nil {
		return nil, fmt.Errorf("readings repo: range: %w", err)
	}
	return out, nil
}

// Aggregate averages readings into fixed buckets aligned to the Unix epoch.
// The session time zone is UTC, so UNIX_TIMESTAMP/FROM_UNIXTIME round-trip exactly.
func (r *mysqlRepository) Aggregate(ctx context.Context, deviceID string, from, to time.Time, bucketSeconds int) ([]Point, error) {
	out := []Point{}
	err := r.db.SelectContext(ctx, &out, `
		SELECT
			FROM_UNIXTIME(FLOOR(UNIX_TIMESTAMP(recorded_at) / ?) * ?) AS recorded_at,
			AVG(ph) AS ph, AVG(ec_ms_cm) AS ec_ms_cm, AVG(water_temp_c) AS water_temp_c,
			AVG(air_temp_c) AS air_temp_c, AVG(humidity_pct) AS humidity_pct, AVG(pressure_hpa) AS pressure_hpa
		FROM readings
		WHERE device_id = ? AND recorded_at >= ? AND recorded_at <= ?
		GROUP BY 1
		ORDER BY 1`, bucketSeconds, bucketSeconds, deviceID, from, to)
	if err != nil {
		return nil, fmt.Errorf("readings repo: aggregate: %w", err)
	}
	return out, nil
}
