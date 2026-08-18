package scoring

import (
	"fpl-assistant/internal/fpl"
	"math"
	"strconv"
)

// PlayerProjection holds projected points and fixture context for a specific gameweek
type PlayerProjection struct {
	PlayerID           int     `json:"player_id"`
	WebName            string  `json:"web_name"`
	TeamShortName      string  `json:"team_short_name"`
	ElementType        int     `json:"element_type"` // 1=GKP, 2=DEF, 3=MID, 4=FWD
	Gameweek           int     `json:"gameweek"`
	OpponentShortName  string  `json:"opponent_short_name"`
	IsHome             bool    `json:"is_home"`
	FDR                int     `json:"fdr"`
	AvailabilityChance int     `json:"availability_chance"`
	StatusBadge        string  `json:"status_badge"` // success, warning, caution, danger
	ProjectedXP        float64 `json:"projected_xp"`
	ProjectedGoals     float64 `json:"projected_goals"`
	ProjectedAssists   float64 `json:"projected_assists"`
	CleanSheetProb     float64 `json:"clean_sheet_prob"`
}

type Engine interface {
	CalculatePlayerXP(element *fpl.Element, teamMap map[int]fpl.Team, fixtures []fpl.Fixture, targetGW int) PlayerProjection
	CalculatePlayerXPIndexed(element *fpl.Element, teamMap map[int]fpl.Team, idx *FixtureIndex, targetGW int) PlayerProjection
	CalculateMultiGWProjections(elements []*fpl.Element, teamMap map[int]fpl.Team, fixtures []fpl.Fixture, startGW int, endGW int) map[int][]PlayerProjection
}

type XPEngine struct{}

func NewXPEngine() *XPEngine {
	return &XPEngine{}
}

// CalculatePlayerXP projects points for a single player in a single gameweek.
// Prefer CalculatePlayerXPIndexed when scoring many players against the same fixture list.
func (e *XPEngine) CalculatePlayerXP(element *fpl.Element, teamMap map[int]fpl.Team, fixtures []fpl.Fixture, targetGW int) PlayerProjection {
	return e.CalculatePlayerXPIndexed(element, teamMap, NewFixtureIndex(fixtures), targetGW)
}

