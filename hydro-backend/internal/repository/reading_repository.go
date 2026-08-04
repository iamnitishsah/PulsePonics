// internal/repository/reading_repository.go
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"hydro-backend/internal/domain"
)

// ReadingRepository defines what the service layer needs from storage —
// not how storage does it. This interface is the boundary: your service
// code will depend on this interface, not on *sqlx.DB directly. That means
// later you could swap MySQL for something else, or mock this interface
// entirely in unit tests, without touching service-layer code.
type ReadingRepository interface {
	Insert(ctx context.Context, r domain.NewReadingInput) (int64, error)
	GetByDevice(ctx context.Context, deviceID string, from, to time.Time, limit, offset int) ([]domain.Reading, error)
	GetLatest(ctx context.Context, deviceID string) (*domain.Reading, error)
}

// mysqlReadingRepository is the concrete MySQL implementation of
// ReadingRepository. It's unexported (lowercase) on purpose — outside code
// should only ever hold a ReadingRepository interface value, obtained via
// NewMySQLReadingRepository below. This is a common Go idiom: "accept
// interfaces, return structs" but keep the struct itself private.
type mysqlReadingRepository struct {
	db *sqlx.DB
}

func NewMySQLReadingRepository(db *sqlx.DB) ReadingRepository {
	return &mysqlReadingRepository{db: db}
}

// Insert writes one telemetry reading and returns its generated ID.
//
// context.Context is threaded through every DB call — it lets HTTP
// handlers cancel in-flight queries if the client disconnects, and lets you
// set query timeouts later. Always accept ctx as the first param in Go for
// anything doing I/O; it's idiomatic and you'll need it once handlers use
// r.Context().
func (repo *mysqlReadingRepository) Insert(ctx context.Context, r domain.NewReadingInput) (int64, error) {
	query := `
		INSERT INTO readings
			(device_id, ph, ec_ms_cm, water_temp_c, air_temp_c, humidity_pct, pressure_hpa, recorded_at)
		VALUES
			(:device_id, :ph, :ec_ms_cm, :water_temp_c, :air_temp_c, :humidity_pct, :pressure_hpa, :recorded_at)
	`

	// NamedExecContext lets us pass a struct directly and match fields to
	// query placeholders by name (:device_id etc.) via the `db` struct
	// tags we defined on NewReadingInput. This avoids the error-prone,
	// hard-to-read alternative of positional `?` placeholders where you
	// have to count argument order by hand.
	result, err := repo.db.NamedExecContext(ctx, query, r)
	if err != nil {
		return 0, fmt.Errorf("repository: insert reading: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("repository: get last insert id: %w", err)
	}

	return id, nil
}

// GetByDevice fetches readings for one device within a time range, newest
// first, with pagination. This is the query your dashboard's chart view
// will call — e.g. "show me device esp32-tank-01's last 24 hours".
func (repo *mysqlReadingRepository) GetByDevice(
	ctx context.Context,
	deviceID string,
	from, to time.Time,
	limit, offset int,
) ([]domain.Reading, error) {
	query := `
		SELECT id, device_id, ph, ec_ms_cm, water_temp_c, air_temp_c,
		       humidity_pct, pressure_hpa, recorded_at, created_at
		FROM readings
		WHERE device_id = ?
		  AND recorded_at BETWEEN ? AND ?
		ORDER BY recorded_at DESC
		LIMIT ? OFFSET ?
	`

	var readings []domain.Reading
	// SelectContext scans all matching rows directly into a slice of
	// structs, using the `db` tags on domain.Reading. This is the main
	// productivity win sqlx gives you over the stdlib database/sql
	// package, which would require manually calling rows.Scan() in a loop.
	err := repo.db.SelectContext(ctx, &readings, query, deviceID, from, to, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("repository: get readings by device: %w", err)
	}

	return readings, nil
}

// GetLatest fetches the single most recent reading for a device — this is
// what your dashboard's "current status" panel and your alerting engine
// (Sprint 4) will poll.
func (repo *mysqlReadingRepository) GetLatest(ctx context.Context, deviceID string) (*domain.Reading, error) {
	query := `
		SELECT id, device_id, ph, ec_ms_cm, water_temp_c, air_temp_c,
		       humidity_pct, pressure_hpa, recorded_at, created_at
		FROM readings
		WHERE device_id = ?
		ORDER BY recorded_at DESC
		LIMIT 1
	`

	var reading domain.Reading
	// GetContext expects exactly one row. sql.ErrNoRows is returned if
	// there's no data yet (e.g. device never reported) — we deliberately
	// don't wrap that as a generic error, so calling code can check for it
	// specifically with errors.Is(err, sql.ErrNoRows) and respond with a
	// clean "no data yet" instead of a 500.
	err := repo.db.GetContext(ctx, &reading, query, deviceID)
	if err != nil {
		return nil, err // deliberately unwrapped — see comment above
	}

	return &reading, nil
}
