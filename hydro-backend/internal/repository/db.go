// internal/repository/db.go
package repository

import (
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // driver registers itself via init() — blank import is intentional
	"github.com/jmoiron/sqlx"
)

// NewDB opens a connection pool to MySQL and verifies it's reachable.
// sqlx.Connect (vs sqlx.Open) does an actual Ping under the hood, so if
// your credentials or host are wrong, you find out immediately at startup
// rather than on the first query.
func NewDB(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to connect to db: %w", err)
	}

	// Connection pool tuning. Defaults are technically "unlimited" which is
	// dangerous — a bug or traffic spike could open unbounded connections
	// and take down your MySQL server. These numbers are conservative and
	// fine for a single-device hydroponics project; you'd raise them for
	// heavier production traffic.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
