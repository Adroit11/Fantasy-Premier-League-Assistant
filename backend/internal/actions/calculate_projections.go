package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
)

type CalculateProjectionsInput struct {
	TeamID    *int  `json:"team_id,omitempty"`
	PlayerIDs []int `json:"player_ids,omitempty"`
	StartGW   *int  `json:"start_gw,omitempty"`
	EndGW     *int  `json:"end_gw,omitempty"`
}

type PlayerGWProjectionSummary struct {
	PlayerID      int                        `json:"player_id"`
	WebName       string                     `json:"web_name"`
	TeamShortName string                     `json:"team_short_name"`
	PositionName  string                     `json:"position_name"`
	ElementType   int                        `json:"element_type"`
	Cost          float64                    `json:"cost"`
	TotalXP       float64                    `json:"total_xp"`
	AverageXP     float64                    `json:"average_xp"`
	Projections   []scoring.PlayerProjection `json:"projections"`
}

type CalculateProjectionsOutput struct {
	StartGW     int                         `json:"start_gw"`
	EndGW       int                         `json:"end_gw"`
	TotalXP     float64                     `json:"total_squad_xp"`
	PlayerCount int                         `json:"player_count"`
	Players     []PlayerGWProjectionSummary `json:"players"`
}

type CalculateProjectionsAction struct {
	client  fpl.Client
	engine  scoring.Engine
}

func NewCalculateProjectionsAction(client fpl.Client, engine scoring.Engine) *CalculateProjectionsAction {
	return &CalculateProjectionsAction{
		client: client,
		engine: engine,
	}
}

func (a *CalculateProjectionsAction) Execute(ctx context.Context, input CalculateProjectionsInput) (*CalculateProjectionsOutput, error) {
	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("calculate_projections: failed to fetch bootstrap: %w", err)
	}

	fixtures, err := a.client.GetFixtures(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("calculate_projections: failed to fetch fixtures: %w", err)
	}

	// Determine start and end GW
	currentGW := 1
	for _, ev := range bootstrap.Events {
		if ev.IsCurrent {
			currentGW = ev.ID
			break
		} else if ev.IsNext {
			currentGW = ev.ID
		}
	}

	startGW := currentGW
	if input.StartGW != nil && *input.StartGW > 0 {
		startGW = *input.StartGW
	}

	endGW := startGW + 4
	if input.EndGW != nil && *input.EndGW >= startGW {
		endGW = *input.EndGW
	}
	if endGW > 38 {
		endGW = 38
	}

	elementMap := make(map[int]fpl.Element)
	for _, el := range bootstrap.Elements {
		elementMap[el.ID] = el
	}
	teamMap := make(map[int]fpl.Team)
	for _, tm := range bootstrap.Teams {
		teamMap[tm.ID] = tm
	}
	posMap := map[int]string{1: "GKP", 2: "DEF", 3: "MID", 4: "FWD"}

	var targetIDs []int
	if input.TeamID != nil && *input.TeamID > 0 {
		picksResp, err := a.client.GetPicks(ctx, *input.TeamID, currentGW)
		if err == nil && picksResp != nil {
			for _, p := range picksResp.Picks {
				targetIDs = append(targetIDs, p.Element)
			}
		}
	} else if len(input.PlayerIDs) > 0 {
		targetIDs = input.PlayerIDs
	} else {
		// Default top 20 popular players
		for i, el := range bootstrap.Elements {
			if i >= 20 {
				break
			}
			targetIDs = append(targetIDs, el.ID)
		}
	}

	var summaries []PlayerGWProjectionSummary
	totalSquadXP := 0.0

	for _, pid := range targetIDs {
		el, ok := elementMap[pid]
		if !ok {
			continue
		}

		teamShort := ""
		if tm, ok := teamMap[el.Team]; ok {
			teamShort = tm.ShortName
		}

		var playerProjs []scoring.PlayerProjection
		playerTotalXP := 0.0

		for gw := startGW; gw <= endGW; gw++ {
			proj := a.engine.CalculatePlayerXP(&el, teamMap, fixtures, gw)
			playerProjs = append(playerProjs, proj)
			playerTotalXP += proj.ProjectedXP
		}

		gwCount := float64(endGW - startGW + 1)
		avgXP := 0.0
		if gwCount > 0 {
			avgXP = playerTotalXP / gwCount
		}

		totalSquadXP += playerTotalXP

		summaries = append(summaries, PlayerGWProjectionSummary{
			PlayerID:      el.ID,
			WebName:       el.WebName,
			TeamShortName: teamShort,
			PositionName:  posMap[el.ElementType],
			ElementType:   el.ElementType,
			Cost:          float64(el.NowCost) / 10.0,
			TotalXP:       playerTotalXP,
			AverageXP:     avgXP,
			Projections:   playerProjs,
		})
	}

	return &CalculateProjectionsOutput{
		StartGW:     startGW,
		EndGW:       endGW,
		TotalXP:     totalSquadXP,
		PlayerCount: len(summaries),
		Players:     summaries,
	}, nil
}

func (a *CalculateProjectionsAction) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var input CalculateProjectionsInput
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