// CalculatePlayerXPIndexed is the hot-path xP calculator. Callers MUST build FixtureIndex once.
func (e *XPEngine) CalculatePlayerXPIndexed(element *fpl.Element, teamMap map[int]fpl.Team, idx *FixtureIndex, targetGW int) PlayerProjection {
	proj := PlayerProjection{
		PlayerID:           element.ID,
		WebName:            element.WebName,
		ElementType:        element.ElementType,
		Gameweek:           targetGW,
		AvailabilityChance: 100,
		StatusBadge:        "success",
	}

	if myTeam, ok := teamMap[element.Team]; ok {
		proj.TeamShortName = myTeam.ShortName
	}

	// 1. Calculate Availability Factor
	availFactor := 1.0
	if element.ChanceOfPlayingNextRound != nil {
		proj.AvailabilityChance = *element.ChanceOfPlayingNextRound
		switch *element.ChanceOfPlayingNextRound {
		case 100:
			availFactor = 1.0
			proj.StatusBadge = "success"
		case 75:
			availFactor = 0.75
			proj.StatusBadge = "warning"
		case 50:
			availFactor = 0.50
			proj.StatusBadge = "caution"
		case 25:
			availFactor = 0.25
			proj.StatusBadge = "caution"
		case 0:
			availFactor = 0.0
			proj.StatusBadge = "danger"
		}
	} else if element.Status == "i" || element.Status == "s" || element.Status == "u" {
		availFactor = 0.0
		proj.AvailabilityChance = 0
		proj.StatusBadge = "danger"
	} else if element.Status == "d" {
		availFactor = 0.5
		proj.AvailabilityChance = 50
		proj.StatusBadge = "caution"
	}

	if availFactor == 0.0 {
		proj.ProjectedXP = 0.0
		return proj
	}

	gwFixtures := idx.ForTeamGW(element.Team, targetGW)

	// Blank Gameweek
	if len(gwFixtures) == 0 {
		proj.ProjectedXP = 0.0
		proj.OpponentShortName = "BLANK"
		proj.FDR = 0
		return proj
	}

	totalXP := 0.0
	for _, fix := range gwFixtures {
		isHome := fix.TeamH == element.Team
		fdr := fix.TeamADifficulty
		oppTeamID := fix.TeamA
		if !isHome {
			fdr = fix.TeamHDifficulty
			oppTeamID = fix.TeamH
		}

		proj.IsHome = isHome
		proj.FDR = fdr
		if oppTeam, ok := teamMap[oppTeamID]; ok {
			proj.OpponentShortName = oppTeam.ShortName
		}

		// FDR Multiplier (lower FDR = easier match = higher score)
		fdrMultiplier := 1.0 + float64(3-fdr)*0.08
		if !isHome {
			fdrMultiplier = 0.95 + float64(3-fdr)*0.08
		}

		// Base Appearance Points (typically 2 for starters)
		basePoints := 2.0
		if element.Minutes > 0 && float64(element.Minutes)/38.0 < 45.0 {
			basePoints = 1.0
		}

		// Parse stats
		formVal, _ := strconv.ParseFloat(element.Form, 64)
		ppgVal, _ := strconv.ParseFloat(element.PointsPerGame, 64)
		xG, _ := strconv.ParseFloat(element.ExpectedGoals, 64)
		xA, _ := strconv.ParseFloat(element.ExpectedAssists, 64)

		if ppgVal <= 0 {
			ppgVal = 2.0
		}

		// Baseline Form Contribution
		formContribution := (formVal*0.6 + ppgVal*0.4) * 0.4

		// Attacking Points Calculation based on Position
		attackingXP := 0.0
		switch element.ElementType {
		case 1: // GKP
			attackingXP = (xG*6.0 + xA*3.0) * 0.05
		case 2: // DEF
			attackingXP = (xG*6.0 + xA*3.0) * 0.08
		case 3: // MID
			attackingXP = (xG*5.0 + xA*3.0) * 0.12
		case 4: // FWD
			attackingXP = (xG*4.0 + xA*3.0) * 0.15
		}

		// Clean sheet odds estimation based on team strength and FDR
		csOdds := math.Max(0.05, math.Min(0.65, 0.45-float64(fdr-1)*0.08))
		if !isHome {
			csOdds *= 0.85
		}
		proj.CleanSheetProb = math.Round(csOdds*100) / 100

		defensiveXP := 0.0
		switch element.ElementType {
		case 1, 2: // GKP, DEF
			defensiveXP = csOdds * 4.0
		case 3: // MID
			defensiveXP = csOdds * 1.0
		}

		// Bonus probability
		bonusXP := math.Min(1.5, formVal*0.1)

		// Aggregate match xP
		matchXP := (basePoints + formContribution + attackingXP + defensiveXP + bonusXP) * fdrMultiplier
		totalXP += matchXP
	}

	// Apply final availability factor and round to 1 decimal place
	finalXP := totalXP * availFactor
	proj.ProjectedXP = math.Round(finalXP*10) / 10

	return proj
}

// CalculateMultiGWProjections calculates xP across multiple upcoming gameweeks
func (e *XPEngine) CalculateMultiGWProjections(elements []*fpl.Element, teamMap map[int]fpl.Team, fixtures []fpl.Fixture, startGW int, endGW int) map[int][]PlayerProjection {
	result := make(map[int][]PlayerProjection)
	idx := NewFixtureIndex(fixtures)

	for _, elem := range elements {
		var projections []PlayerProjection
		for gw := startGW; gw <= endGW; gw++ {
			p := e.CalculatePlayerXPIndexed(elem, teamMap, idx, gw)
			projections = append(projections, p)
		}
		result[elem.ID] = projections
	}

	return result
}
