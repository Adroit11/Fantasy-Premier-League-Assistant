package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
)

type SuggestLineupInput struct {
	TeamID   int  `json:"team_id"`
	Gameweek *int `json:"gameweek,omitempty"`
}

type SuggestLineupOutput struct {
	TeamID           int                      `json:"team_id"`
	Gameweek         int                      `json:"gameweek"`
	OptimalLineup    *scoring.LineupSelection `json:"optimal_lineup"`
	CurrentLineupXP  float64                  `json:"current_lineup_xp"`
	OptimizationGain float64                  `json:"optimization_gain"` // XP gained over current pick
	CaptainDelta     string                   `json:"captain_delta"`
	Recommendations  []string                 `json:"recommendations"`
}

type SuggestLineupAction struct {
	client    fpl.Client
	engine    scoring.Engine
	optimizer scoring.Optimizer
}

func NewSuggestLineupAction(client fpl.Client, engine scoring.Engine, optimizer scoring.Optimizer) *SuggestLineupAction {
	return &SuggestLineupAction{
		client:    client,
		engine:    engine,
		optimizer: optimizer,
	}
}

func (a *SuggestLineupAction) Execute(ctx context.Context, input SuggestLineupInput) (*SuggestLineupOutput, error) {
	if input.TeamID <= 0 {
		return nil, fmt.Errorf("invalid team_id: must be greater than zero")
	}

	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("suggest_lineup: failed to fetch bootstrap: %w", err)
	}

	fixtures, err := a.client.GetFixtures(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("suggest_lineup: failed to fetch fixtures: %w", err)
	}

	targetGW := 1
	if input.Gameweek != nil && *input.Gameweek > 0 {
		targetGW = *input.Gameweek
	} else {
		for _, ev := range bootstrap.Events {
			if ev.IsCurrent {
				targetGW = ev.ID
				break
			} else if ev.IsNext {
				targetGW = ev.ID
			}
		}
	}

	picksResp, err := a.client.GetPicks(ctx, input.TeamID, targetGW)
	if err != nil {
		return nil, fmt.Errorf("suggest_lineup: failed to fetch squad picks for GW %d: %w", targetGW, err)
	}

	elementMap := make(map[int]fpl.Element)
	for _, el := range bootstrap.Elements {
		elementMap[el.ID] = el
	}
	teamMap := make(map[int]fpl.Team)
	for _, tm := range bootstrap.Teams {
		teamMap[tm.ID] = tm
	}

	var projections []scoring.PlayerProjection
	var currentPicksXP float64

	for _, pick := range picksResp.Picks {
		el, ok := elementMap[pick.Element]
		if !ok {
			continue
		}

		proj := a.engine.CalculatePlayerXP(&el, teamMap, fixtures, targetGW)
		projections = append(projections, proj)

		// Calculate current pick lineup XP
		if pick.Position <= 11 {
			multiplier := pick.Multiplier
			if multiplier <= 0 {
				multiplier = 1
			}
			currentPicksXP += proj.ProjectedXP * float64(multiplier)
		}
	}

	if len(projections) < 15 {
		return nil, fmt.Errorf("suggest_lineup: incomplete squad (found %d players, expected 15)", len(projections))
	}

	optimal := a.optimizer.OptimizeSquad(projections)
	gain := optimal.TotalProjectedXP - currentPicksXP
	if gain < 0 {
		gain = 0
	}

	var recs []string
	recs = append(recs, fmt.Sprintf("Recommended formation: %s for optimal expected points.", optimal.Formation))
	recs = append(recs, fmt.Sprintf("Captain %s (Projected: %.1f pts) and Vice-Captain %s.", optimal.CaptainName, optimal.StartingXI[0].ProjectedXP, optimal.ViceCaptainName))
	if len(optimal.Bench) > 1 {
		recs = append(recs, fmt.Sprintf("Priority 1 Bench Substitute: %s (xP: %.1f).", optimal.Bench[1].WebName, optimal.Bench[1].ProjectedXP))
	}

	return &SuggestLineupOutput{
		TeamID:           input.TeamID,
		Gameweek:         targetGW,
		OptimalLineup:    optimal,
		CurrentLineupXP:  currentPicksXP,
		OptimizationGain: gain,
		CaptainDelta:     fmt.Sprintf("Recommended Captain: %s", optimal.CaptainName),
		Recommendations:  recs,
	}, nil
}

func (a *SuggestLineupAction) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var input SuggestLineupInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"bad_request","message":%q}`, err.Error()), http.StatusBadRequest)
		return
	}

	res, err := a.Execute(r.Context(), input)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"execution_failed","message":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}
