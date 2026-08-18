package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"fpl-assistant/internal/fpl"
)

type GetTeamOverviewInput struct {
	TeamID   int  `json:"team_id"`
	Gameweek *int `json:"gameweek,omitempty"`
}

type SquadPlayer struct {
	ElementID          int     `json:"element_id"`
	WebName            string  `json:"web_name"`
	FullName           string  `json:"full_name"`
	TeamShortName      string  `json:"team_short_name"`
	ElementType        int     `json:"element_type"` // 1=GKP, 2=DEF, 3=MID, 4=FWD
	PositionName       string  `json:"position_name"`
	Cost               float64 `json:"cost"` // £m
	SlotPosition       int     `json:"slot_position"` // 1 to 15
	IsStarting         bool    `json:"is_starting"`   // slots 1-11
	IsCaptain          bool    `json:"is_captain"`
	IsViceCaptain      bool    `json:"is_vice_captain"`
	Multiplier         int     `json:"multiplier"`
	Status             string  `json:"status"`
	StatusBadge        string  `json:"status_badge"` // success, warning, caution, danger
	ChanceOfPlaying    int     `json:"chance_of_playing"`
	News               string  `json:"news"`
	Form               string  `json:"form"`
	PointsPerGame      string  `json:"points_per_game"`
	TotalPoints        int     `json:"total_points"`
	ExpectedGoals      string  `json:"expected_goals"`
	ExpectedAssists    string  `json:"expected_assists"`
}

type GetTeamOverviewOutput struct {
	TeamID          int           `json:"team_id"`
	ManagerName     string        `json:"manager_name"`
	TeamName        string        `json:"team_name"`
	Gameweek        int           `json:"gameweek"`
	Bank            float64       `json:"bank"`
	TeamValue       float64       `json:"team_value"`
	ActiveChip      *string       `json:"active_chip"`
	StartingXI      []SquadPlayer `json:"starting_xi"`
	Bench           []SquadPlayer `json:"bench"`
	AllPlayers      []SquadPlayer `json:"all_players"`
}

type GetTeamOverviewAction struct {
	client fpl.Client
}

func NewGetTeamOverviewAction(client fpl.Client) *GetTeamOverviewAction {
	return &GetTeamOverviewAction{
		client: client,
	}
}

func (a *GetTeamOverviewAction) Execute(ctx context.Context, input GetTeamOverviewInput) (*GetTeamOverviewOutput, error) {
	if input.TeamID <= 0 {
		return nil, fmt.Errorf("invalid team_id: must be greater than zero")
	}

	entry, err := a.client.GetEntry(ctx, input.TeamID)
	if err != nil {
		return nil, fmt.Errorf("get_team_overview: failed to fetch entry: %w", err)
	}

	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("get_team_overview: failed to fetch bootstrap: %w", err)
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
		return nil, fmt.Errorf("get_team_overview: failed to fetch picks for GW %d: %w", targetGW, err)
	}

	// Index elements and teams
	elementMap := make(map[int]fpl.Element)
	for _, el := range bootstrap.Elements {
		elementMap[el.ID] = el
	}
	teamMap := make(map[int]fpl.Team)
	for _, tm := range bootstrap.Teams {
		teamMap[tm.ID] = tm
	}
	posMap := map[int]string{1: "GKP", 2: "DEF", 3: "MID", 4: "FWD"}

	var startingXI []SquadPlayer
	var bench []SquadPlayer
	var allPlayers []SquadPlayer

	for _, pick := range picksResp.Picks {
		el, ok := elementMap[pick.Element]
		if !ok {
			continue
		}

		teamShort := ""
		if tm, ok := teamMap[el.Team]; ok {
			teamShort = tm.ShortName
		}

		chance := 100
		badge := "success"
		if el.ChanceOfPlayingNextRound != nil {
			chance = *el.ChanceOfPlayingNextRound
			switch chance {
			case 100:
				badge = "success"
			case 75:
				badge = "warning"
			case 50, 25:
				badge = "caution"
			case 0:
				badge = "danger"
			}
		} else if el.Status == "i" || el.Status == "s" || el.Status == "u" {
			chance = 0
			badge = "danger"
		} else if el.Status == "d" {
			chance = 50
			badge = "caution"
		}

		player := SquadPlayer{
			ElementID:       el.ID,
			WebName:         el.WebName,
			FullName:        fmt.Sprintf("%s %s", el.FirstName, el.SecondName),
			TeamShortName:   teamShort,
			ElementType:     el.ElementType,
			PositionName:    posMap[el.ElementType],
			Cost:            float64(el.NowCost) / 10.0,
			SlotPosition:    pick.Position,
			IsStarting:      pick.Position <= 11,
			IsCaptain:       pick.IsCaptain,
			IsViceCaptain:   pick.IsViceCaptain,
			Multiplier:      pick.Multiplier,
			Status:          el.Status,
			StatusBadge:     badge,
			ChanceOfPlaying: chance,
			News:            el.News,
			Form:            el.Form,
			PointsPerGame:   el.PointsPerGame,
			TotalPoints:     el.TotalPoints,
			ExpectedGoals:   el.ExpectedGoals,
			ExpectedAssists: el.ExpectedAssists,
		}

		allPlayers = append(allPlayers, player)
		if player.IsStarting {
			startingXI = append(startingXI, player)
		} else {
			bench = append(bench, player)
		}
	}

	return &GetTeamOverviewOutput{
		TeamID:      input.TeamID,
		ManagerName: fmt.Sprintf("%s %s", entry.PlayerFirstName, entry.PlayerLastName),
		TeamName:    entry.Name,
		Gameweek:    targetGW,
		Bank:        float64(entry.LastDeadlineBank) / 10.0,
		TeamValue:   float64(entry.LastDeadlineValue) / 10.0,
		ActiveChip:  picksResp.ActiveChip,
		StartingXI:  startingXI,
		Bench:       bench,
		AllPlayers:  allPlayers,
	}, nil
}

func (a *GetTeamOverviewAction) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var input GetTeamOverviewInput
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
