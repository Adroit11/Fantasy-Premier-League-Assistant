package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
)

type SuggestTransfersInput struct {
	TeamID       int  `json:"team_id"`
	MaxTransfers *int `json:"max_transfers,omitempty"`
	Gameweek     *int `json:"gameweek,omitempty"`
}

type TransferRecommendation struct {
	PlayerOutID    int     `json:"player_out_id"`
	PlayerOutName  string  `json:"player_out_name"`
	PlayerOutTeam  string  `json:"player_out_team"`
	PlayerOutCost  float64 `json:"player_out_cost"`
	PlayerOutXP    float64 `json:"player_out_xp"`
	PlayerOutNews  string  `json:"player_out_news"`
	PlayerInID     int     `json:"player_in_id"`
	PlayerInName   string  `json:"player_in_name"`
	PlayerInTeam   string  `json:"player_in_team"`
	PlayerInCost   float64 `json:"player_in_cost"`
	PlayerInXP     float64 `json:"player_in_xp"`
	NetXPGain      float64 `json:"net_xp_gain"`
	CostDifference float64 `json:"cost_difference"`
	PositionName   string  `json:"position_name"`
	Reason         string  `json:"reason"`
}

type SuggestTransfersOutput struct {
	TeamID          int                      `json:"team_id"`
	Gameweek        int                      `json:"gameweek"`
	BankAvailable   float64                  `json:"bank_available"`
	Recommendations []TransferRecommendation `json:"recommendations"`
	Message         string                   `json:"message"`
}

type SuggestTransfersAction struct {
	client fpl.Client
	engine scoring.Engine
}

func NewSuggestTransfersAction(client fpl.Client, engine scoring.Engine) *SuggestTransfersAction {
	return &SuggestTransfersAction{
		client: client,
		engine: engine,
	}
}

func (a *SuggestTransfersAction) Execute(ctx context.Context, input SuggestTransfersInput) (*SuggestTransfersOutput, error) {
	if input.TeamID <= 0 {
		return nil, fmt.Errorf("invalid team_id: must be greater than zero")
	}

	entry, err := a.client.GetEntry(ctx, input.TeamID)
	if err != nil {
		return nil, fmt.Errorf("suggest_transfers: failed to fetch entry: %w", err)
	}

	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("suggest_transfers: failed to fetch bootstrap: %w", err)
	}

	fixtures, err := a.client.GetFixtures(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("suggest_transfers: failed to fetch fixtures: %w", err)
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
		return nil, fmt.Errorf("suggest_transfers: failed to fetch squad picks for GW %d: %w", targetGW, err)
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

	bankTenths := entry.LastDeadlineBank
	bankMillions := float64(bankTenths) / 10.0

	// Set of current squad IDs
	squadIDs := make(map[int]bool)
	var squadProjections []scoring.PlayerProjection
	for _, p := range picksResp.Picks {
		squadIDs[p.Element] = true
		if el, ok := elementMap[p.Element]; ok {
			proj := a.engine.CalculatePlayerXP(&el, teamMap, fixtures, targetGW)
			squadProjections = append(squadProjections, proj)
		}
	}

	// Group all potential transfers by position
	candidatesByPos := make(map[int][]scoring.PlayerProjection)
	for _, el := range bootstrap.Elements {
		if squadIDs[el.ID] {
			continue
		}
		// Skip long-term injured or unselected
		if el.Status == "i" || el.Status == "u" || el.Status == "s" {
			continue
		}
		proj := a.engine.CalculatePlayerXP(&el, teamMap, fixtures, targetGW)
		if proj.ProjectedXP > 3.5 {
			candidatesByPos[el.ElementType] = append(candidatesByPos[el.ElementType], proj)
		}
	}

	// Sort candidates by projected xP descending
	for pos := range candidatesByPos {
		sort.Slice(candidatesByPos[pos], func(i, j int) bool {
			return candidatesByPos[pos][i].ProjectedXP > candidatesByPos[pos][j].ProjectedXP
		})
	}

	var recs []TransferRecommendation

	for _, outProj := range squadProjections {
		outEl := elementMap[outProj.PlayerID]
		maxAffordableTenths := bankTenths + outEl.NowCost

		candidates := candidatesByPos[outProj.ElementType]
		for _, inProj := range candidates {
			inEl := elementMap[inProj.PlayerID]
			if inEl.NowCost <= maxAffordableTenths {
				xpGain := inProj.ProjectedXP - outProj.ProjectedXP
				if xpGain >= 1.2 || (outEl.Status != "a" && xpGain > 0) {
					reason := fmt.Sprintf("+%.1f projected xP upgrade over upcoming fixture against %s", xpGain, inProj.OpponentShortName)
					if outEl.Status != "a" {
						reason = fmt.Sprintf("Replace injured/doubtful player (%s) with fit starter", outEl.News)
					}

					recs = append(recs, TransferRecommendation{
						PlayerOutID:    outProj.PlayerID,
						PlayerOutName:  outProj.WebName,
						PlayerOutTeam:  outProj.TeamShortName,
						PlayerOutCost:  float64(outEl.NowCost) / 10.0,
						PlayerOutXP:    outProj.ProjectedXP,
						PlayerOutNews:  outEl.News,
						PlayerInID:     inProj.PlayerID,
						PlayerInName:   inProj.WebName,
						PlayerInTeam:   inProj.TeamShortName,
						PlayerInCost:   float64(inEl.NowCost) / 10.0,
						PlayerInXP:     inProj.ProjectedXP,
						NetXPGain:      math.Round(xpGain*10) / 10,
						CostDifference: float64(inEl.NowCost-outEl.NowCost) / 10.0,
						PositionName:   posMap[outProj.ElementType],
						Reason:         reason,
					})
					break // Take top candidate per squad member
				}
			}
		}
	}

	// Sort recommendations by net XP gain descending
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].NetXPGain > recs[j].NetXPGain
	})

	maxRecs := 5
	if input.MaxTransfers != nil && *input.MaxTransfers > 0 {
		maxRecs = *input.MaxTransfers
	}
	if len(recs) > maxRecs {
		recs = recs[:maxRecs]
	}

	return &SuggestTransfersOutput{
		TeamID:          input.TeamID,
		Gameweek:        targetGW,
		BankAvailable:   bankMillions,
		Recommendations: recs,
		Message:         fmt.Sprintf("Found %d high-value transfer recommendations", len(recs)),
	}, nil
}

func (a *SuggestTransfersAction) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var input SuggestTransfersInput
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
