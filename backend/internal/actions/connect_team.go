package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"fpl-assistant/internal/fpl"
)

// ConnectTeamInput represents the input request for connecting an FPL team
type ConnectTeamInput struct {
	TeamID int `json:"team_id"`
}

// ConnectTeamOutput represents the manager summary and squad info
type ConnectTeamOutput struct {
	TeamID           int                  `json:"team_id"`
	ManagerName      string               `json:"manager_name"`
	TeamName         string               `json:"team_name"`
	OverallRank      int                  `json:"overall_rank"`
	TotalPoints      int                  `json:"total_points"`
	CurrentGameweek  int                  `json:"current_gameweek"`
	Bank             float64              `json:"bank"`       // in millions (e.g. 1.5m)
	TeamValue        float64              `json:"team_value"` // in millions (e.g. 102.5m)
	ActiveChip       *string              `json:"active_chip"`
	ClassicLeagues   []fpl.ClassicLeague  `json:"classic_leagues"`
	PicksCount       int                  `json:"picks_count"`
	Message          string               `json:"message"`
}

// ConnectTeamAction handles manager authentication & initial squad sync
type ConnectTeamAction struct {
	client fpl.Client
}

func NewConnectTeamAction(client fpl.Client) *ConnectTeamAction {
	return &ConnectTeamAction{
		client: client,
	}
}

func (a *ConnectTeamAction) Execute(ctx context.Context, input ConnectTeamInput) (*ConnectTeamOutput, error) {
	if input.TeamID <= 0 {
		return nil, fmt.Errorf("invalid team_id: must be greater than zero")
	}

	entry, err := a.client.GetEntry(ctx, input.TeamID)
	if err != nil {
		return nil, fmt.Errorf("connect_team: failed to load manager profile: %w", err)
	}

	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect_team: failed to load bootstrap data: %w", err)
	}

	// Determine current or next active gameweek
	activeGW := 1
	for _, ev := range bootstrap.Events {
		if ev.IsCurrent {
			activeGW = ev.ID
			break
		} else if ev.IsNext {
			activeGW = ev.ID
		}
	}

	picksResp, err := a.client.GetPicks(ctx, input.TeamID, activeGW)
	var activeChip *string
	picksCount := 0
	if err == nil && picksResp != nil {
		activeChip = picksResp.ActiveChip
		picksCount = len(picksResp.Picks)
	}

	bankMillions := float64(entry.LastDeadlineBank) / 10.0
	valueMillions := float64(entry.LastDeadlineValue) / 10.0

	return &ConnectTeamOutput{
		TeamID:          input.TeamID,
		ManagerName:     fmt.Sprintf("%s %s", entry.PlayerFirstName, entry.PlayerLastName),
		TeamName:        entry.Name,
		OverallRank:     entry.SummaryOverallRank,
		TotalPoints:     entry.SummaryOverallPoints,
		CurrentGameweek: activeGW,
		Bank:            bankMillions,
		TeamValue:       valueMillions,
		ActiveChip:      activeChip,
		ClassicLeagues:  entry.Leagues.Classic,
		PicksCount:      picksCount,
		Message:         fmt.Sprintf("Successfully connected team: %s", entry.Name),
	}, nil
}

func (a *ConnectTeamAction) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var input ConnectTeamInput
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
