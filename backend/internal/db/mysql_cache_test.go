package db

import (
	"testing"

	"fpl-assistant/internal/fpl"
)

func TestCacheEnvelope_RoundTripTypedValues(t *testing.T) {
	bootstrap := &fpl.BootstrapStatic{
		Events: []fpl.Event{{ID: 1, IsCurrent: true}},
	}
	raw, err := encodeCacheValue(bootstrap)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, ok := decodeCacheValue(raw)
	if !ok {
		t.Fatal("decode failed")
	}
	typed, ok := got.(*fpl.BootstrapStatic)
	if !ok {
		t.Fatalf("expected *BootstrapStatic, got %T", got)
	}
	if len(typed.Events) != 1 || typed.Events[0].ID != 1 {
		t.Fatalf("typed restore lost events: %+v", typed.Events)
	}

	fixtures := []fpl.Fixture{{ID: 9, TeamH: 1, TeamA: 2}}
	raw, err = encodeCacheValue(fixtures)
	if err != nil {
		t.Fatalf("encode fixtures: %v", err)
	}
	got, ok = decodeCacheValue(raw)
	if !ok {
		t.Fatal("decode fixtures failed")
	}
	if _, ok := got.([]fpl.Fixture); !ok {
		t.Fatalf("expected []Fixture, got %T", got)
	}
}

func TestCacheEnvelope_RejectsBareJSON(t *testing.T) {
	if _, ok := decodeCacheValue([]byte(`{"events":[]}`)); ok {
		t.Fatal("bare JSON without envelope must not restore (avoids map[string]interface{} type loss)")
	}
}
