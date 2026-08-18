package scoring

import "fpl-assistant/internal/fpl"

// FixtureIndex is a precomputed lookup of fixtures by team and gameweek.
// Building once avoids an O(fixtures) scan on every CalculatePlayerXP call
// (e.g. ~700 players × 380 fixtures on suggest_transfers).
type FixtureIndex struct {
	byTeamGW map[int]map[int][]fpl.Fixture
}

// NewFixtureIndex indexes fixtures as teamID -> gameweek -> []Fixture.
func NewFixtureIndex(fixtures []fpl.Fixture) *FixtureIndex {
	idx := &FixtureIndex{
		byTeamGW: make(map[int]map[int][]fpl.Fixture, 20),
	}
	for _, fix := range fixtures {
		if fix.Event == nil {
			continue
		}
		gw := *fix.Event
		idx.add(fix.TeamH, gw, fix)
		idx.add(fix.TeamA, gw, fix)
	}
	return idx
}

func (idx *FixtureIndex) add(teamID, gw int, fix fpl.Fixture) {
	if idx.byTeamGW[teamID] == nil {
		idx.byTeamGW[teamID] = make(map[int][]fpl.Fixture, 38)
	}
	idx.byTeamGW[teamID][gw] = append(idx.byTeamGW[teamID][gw], fix)
}

// ForTeamGW returns the fixtures (including blanks/doubles) for a club in a gameweek.
func (idx *FixtureIndex) ForTeamGW(teamID, gw int) []fpl.Fixture {
	if idx == nil {
		return nil
	}
	return idx.byTeamGW[teamID][gw]
}
