package readings

import (
	"context"
	"errors"
	"testing"
	"time"

	"pulse-backend/internal/platform/apperr"
)

func ptr(v float64) *float64 { return &v }

var testNow = time.Date(2026, 10, 24, 8, 20, 0, 0, time.UTC)

// The exact payload documented in pulse-hardware/README.md.
func firmwarePayload() Input {
	return Input{
		DeviceID:   "esp32-hydro-01",
		WaterTempC: ptr(28.44),
		RecordedAt: time.Date(2026, 10, 24, 8, 20, 0, 0, time.UTC),
	}
}

func TestValidateAcceptsFirmwarePayload(t *testing.T) {
	in := firmwarePayload()
	if err := validate(&in, testNow); err != nil {
		t.Fatalf("firmware payload rejected: %v", err)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]struct {
		mutate func(*Input)
		field  string
	}{
		"missing device":   {func(in *Input) { in.DeviceID = "" }, "device_id"},
		"topic chars":      {func(in *Input) { in.DeviceID = "esp32/01" }, "device_id"},
		"no sensor values": {func(in *Input) { in.WaterTempC = nil }, ""},
		"ph too high":      {func(in *Input) { in.PH = ptr(14.5) }, "ph"},
		"ec negative":      {func(in *Input) { in.ECmScm = ptr(-0.1) }, "ec_ms_cm"},
		"water too cold":   {func(in *Input) { in.WaterTempC = ptr(-11) }, "water_temp_c"},
		"air too hot":      {func(in *Input) { in.AirTempC = ptr(61) }, "air_temp_c"},
		"humidity > 100":   {func(in *Input) { in.HumidityPct = ptr(101) }, "humidity_pct"},
		"pressure too low": {func(in *Input) { in.PressureHPa = ptr(200) }, "pressure_hpa"},
		"future timestamp": {func(in *Input) { in.RecordedAt = testNow.Add(time.Hour) }, "recorded_at"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			in := firmwarePayload()
			tc.mutate(&in)
			err := validate(&in, testNow)
			e, ok := apperr.As(err)
			if !ok || e.Kind != apperr.KindInvalid {
				t.Fatalf("want invalid error, got %v", err)
			}
			if e.Field != tc.field {
				t.Errorf("field = %q, want %q", e.Field, tc.field)
			}
		})
	}
}

func TestValidateNormalisesTime(t *testing.T) {
	in := firmwarePayload()
	in.RecordedAt = time.Time{}
	if err := validate(&in, testNow); err != nil {
		t.Fatal(err)
	}
	if !in.RecordedAt.Equal(testNow) {
		t.Errorf("missing recorded_at should default to now, got %v", in.RecordedAt)
	}

	ist := time.FixedZone("IST", 5*3600+1800)
	in = firmwarePayload()
	in.RecordedAt = time.Date(2026, 10, 24, 13, 50, 0, 123456789, ist)
	if err := validate(&in, testNow); err != nil {
		t.Fatal(err)
	}
	if in.RecordedAt.Location() != time.UTC || in.RecordedAt.Nanosecond() != 123000000 {
		t.Errorf("want UTC with ms precision, got %v", in.RecordedAt)
	}
}

// ---- service with fakes ----

type fakeRepo struct {
	Repository
	inserted []Input
	dupe     bool
}

func (f *fakeRepo) Insert(_ context.Context, in Input) (Reading, error) {
	if f.dupe {
		return Reading{}, ErrDuplicate
	}
	f.inserted = append(f.inserted, in)
	return Reading{ID: int64(len(f.inserted)), DeviceID: in.DeviceID, WaterTempC: in.WaterTempC, RecordedAt: in.RecordedAt}, nil
}

type fakeDirectory map[string]int64

func (d fakeDirectory) OwnerOf(_ context.Context, id string) (int64, bool, error) {
	owner, ok := d[id]
	return owner, ok, nil
}

type recorder struct {
	published []int64
	evaluated []int64
}

func (r *recorder) Publish(userID int64, _ string, _ any) { r.published = append(r.published, userID) }
func (r *recorder) Evaluate(_ context.Context, userID int64, _ Reading) {
	r.evaluated = append(r.evaluated, userID)
}

func newTestService(repo *fakeRepo, rec *recorder) *Service {
	s := NewService(repo, fakeDirectory{"esp32-hydro-01": 7}, rec, rec)
	s.now = func() time.Time { return testNow }
	return s
}

