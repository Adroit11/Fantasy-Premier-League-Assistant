package scoring

import (
	"testing"
	"fpl-assistant/internal/fpl"
)

func TestCalculateXP_NormalPlayer(t *testing.T) {
	engine := NewXPEngine()
	element := &fpl.Element{
		ID:            101,
		WebName:       "Saka",
		Team:          1, // Arsenal
		ElementType:   3, // MID
		Status:        "a",
		Form:          "7.5",
		PointsPerGame: "6.2",
		Minutes:       2400,
		ExpectedGoals: "0.45",
		ExpectedAssists: "0.35",
	}

	teamMap := map[int]fpl.Team{
		1: {ID: 1, ShortName: "ARS"},
		2: {ID: 2, ShortName: "AVL"},
	}

	eventGW := 1
	fixtures := []fpl.Fixture{
		{
			ID:              1,
			Event:           &eventGW,
			TeamH:           1,
			TeamA:           2,
			TeamHDifficulty: 2,
			TeamADifficulty: 4,
		},
	}

	proj := engine.CalculatePlayerXP(element, teamMap, fixtures, 1)

	if proj.ProjectedXP <= 0 {
		t.Fatalf("expected positive xP for healthy starter, got %.2f", proj.ProjectedXP)
	}
	if proj.StatusBadge != "success" {
		t.Fatalf("expected status badge success, got %s", proj.StatusBadge)
	}
	if proj.OpponentShortName != "AVL" {
		t.Fatalf("expected opponent AVL, got %s", proj.OpponentShortName)
	}
}

func TestCalculateXP_InjuredPlayer(t *testing.T) {
	engine := NewXPEngine()
	element := &fpl.Element{
		ID:          102,
		WebName:     "De Bruyne",
		Team:        1,
		ElementType: 3,
		Status:      "i", // Injured
		News:        "Hamstring injury",
	}

	teamMap := map[int]fpl.Team{1: {ID: 1, ShortName: "MCI"}}
	proj := engine.CalculatePlayerXP(element, teamMap, nil, 1)

	if proj.ProjectedXP != 0.0 {
		t.Fatalf("expected 0.0 xP for injured player, got %.2f", proj.ProjectedXP)
	}
	if proj.StatusBadge != "danger" {
		t.Fatalf("expected danger status badge, got %s", proj.StatusBadge)
	}
}
