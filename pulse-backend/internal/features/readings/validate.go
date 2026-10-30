package readings

import (
	"fmt"
	"strconv"
	"time"

	"pulse-backend/internal/features/devices"
	"pulse-backend/internal/platform/apperr"
)

// maxClockSkew tolerates a device clock slightly ahead of the server's. A
// reading further in the future would stay "latest" forever, so it is rejected.
const maxClockSkew = 5 * time.Minute

// validate checks a payload against the contract in pulse-hardware/README.md
// and normalises recorded_at (defaults to now, UTC, millisecond precision to
// match DATETIME(3)).
func validate(in *Input, now time.Time) error {
	if in.DeviceID == "" {
		return apperr.Invalid("device_id", "device_id is required")
	}
	if err := devices.ValidateDeviceID(in.DeviceID); err != nil {
		return err
	}

	any := false
	for _, m := range Metrics {
		v := in.value(m.Key)
		if v == nil {
			continue
		}
		any = true
		if *v < m.Min || *v > m.Max {
			return apperr.Invalid(m.Key, fmt.Sprintf("%s must be between %s and %s", m.Key, fmtNum(m.Min), fmtNum(m.Max)))
		}
	}
	if !any {
		return apperr.Invalid("", "at least one sensor value must be provided")
	}

	if in.RecordedAt.IsZero() {
		in.RecordedAt = now
	}
	if in.RecordedAt.After(now.Add(maxClockSkew)) {
		return apperr.Invalid("recorded_at", "recorded_at is in the future; check the device clock")
	}
	in.RecordedAt = in.RecordedAt.UTC().Truncate(time.Millisecond)
	return nil
}

func fmtNum(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
