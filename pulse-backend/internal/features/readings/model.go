package readings

import "time"

// Reading is one stored telemetry sample. Sensor fields are nil when the
// firmware sent null (sensor missing, failed, uncalibrated or out of range).
type Reading struct {
	ID       int64  `db:"id" json:"id"`
	DeviceID string `db:"device_id" json:"device_id"`

	PH          *float64 `db:"ph" json:"ph"`
	ECmScm      *float64 `db:"ec_ms_cm" json:"ec_ms_cm"`
	WaterTempC  *float64 `db:"water_temp_c" json:"water_temp_c"`
	AirTempC    *float64 `db:"air_temp_c" json:"air_temp_c"`
	HumidityPct *float64 `db:"humidity_pct" json:"humidity_pct"`
	PressureHPa *float64 `db:"pressure_hpa" json:"pressure_hpa"`

	RecordedAt time.Time `db:"recorded_at" json:"recorded_at"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// Input is the telemetry payload defined in pulse-hardware/README.md
// ("Telemetry payload"). MQTT and POST /api/readings both use it unchanged.
type Input struct {
	DeviceID    string    `json:"device_id"`
	PH          *float64  `json:"ph"`
	ECmScm      *float64  `json:"ec_ms_cm"`
	WaterTempC  *float64  `json:"water_temp_c"`
	AirTempC    *float64  `json:"air_temp_c"`
	HumidityPct *float64  `json:"humidity_pct"`
	PressureHPa *float64  `json:"pressure_hpa"`
	RecordedAt  time.Time `json:"recorded_at"`
}

// Point is one chart sample. In raw mode it is a single reading; in
// aggregated mode it is the average of a time bucket starting at RecordedAt.
type Point struct {
	RecordedAt  time.Time `db:"recorded_at" json:"recorded_at"`
	PH          *float64  `db:"ph" json:"ph"`
	ECmScm      *float64  `db:"ec_ms_cm" json:"ec_ms_cm"`
	WaterTempC  *float64  `db:"water_temp_c" json:"water_temp_c"`
	AirTempC    *float64  `db:"air_temp_c" json:"air_temp_c"`
	HumidityPct *float64  `db:"humidity_pct" json:"humidity_pct"`
	PressureHPa *float64  `db:"pressure_hpa" json:"pressure_hpa"`
}

type History struct {
	DeviceID string    `json:"device_id"`
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
	// 0 = raw readings; otherwise each point averages this many seconds.
	BucketSeconds int     `json:"bucket_seconds"`
	Points        []Point `json:"points"`
}

// Metric describes one sensor field. Min/Max are the accepted range: they
// match the firmware's include/config.h, which nulls anything outside them,
// so real devices never send a value this backend rejects.
type Metric struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Unit  string  `json:"unit"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

var Metrics = []Metric{
	{Key: "ph", Label: "pH", Unit: "", Min: 0, Max: 14},
	{Key: "ec_ms_cm", Label: "EC", Unit: "mS/cm", Min: 0, Max: 20},
	{Key: "water_temp_c", Label: "Water temperature", Unit: "°C", Min: -10, Max: 60},
	{Key: "air_temp_c", Label: "Air temperature", Unit: "°C", Min: -20, Max: 60},
	{Key: "humidity_pct", Label: "Humidity", Unit: "%", Min: 0, Max: 100},
	{Key: "pressure_hpa", Label: "Pressure", Unit: "hPa", Min: 300, Max: 1100},
}

func MetricByKey(key string) (Metric, bool) {
	for _, m := range Metrics {
		if m.Key == key {
			return m, true
		}
	}
	return Metric{}, false
}

// Value returns the reading's value for a metric key, or nil.
func (r Reading) Value(metric string) *float64 {
	switch metric {
	case "ph":
		return r.PH
	case "ec_ms_cm":
		return r.ECmScm
	case "water_temp_c":
		return r.WaterTempC
	case "air_temp_c":
		return r.AirTempC
	case "humidity_pct":
		return r.HumidityPct
	case "pressure_hpa":
		return r.PressureHPa
	}
	return nil
}

func (in Input) value(metric string) *float64 {
	return Reading{
		PH: in.PH, ECmScm: in.ECmScm, WaterTempC: in.WaterTempC,
		AirTempC: in.AirTempC, HumidityPct: in.HumidityPct, PressureHPa: in.PressureHPa,
	}.Value(metric)
}
