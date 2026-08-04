// internal/domain/reading.go
package domain

import "time"

// Reading represents a single telemetry payload from the ESP32 — one row
// in the `readings` table. Pointer types (*float64) are used for sensor
// values because they can be null (sensor missing/failed), and Go's zero
// value for float64 is 0.0 — which is a valid pH/EC reading, so we can't
// use bare float64 to represent "no data".
type Reading struct {
	ID       int64  `db:"id" json:"id"`
	DeviceID string `db:"device_id" json:"device_id"`

	PH          *float64 `db:"ph" json:"ph,omitempty"`
	ECmScm      *float64 `db:"ec_ms_cm" json:"ec_ms_cm,omitempty"`
	WaterTempC  *float64 `db:"water_temp_c" json:"water_temp_c,omitempty"`
	AirTempC    *float64 `db:"air_temp_c" json:"air_temp_c,omitempty"`
	HumidityPct *float64 `db:"humidity_pct" json:"humidity_pct,omitempty"`
	PressureHPa *float64 `db:"pressure_hpa" json:"pressure_hpa,omitempty"`

	RecordedAt time.Time `db:"recorded_at" json:"recorded_at"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// NewReadingInput is what we accept from the HTTP POST body / MQTT payload.
// It's a separate type from Reading on purpose: the incoming payload never
// includes ID or CreatedAt (server-generated), and keeping this separate
// means the JSON tags for "what a client sends" and "what a client reads
// back" can evolve independently without fighting each other.
type NewReadingInput struct {
	DeviceID    string    `db:"device_id" json:"device_id"`
	PH          *float64  `db:"ph" json:"ph,omitempty"`
	ECmScm      *float64  `db:"ec_ms_cm" json:"ec_ms_cm,omitempty"`
	WaterTempC  *float64  `db:"water_temp_c" json:"water_temp_c,omitempty"`
	AirTempC    *float64  `db:"air_temp_c" json:"air_temp_c,omitempty"`
	HumidityPct *float64  `db:"humidity_pct" json:"humidity_pct,omitempty"`
	PressureHPa *float64  `db:"pressure_hpa" json:"pressure_hpa,omitempty"`
	RecordedAt  time.Time `db:"recorded_at" json:"recorded_at"`
}
