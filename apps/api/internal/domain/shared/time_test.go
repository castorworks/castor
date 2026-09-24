package shared

import (
	"testing"
	"time"
)

func TestCustomTime_MarshalJSON(t *testing.T) {
	t.Run("marshals time to JSON", func(t *testing.T) {
		tt := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)
		ct := CustomTime(tt)
		data, err := ct.MarshalJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := `"2024-06-15 10:30:45"`
		if string(data) != expected {
			t.Errorf("expected %s, got %s", expected, string(data))
		}
	})

	t.Run("marshals zero time", func(t *testing.T) {
		ct := CustomTime(time.Time{})
		data, err := ct.MarshalJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := `"0001-01-01 00:00:00"`
		if string(data) != expected {
			t.Errorf("expected %s, got %s", expected, string(data))
		}
	})
}

func TestCustomTime_UnmarshalJSON(t *testing.T) {
	t.Run("unmarshals valid JSON time", func(t *testing.T) {
		var ct CustomTime
		err := ct.UnmarshalJSON([]byte(`"2024-06-15 10:30:45"`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)
		if time.Time(ct).Unix() != expected.Unix() {
			t.Errorf("expected %v, got %v", expected, time.Time(ct))
		}
	})

	t.Run("empty string results in zero time", func(t *testing.T) {
		var ct CustomTime
		err := ct.UnmarshalJSON([]byte(`""`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !time.Time(ct).IsZero() {
			t.Errorf("expected zero time, got %v", time.Time(ct))
		}
	})
}

func TestCustomTime_String(t *testing.T) {
	tt := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	ct := CustomTime(tt)
	result := ct.String()
	if result != "2024-03-01 12:00:00" {
		t.Errorf("expected '2024-03-01 12:00:00', got %q", result)
	}
}

func TestCustomTime_Value(t *testing.T) {
	t.Run("non-zero time returns byte slice", func(t *testing.T) {
		tt := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)
		ct := CustomTime(tt)
		val, err := ct.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		b, ok := val.([]byte)
		if !ok {
			t.Fatalf("expected []byte, got %T", val)
		}
		if string(b) != "2024-06-15 10:30:45" {
			t.Errorf("expected '2024-06-15 10:30:45', got %s", string(b))
		}
	})

	t.Run("zero time returns nil", func(t *testing.T) {
		ct := CustomTime(time.Time{})
		val, err := ct.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil, got %v", val)
		}
	})
}

func TestCustomTime_Scan(t *testing.T) {
	t.Run("scans time.Time value", func(t *testing.T) {
		var ct CustomTime
		source := time.Date(2024, 6, 15, 10, 30, 45, 0, time.Local)
		err := ct.Scan(source)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify the parsed time has the correct components
		parsed := time.Time(ct)
		if parsed.Year() != 2024 || parsed.Month() != 6 || parsed.Day() != 15 {
			t.Errorf("expected 2024-06-15, got %v", parsed)
		}
	})

	t.Run("preserves instant in any location", func(t *testing.T) {
		var ct CustomTime
		source := time.Date(2024, 6, 15, 23, 30, 0, 0, time.FixedZone("UTC-5", -5*3600))
		if err := ct.Scan(source); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !time.Time(ct).Equal(source) {
			t.Errorf("expected %v, got %v", source, time.Time(ct))
		}
	})

	t.Run("round trips Value output", func(t *testing.T) {
		original := CustomTime(time.Date(2024, 6, 15, 10, 30, 45, 0, time.Local))
		val, err := original.Value()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var ct CustomTime
		if err := ct.Scan(val); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !time.Time(ct).Equal(time.Time(original)) {
			t.Errorf("expected %v, got %v", time.Time(original), time.Time(ct))
		}
	})

	t.Run("nil scans to zero time", func(t *testing.T) {
		ct := CustomTime(time.Now())
		if err := ct.Scan(nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !time.Time(ct).IsZero() {
			t.Errorf("expected zero time, got %v", time.Time(ct))
		}
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		var ct CustomTime
		if err := ct.Scan("not a time"); err == nil {
			t.Error("expected error for malformed string")
		}
		if err := ct.Scan(42); err == nil {
			t.Error("expected error for unsupported type")
		}
	})
}
