// Package alerts stores per-device thresholds and opens/resolves in-app
// alerts when readings cross them.
package alerts

import (
	"fmt"
	"strconv"

	"pulse-backend/internal/features/readings"
	"pulse-backend/internal/platform/apperr"
)

// Threshold is the allowed band for one metric of one device. A nil bound
// means "no limit on that side".
type Threshold struct {
	Metric  string   `db:"metric" json:"metric"`
	Min     *float64 `db:"min_value" json:"min"`
	Max     *float64 `db:"max_value" json:"max"`
	Enabled bool     `db:"enabled" json:"enabled"`
}

func f(v float64) *float64 { return &v }

// DefaultThresholds are typical leafy-greens hydroponic ranges. They apply
// until the user saves their own values for a device.
func DefaultThresholds() []Threshold {
	return []Threshold{
		{Metric: "ph", Min: f(5.5), Max: f(6.5), Enabled: true},
		{Metric: "ec_ms_cm", Min: f(1.2), Max: f(2.4), Enabled: true},
		{Metric: "water_temp_c", Min: f(18), Max: f(24), Enabled: true},
		{Metric: "air_temp_c", Min: f(18), Max: f(28), Enabled: true},
		{Metric: "humidity_pct", Min: f(45), Max: f(75), Enabled: true},
		{Metric: "pressure_hpa", Min: nil, Max: nil, Enabled: false},
	}
}

// merge overlays stored rows on the defaults, in the order of readings.Metrics.
func merge(stored []Threshold) []Threshold {
	byMetric := make(map[string]Threshold, len(stored))
	for _, t := range stored {
		byMetric[t.Metric] = t
	}
	out := DefaultThresholds()
	for i, def := range out {
		if t, ok := byMetric[def.Metric]; ok {
			out[i] = t
		}
	}
	return out
}

func validateThreshold(t Threshold) error {
	m, ok := readings.MetricByKey(t.Metric)
	if !ok {
		return apperr.Invalid("metric", fmt.Sprintf("unknown metric %q", t.Metric))
	}
	if t.Enabled && t.Min == nil && t.Max == nil {
		return apperr.Invalid(t.Metric, fmt.Sprintf("%s: set a minimum, a maximum, or both to enable it", m.Label))
	}
	for _, bound := range []*float64{t.Min, t.Max} {
		if bound != nil && (*bound < m.Min || *bound > m.Max) {
			return apperr.Invalid(t.Metric, fmt.Sprintf("%s limits must be between %s and %s",
				m.Label, strconv.FormatFloat(m.Min, 'f', -1, 64), strconv.FormatFloat(m.Max, 'f', -1, 64)))
		}
	}
	if t.Min != nil && t.Max != nil && *t.Min >= *t.Max {
		return apperr.Invalid(t.Metric, fmt.Sprintf("%s: minimum must be lower than maximum", m.Label))
	}
	return nil
}

// hysteresisFraction: an open alert only resolves once the value is back
// inside the band by this fraction of the band's width. Without it, a value
// sitting right on a limit (sensor noise) would open and resolve an alert on
// every reading.
const hysteresisFraction = 0.05

type action int

const (
	actionNone    action = iota
	actionOpen           // no open alert, value breaches
	actionUpdate         // open alert, value still out of band (or inside the hysteresis margin)
	actionResolve        // open alert, value comfortably back in band
)

type decision struct {
	action action
	kind   string  // "low" or "high", for open/update
	bound  float64 // the limit that was crossed
}

// decide is the pure core of alerting: given a threshold, a value and the
// currently open alert (nil if none), what should happen?
func decide(t Threshold, value float64, open *Alert) decision {
	if !t.Enabled {
		if open != nil {
			return decision{action: actionResolve}
		}
		return decision{action: actionNone}
	}

	var breached decision
	switch {
	case t.Min != nil && value < *t.Min:
		breached = decision{kind: "low", bound: *t.Min}
	case t.Max != nil && value > *t.Max:
		breached = decision{kind: "high", bound: *t.Max}
	}

	if breached.kind != "" {
		if open == nil {
			breached.action = actionOpen
		} else {
			breached.action = actionUpdate
		}
		return breached
	}

	if open == nil {
		return decision{action: actionNone}
	}

	margin := 0.0
	if t.Min != nil && t.Max != nil {
		margin = (*t.Max - *t.Min) * hysteresisFraction
	}
	cleared := true
	if open.Kind == "low" && t.Min != nil && value < *t.Min+margin {
		cleared = false
	}
	if open.Kind == "high" && t.Max != nil && value > *t.Max-margin {
		cleared = false
	}
	if cleared {
		return decision{action: actionResolve}
	}
	return decision{action: actionUpdate, kind: open.Kind, bound: open.ThresholdValue}
}
