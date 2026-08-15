package readings

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type ReadingRepository interface {
	Insert(ctx context.Context, r NewReadingInput) (int64, error)
	GetByDevice(ctx context.Context, deviceID string, from, to time.Time, limit, offset int) ([]Reading, error)
	GetLatest(ctx context.Context, deviceID string) (*Reading, error)
}

type mysqlReadingRepository struct {
	db *sqlx.DB
}

func NewMySQLRepository(db *sqlx.DB) ReadingRepository {
	return &mysqlReadingRepository{db: db}
}

func (repo *mysqlReadingRepository) Insert(ctx context.Context, r NewReadingInput) (int64, error) {
	query := `
		INSERT INTO readings
			(device_id, ph, ec_ms_cm, water_temp_c, air_temp_c, humidity_pct, pressure_hpa, recorded_at)
		VALUES
			(:device_id, :ph, :ec_ms_cm, :water_temp_c, :air_temp_c, :humidity_pct, :pressure_hpa, :recorded_at)
	`

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

func (repo *mysqlReadingRepository) GetByDevice(ctx context.Context, deviceID string, from time.Time, to time.Time, limit int, offset int) ([]Reading, error) {
	query := `
		SELECT id, device_id, ph, ec_ms_cm, water_temp_c, air_temp_c,
		       humidity_pct, pressure_hpa, recorded_at, created_at
		FROM readings
		WHERE device_id = ?
		  AND recorded_at BETWEEN ? AND ?
		ORDER BY recorded_at DESC
		LIMIT ? OFFSET ?
	`

	var readings []Reading
	err := repo.db.SelectContext(ctx, &readings, query, deviceID, from, to, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("repository: get readings by device: %w", err)
	}

	return readings, nil
}

func (repo *mysqlReadingRepository) GetLatest(ctx context.Context, deviceID string) (*Reading, error) {
	query := `
		SELECT id, device_id, ph, ec_ms_cm, water_temp_c, air_temp_c,
		       humidity_pct, pressure_hpa, recorded_at, created_at
		FROM readings
		WHERE device_id = ?
		ORDER BY recorded_at DESC
		LIMIT 1
	`

	var reading Reading
	err := repo.db.GetContext(ctx, &reading, query, deviceID)
	if err != nil {
		return nil, err
	}

	return &reading, nil
}
