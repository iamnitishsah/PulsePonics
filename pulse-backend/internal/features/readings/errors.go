package readings

import "errors"

var (
	ErrMissingDeviceID  = errors.New("device_id is required")
	ErrNoSensorData     = errors.New("at least one sensor value must be provided")
	ErrInvalidPH        = errors.New("ph must be between 0 and 14")
	ErrInvalidEC        = errors.New("ec_ms_cm cannot be negative")
	ErrInvalidWaterTemp = errors.New("water_temp_c must be between -10 and 60")
	ErrInvalidAirTemp   = errors.New("air_temp_c must be between -20 and 60")
	ErrInvalidHumidity  = errors.New("humidity_pct must be between 0 and 100")
)
