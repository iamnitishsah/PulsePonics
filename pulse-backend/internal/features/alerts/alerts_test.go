package alerts

import (
	"context"
	"testing"
	"time"

	"pulse-backend/internal/features/readings"
	"pulse-backend/internal/platform/apperr"
)

var phBand = Threshold{Metric: "ph", Min: f(5.5), Max: f(6.5), Enabled: true}

func TestDecide(t *testing.T) {
	openLow := &Alert{ID: 1, Kind: "low", ThresholdValue: 5.5}
	openHigh := &Alert{ID: 2, Kind: "high", ThresholdValue: 6.5}

	cases := []struct {
		name  string
		t     Threshold
		value float64
		open  *Alert
		want  action
		kind  string
	}{
		{"in band, nothing open", phBand, 6.0, nil, actionNone, ""},
		{"below min opens low", phBand, 5.2, nil, actionOpen, "low"},
		{"above max opens high", phBand, 6.9, nil, actionOpen, "high"},
		{"still low updates", phBand, 5.3, openLow, actionUpdate, "low"},
		{"just back in band stays open (hysteresis)", phBand, 5.52, openLow, actionUpdate, "low"},
		{"comfortably back resolves low", phBand, 5.6, openLow, actionResolve, ""},
		{"comfortably back resolves high", phBand, 6.4, openHigh, actionResolve, ""},
		{"flip from low to high updates kind", phBand, 7.0, openLow, actionUpdate, "high"},
		{"disabled resolves open alert", Threshold{Metric: "ph", Enabled: false}, 9, openLow, actionResolve, ""},
		{"disabled with nothing open", Threshold{Metric: "ph", Enabled: false}, 9, nil, actionNone, ""},
		{"max-only threshold", Threshold{Metric: "water_temp_c", Max: f(24), Enabled: true}, 25, nil, actionOpen, "high"},
		{"max-only resolves exactly at limit", Threshold{Metric: "water_temp_c", Max: f(24), Enabled: true}, 24, openHigh, actionResolve, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := decide(tc.t, tc.value, tc.open)
			if d.action != tc.want || d.kind != tc.kind {
				t.Errorf("got action=%d kind=%q, want action=%d kind=%q", d.action, d.kind, tc.want, tc.kind)
			}
		})
	}
}

func TestValidateThreshold(t *testing.T) {
	bad := []Threshold{
		{Metric: "co2", Min: f(1), Enabled: true},
		{Metric: "ph", Enabled: true},                       // enabled without bounds
		{Metric: "ph", Min: f(7), Max: f(6), Enabled: true}, // min >= max
		{Metric: "ph", Max: f(15), Enabled: true},           // outside sensor range
	}
	for _, th := range bad {
		if err := validateThreshold(th); !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("%+v: want invalid, got %v", th, err)
		}
	}
	for _, th := range DefaultThresholds() {
		if err := validateThreshold(th); err != nil {
			t.Errorf("default %s invalid: %v", th.Metric, err)
		}
	}
}

func TestMergeUsesStoredOverDefaults(t *testing.T) {
	got := merge([]Threshold{{Metric: "ph", Min: f(6), Max: f(7), Enabled: true}})
	if len(got) != len(readings.Metrics) {
		t.Fatalf("want %d thresholds, got %d", len(readings.Metrics), len(got))
	}
	if *got[0].Min != 6 {
		t.Errorf("stored pH threshold not applied: %+v", got[0])
	}
	if *got[1].Min != 1.2 {
		t.Errorf("EC should keep its default: %+v", got[1])
	}
}

// ---- evaluate with an in-memory repository ----

type memRepo struct {
	Repository
	stored []Threshold
	alerts []Alert
}

func (m *memRepo) Thresholds(context.Context, string) ([]Threshold, error) { return m.stored, nil }

func (m *memRepo) OpenAlerts(context.Context, string) (map[string]*Alert, error) {
	out := map[string]*Alert{}
	for i := range m.alerts {
		if m.alerts[i].ResolvedAt == nil {
			out[m.alerts[i].Metric] = &m.alerts[i]
		}
	}
	return out, nil
}

func (m *memRepo) Insert(_ context.Context, a Alert) (Alert, error) {
	a.ID = int64(len(m.alerts) + 1)
	a.setStatus()
	m.alerts = append(m.alerts, a)
	return a, nil
}

func (m *memRepo) UpdateOpen(_ context.Context, id int64, kind string, threshold, last float64) error {
	a := &m.alerts[id-1]
	a.Kind, a.ThresholdValue, a.LatestValue = kind, threshold, last
	return nil
}

func (m *memRepo) Resolve(_ context.Context, id int64, at time.Time) error {
	m.alerts[id-1].ResolvedAt = &at
	return nil
}

func (m *memRepo) Get(_ context.Context, id int64) (Alert, error) {
	a := m.alerts[id-1]
	a.setStatus()
	return a, nil
}

type events struct{ statuses []string }

func (e *events) Publish(_ int64, _ string, data any) {
	e.statuses = append(e.statuses, data.(Alert).Status)
}

func reading(ph float64, minute int) readings.Reading {
	return readings.Reading{
		DeviceID:   "esp32-hydro-01",
		PH:         &ph,
		RecordedAt: time.Date(2026, 10, 24, 8, minute, 0, 0, time.UTC),
	}
}

func TestEvaluateLifecycle(t *testing.T) {
	repo, ev := &memRepo{}, &events{}
	s := NewService(repo, ev)
	ctx := context.Background()

	s.Evaluate(ctx, 1, reading(6.0, 0)) // fine
	s.Evaluate(ctx, 1, reading(5.0, 1)) // opens
	s.Evaluate(ctx, 1, reading(4.8, 2)) // updates
	s.Evaluate(ctx, 1, reading(6.0, 3)) // resolves

	if len(repo.alerts) != 1 {
		t.Fatalf("want exactly one alert, got %d", len(repo.alerts))
	}
	a := repo.alerts[0]
	if a.Kind != "low" || a.TriggerValue != 5.0 || a.LatestValue != 4.8 || a.ResolvedAt == nil {
		t.Errorf("unexpected alert: %+v", a)
	}
	if got := ev.statuses; len(got) != 2 || got[0] != "open" || got[1] != "resolved" {
		t.Errorf("events = %v, want [open resolved]", got)
	}
}

func TestEvaluateIgnoresNullValues(t *testing.T) {
	repo, ev := &memRepo{}, &events{}
	s := NewService(repo, ev)
	s.Evaluate(context.Background(), 1, reading(5.0, 0))
	s.Evaluate(context.Background(), 1, readings.Reading{DeviceID: "esp32-hydro-01"}) // pH null
	if repo.alerts[0].ResolvedAt != nil {
		t.Error("a null reading must not resolve an open alert")
	}
}
