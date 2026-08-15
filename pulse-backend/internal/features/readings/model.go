package readings

import "time"

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
