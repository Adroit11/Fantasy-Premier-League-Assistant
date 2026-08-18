package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"fpl-assistant/internal/fpl"
)

type GetAvailabilityNewsInput struct {
	TeamID    *int  `json:"team_id,omitempty"`
	PlayerIDs []int `json:"player_ids,omitempty"`
}

type PlayerAvailabilityAlert struct {
	PlayerID        int    `json:"player_id"`
	WebName         string `json:"web_name"`
	FullName        string `json:"full_name"`
	TeamShortName   string `json:"team_short_name"`
	PositionName    string `json:"position_name"`
	Status          string `json:"status"` // "a", "d", "i", "s", "u"
	ChanceOfPlaying int    `json:"chance_of_playing"`
	Severity        string `json:"severity"` // danger, caution, warning, success
	News            string `json:"news"`
	NewsAdded       string `json:"news_added"`
}

type GetAvailabilityNewsOutput struct {
	TotalPlayersChecked int                       `json:"total_players_checked"`
	DoubtCount          int                       `json:"doubt_count"`
	InjuredCount        int                       `json:"injured_count"`
	Alerts              []PlayerAvailabilityAlert `json:"alerts"`
	HealthyPlayers      []PlayerAvailabilityAlert `json:"healthy_players"`
}

type GetAvailabilityNewsAction struct {
	client fpl.Client
}

func NewGetAvailabilityNewsAction(client fpl.Client) *GetAvailabilityNewsAction {
	return &GetAvailabilityNewsAction{
		client: client,
	}
}

func (a *GetAvailabilityNewsAction) Execute(ctx context.Context, input GetAvailabilityNewsInput) (*GetAvailabilityNewsOutput, error) {
	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("get_availability_news: failed to fetch bootstrap: %w", err)
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
		activeGW := 1
		for _, ev := range bootstrap.Events {
			if ev.IsCurrent {
				activeGW = ev.ID
				break
			} else if ev.IsNext {
				activeGW = ev.ID
			}
		}
		picksResp, err := a.client.GetPicks(ctx, *input.TeamID, activeGW)
		if err == nil && picksResp != nil {
			for _, p := range picksResp.Picks {
				targetIDs = append(targetIDs, p.Element)
			}
		}
	} else if len(input.PlayerIDs) > 0 {
		targetIDs = input.PlayerIDs
	} else {
		// If neither specified, check all players with active news
		for _, el := range bootstrap.Elements {
			if el.News != "" || el.Status != "a" {
				targetIDs = append(targetIDs, el.ID)
			}
		}
	}

	var alerts []PlayerAvailabilityAlert
	var healthy []PlayerAvailabilityAlert
	doubtCount := 0
	injuredCount := 0

	for _, pid := range targetIDs {
		el, ok := elementMap[pid]
		if !ok {
			continue
		}

		teamShort := ""
		if tm, ok := teamMap[el.Team]; ok {
			teamShort = tm.ShortName
		}

		chance := 100
		severity := "success"
		if el.ChanceOfPlayingNextRound != nil {
			chance = *el.ChanceOfPlayingNextRound
			switch chance {
			case 100:
				severity = "success"
			case 75:
				severity = "warning"
				doubtCount++
			case 50, 25:
				severity = "caution"
				doubtCount++
			case 0:
				severity = "danger"
				injuredCount++
			}
		} else if el.Status == "i" || el.Status == "s" || el.Status == "u" {
			chance = 0
			severity = "danger"
			injuredCount++
		} else if el.Status == "d" {
			chance = 50
			severity = "caution"
			doubtCount++
		}

		newsAdded := ""
		if el.NewsAdded != nil {
			newsAdded = *el.NewsAdded
		}

		alert := PlayerAvailabilityAlert{
			PlayerID:        el.ID,
			WebName:         el.WebName,
			FullName:        fmt.Sprintf("%s %s", el.FirstName, el.SecondName),
			TeamShortName:   teamShort,
			PositionName:    posMap[el.ElementType],
			Status:          el.Status,
			ChanceOfPlaying: chance,
			Severity:        severity,
			News:            el.News,
			NewsAdded:       newsAdded,
		}

		if severity != "success" || el.News != "" {
			alerts = append(alerts, alert)
		} else {
			healthy = append(healthy, alert)
		}
	}

	return &GetAvailabilityNewsOutput{
		TotalPlayersChecked: len(targetIDs),
		DoubtCount:          doubtCount,
		InjuredCount:        injuredCount,
		Alerts:              alerts,
		HealthyPlayers:      healthy,
	}, nil
}

func (a *GetAvailabilityNewsAction) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var input GetAvailabilityNewsInput
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