func TestIngestStoresAndNotifiesOwner(t *testing.T) {
	repo, rec := &fakeRepo{}, &recorder{}
	r, err := newTestService(repo, rec).Ingest(context.Background(), firmwarePayload())
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != 1 || len(repo.inserted) != 1 {
		t.Fatalf("reading not stored: %+v", r)
	}
	if len(rec.published) != 1 || rec.published[0] != 7 {
		t.Errorf("owner not notified: %v", rec.published)
	}
	if len(rec.evaluated) != 1 {
		t.Errorf("alerts not evaluated")
	}
}

func TestIngestDropsUnregisteredDevice(t *testing.T) {
	repo, rec := &fakeRepo{}, &recorder{}
	in := firmwarePayload()
	in.DeviceID = "esp32-unknown"
	_, err := newTestService(repo, rec).Ingest(context.Background(), in)
	if !errors.Is(err, ErrUnknownDevice) {
		t.Fatalf("want ErrUnknownDevice, got %v", err)
	}
	if len(repo.inserted) != 0 || len(rec.published) != 0 {
		t.Error("unregistered reading must not be stored or published")
	}
}

func TestIngestDuplicateIsNotPublished(t *testing.T) {
	repo, rec := &fakeRepo{dupe: true}, &recorder{}
	_, err := newTestService(repo, rec).Ingest(context.Background(), firmwarePayload())
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("want ErrDuplicate, got %v", err)
	}
	if len(rec.published) != 0 || len(rec.evaluated) != 0 {
		t.Error("duplicate must not be published or evaluated")
	}
}

type historyRepo struct {
	Repository
	count  int
	bucket int
}

func (h *historyRepo) Count(context.Context, string, time.Time, time.Time) (int, error) {
	return h.count, nil
}
func (h *historyRepo) Range(context.Context, string, time.Time, time.Time) ([]Point, error) {
	return []Point{}, nil
}
func (h *historyRepo) Aggregate(_ context.Context, _ string, _, _ time.Time, bucket int) ([]Point, error) {
	h.bucket = bucket
	return []Point{}, nil
}

func TestHistoryAggregatesLargeRanges(t *testing.T) {
	repo := &historyRepo{count: 8640} // one day at 10 s
	s := NewService(repo, nil, nil, nil)
	s.now = func() time.Time { return testNow }

	h, err := s.History(context.Background(), "esp32-hydro-01", HistoryQuery{MaxPoints: 480})
	if err != nil {
		t.Fatal(err)
	}
	if h.BucketSeconds != 300 || repo.bucket != 300 { // 86400 s / 479 = 181 s -> next nice size
		t.Errorf("bucket = %d, want 300", h.BucketSeconds)
	}

	repo.count = 100
	h, err = s.History(context.Background(), "esp32-hydro-01", HistoryQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if h.BucketSeconds != 0 {
		t.Errorf("small ranges must return raw readings, bucket = %d", h.BucketSeconds)
	}
}

func TestBucketForNeverExceedsMaxPoints(t *testing.T) {
	for _, tc := range []struct {
		r   time.Duration
		max int
	}{{time.Hour, 12}, {24 * time.Hour, 500}, {7 * 24 * time.Hour, 500}, {31 * 24 * time.Hour, 10}, {90 * time.Second, 10}} {
		b := bucketFor(tc.r, tc.max)
		// worst case: the range starts just before a bucket boundary
		if buckets := int(tc.r.Seconds())/b + 1; buckets > tc.max {
			t.Errorf("range %s max %d: bucket %ds gives up to %d points", tc.r, tc.max, b, buckets)
		}
	}
}

func TestHistoryRejectsBadRanges(t *testing.T) {
	s := NewService(&historyRepo{}, nil, nil, nil)
	s.now = func() time.Time { return testNow }
	bad := []HistoryQuery{
		{From: testNow, To: testNow.Add(-time.Hour)},
		{From: testNow.Add(-40 * 24 * time.Hour), To: testNow},
		{MaxPoints: 5},
	}
	for _, q := range bad {
		if _, err := s.History(context.Background(), "d", q); !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("%+v: want invalid, got %v", q, err)
		}
	}
}

func TestDeviceFromTopic(t *testing.T) {
	if got := deviceFromTopic("hydroponics/esp32-hydro-01/telemetry"); got != "esp32-hydro-01" {
		t.Errorf("got %q", got)
	}
	if got := deviceFromTopic("hydroponics/telemetry"); got != "" {
		t.Errorf("malformed topic should give empty id, got %q", got)
	}
}
