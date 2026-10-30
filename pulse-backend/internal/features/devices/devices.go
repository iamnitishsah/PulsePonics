// Package devices is the device registry: which user owns which firmware
// DEVICE_ID, and each device's API key for HTTP ingestion.
package devices

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jmoiron/sqlx"

	"pulse-backend/internal/platform/apperr"
	"pulse-backend/internal/platform/database"
)

// Same characters the firmware can safely put in an MQTT topic segment
// (no "/", "+" or "#"), e.g. "esp32-hydro-01".
var deviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{3,64}$`)

const (
	apiKeyPrefix   = "ppk_"
	maxNameLen     = 100
	shownKeyPrefix = 10 // characters of the key kept for display, e.g. "ppk_Ab12Cd"
)

var errNotFound = apperr.NotFound("device not found")

type Device struct {
	ID           int64     `db:"id" json:"-"`
	DeviceID     string    `db:"device_id" json:"device_id"`
	UserID       int64     `db:"user_id" json:"-"`
	Name         string    `db:"name" json:"name"`
	APIKeyHash   string    `db:"api_key_hash" json:"-"`
	APIKeyPrefix string    `db:"api_key_prefix" json:"api_key_prefix"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// DeviceWithKey is returned once, when a key is created or rotated.
type DeviceWithKey struct {
	Device
	APIKey string `json:"api_key"`
}

type CreateInput struct {
	DeviceID string `json:"device_id"`
	Name     string `json:"name"`
}

type UpdateInput struct {
	Name string `json:"name"`
}

func ValidateDeviceID(id string) error {
	if !deviceIDPattern.MatchString(id) {
		return apperr.Invalid("device_id", "device_id must be 3-64 characters: letters, digits, '-' or '_'")
	}
	return nil
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

const deviceColumns = `id, device_id, user_id, name, api_key_hash, api_key_prefix, created_at`

func (s *Service) List(ctx context.Context, userID int64) ([]Device, error) {
	devices := []Device{}
	err := s.db.SelectContext(ctx, &devices,
		`SELECT `+deviceColumns+` FROM devices WHERE user_id = ? ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("devices: list: %w", err)
	}
	return devices, nil
}

// Get returns the device only if userID owns it. Devices owned by someone
// else get the same 404 as missing ones, so IDs can't be probed.
func (s *Service) Get(ctx context.Context, userID int64, deviceID string) (Device, error) {
	d, err := s.byColumn(ctx, "device_id", deviceID)
	if err != nil {
		return Device{}, err
	}
	if d == nil || d.UserID != userID {
		return Device{}, errNotFound
	}
	return *d, nil
}

// Authorize is the ownership check other features use before serving data.
func (s *Service) Authorize(ctx context.Context, userID int64, deviceID string) error {
	_, err := s.Get(ctx, userID, deviceID)
	return err
}

func (s *Service) Create(ctx context.Context, userID int64, in CreateInput) (DeviceWithKey, error) {
	deviceID := strings.TrimSpace(in.DeviceID)
	if err := ValidateDeviceID(deviceID); err != nil {
		return DeviceWithKey{}, err
	}
	name, err := cleanName(in.Name, deviceID)
	if err != nil {
		return DeviceWithKey{}, err
	}
	key, hash, prefix, err := newAPIKey()
	if err != nil {
		return DeviceWithKey{}, err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO devices (device_id, user_id, name, api_key_hash, api_key_prefix) VALUES (?, ?, ?, ?, ?)`,
		deviceID, userID, name, hash, prefix)
	if err != nil {
		if database.IsDuplicateKey(err) {
			return DeviceWithKey{}, apperr.Conflict("this device ID is already registered")
		}
		return DeviceWithKey{}, fmt.Errorf("devices: insert: %w", err)
	}
	d, err := s.Get(ctx, userID, deviceID)
	if err != nil {
		return DeviceWithKey{}, err
	}
	return DeviceWithKey{Device: d, APIKey: key}, nil
}

func (s *Service) Rename(ctx context.Context, userID int64, deviceID string, in UpdateInput) (Device, error) {
	d, err := s.Get(ctx, userID, deviceID)
	if err != nil {
		return Device{}, err
	}
	name, err := cleanName(in.Name, "")
	if err != nil {
		return Device{}, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE devices SET name = ? WHERE id = ?`, name, d.ID); err != nil {
		return Device{}, fmt.Errorf("devices: rename: %w", err)
	}
	d.Name = name
	return d, nil
}

// Delete removes the device and, through ON DELETE CASCADE, all its readings,
// thresholds and alerts. The device_id can then be registered again.
func (s *Service) Delete(ctx context.Context, userID int64, deviceID string) error {
	d, err := s.Get(ctx, userID, deviceID)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM devices WHERE id = ?`, d.ID); err != nil {
		return fmt.Errorf("devices: delete: %w", err)
	}
	return nil
}

func (s *Service) RotateKey(ctx context.Context, userID int64, deviceID string) (DeviceWithKey, error) {
	d, err := s.Get(ctx, userID, deviceID)
	if err != nil {
		return DeviceWithKey{}, err
	}
	key, hash, prefix, err := newAPIKey()
	if err != nil {
		return DeviceWithKey{}, err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE devices SET api_key_hash = ?, api_key_prefix = ? WHERE id = ?`, hash, prefix, d.ID)
	if err != nil {
		return DeviceWithKey{}, fmt.Errorf("devices: rotate key: %w", err)
	}
	d.APIKeyHash, d.APIKeyPrefix = hash, prefix
	return DeviceWithKey{Device: d, APIKey: key}, nil
}

// OwnerOf is used by ingestion: it reports who owns deviceID, and false for
// unregistered devices (whose telemetry is dropped).
func (s *Service) OwnerOf(ctx context.Context, deviceID string) (int64, bool, error) {
	d, err := s.byColumn(ctx, "device_id", deviceID)
	if err != nil || d == nil {
		return 0, false, err
	}
	return d.UserID, true, nil
}

// ByAPIKey returns the device a raw API key belongs to, or nil.
func (s *Service) ByAPIKey(ctx context.Context, key string) (*Device, error) {
	if !strings.HasPrefix(key, apiKeyPrefix) {
		return nil, nil
	}
	return s.byColumn(ctx, "api_key_hash", hashKey(key))
}

func (s *Service) byColumn(ctx context.Context, column, value string) (*Device, error) {
	var d Device
	// column is always a constant from this file, never user input.
	err := s.db.GetContext(ctx, &d, `SELECT `+deviceColumns+` FROM devices WHERE `+column+` = ?`, value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("devices: get by %s: %w", column, err)
	}
	return &d, nil
}

func cleanName(raw, fallback string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		name = fallback
	}
	if name == "" {
		return "", apperr.Invalid("name", "name is required")
	}
	if utf8.RuneCountInString(name) > maxNameLen {
		return "", apperr.Invalid("name", fmt.Sprintf("name must be at most %d characters", maxNameLen))
	}
	return name, nil
}

func newAPIKey() (key, hash, prefix string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("devices: random key: %w", err)
	}
	key = apiKeyPrefix + base64.RawURLEncoding.EncodeToString(b)
	return key, hashKey(key), key[:shownKeyPrefix], nil
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
