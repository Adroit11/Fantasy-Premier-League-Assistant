package actions

import (
	"testing"

	"fpl-assistant/internal/fpl"
)

func TestResolveGameweek(t *testing.T) {
	events := []fpl.Event{
		{ID: 3, IsNext: true},
		{ID: 2, IsCurrent: true},
	}
	if got := resolveActiveGameweek(events); got != 2 {
		t.Fatalf("expected current GW 2, got %d", got)
	}
	override := 7
	if got := resolveGameweek(events, &override); got != 7 {
		t.Fatalf("expected override GW 7, got %d", got)
	}
}

func TestAvailabilityFromElement(t *testing.T) {
	injured := fpl.Element{Status: "i"}
	chance, badge := availabilityFromElement(injured)
	if chance != 0 || badge != "danger" {
		t.Fatalf("injured: got chance=%d badge=%s", chance, badge)
	}

	seventyFive := 75
	doubt := fpl.Element{Status: "d", ChanceOfPlayingNextRound: &seventyFive}
	chance, badge = availabilityFromElement(doubt)
	if chance != 75 || badge != "warning" {
		t.Fatalf("75%% doubt: got chance=%d badge=%s", chance, badge)
	}
}
