package scoring

import (
	"testing"

	"fpl-assistant/internal/fpl"
)

func TestFixtureIndex_LookupAndDoubleGW(t *testing.T) {
	gw1, gw2 := 1, 2
	fixtures := []fpl.Fixture{
		{ID: 1, Event: &gw1, TeamH: 1, TeamA: 2, TeamHDifficulty: 2, TeamADifficulty: 4},
		{ID: 2, Event: &gw1, TeamH: 1, TeamA: 3, TeamHDifficulty: 3, TeamADifficulty: 2}, // DGW for team 1
		{ID: 3, Event: &gw2, TeamH: 4, TeamA: 5, TeamHDifficulty: 2, TeamADifficulty: 3},
	}

	idx := NewFixtureIndex(fixtures)

	if got := len(idx.ForTeamGW(1, 1)); got != 2 {
		t.Fatalf("expected double GW (2 fixtures) for team 1 GW1, got %d", got)
	}
	if got := len(idx.ForTeamGW(2, 1)); got != 1 {
		t.Fatalf("expected 1 fixture for team 2 GW1, got %d", got)
	}
	if got := len(idx.ForTeamGW(1, 2)); got != 0 {
		t.Fatalf("expected blank GW for team 1 GW2, got %d", got)
	}
}

func TestCalculateXP_BlankAndDoubleGameweek(t *testing.T) {
	engine := NewXPEngine()
	element := &fpl.Element{
		ID:              101,
		WebName:         "Saka",
		Team:            1,
		ElementType:     3,
		Status:          "a",
		Form:            "7.5",
		PointsPerGame:   "6.2",
		Minutes:         2400,
		ExpectedGoals:   "0.45",
		ExpectedAssists: "0.35",
	}
	teamMap := map[int]fpl.Team{1: {ID: 1, ShortName: "ARS"}, 2: {ID: 2, ShortName: "AVL"}}
	gw1 := 1
	single := []fpl.Fixture{{ID: 1, Event: &gw1, TeamH: 1, TeamA: 2, TeamHDifficulty: 2, TeamADifficulty: 4}}
	double := append(single, fpl.Fixture{ID: 2, Event: &gw1, TeamH: 3, TeamA: 1, TeamHDifficulty: 3, TeamADifficulty: 2})

	singleXP := engine.CalculatePlayerXP(element, teamMap, single, 1)
	doubleXP := engine.CalculatePlayerXP(element, teamMap, double, 1)
	blankXP := engine.CalculatePlayerXP(element, teamMap, nil, 1)

	if blankXP.ProjectedXP != 0 {
		t.Fatalf("expected 0 xP for blank GW, got %.2f", blankXP.ProjectedXP)
	}
	if doubleXP.ProjectedXP <= singleXP.ProjectedXP {
		t.Fatalf("expected double GW xP (%.2f) > single GW xP (%.2f)", doubleXP.ProjectedXP, singleXP.ProjectedXP)
	}
}
