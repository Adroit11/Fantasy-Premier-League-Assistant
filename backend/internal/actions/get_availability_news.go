package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"

	"fpl-assistant/internal/db"
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
	db     *db.Database
}

func NewGetAvailabilityNewsAction(client fpl.Client, database *db.Database) *GetAvailabilityNewsAction {
	return &GetAvailabilityNewsAction{
		client: client,
		db:     database,
	}
}

func (a *GetAvailabilityNewsAction) Execute(ctx context.Context, input GetAvailabilityNewsInput) (*GetAvailabilityNewsOutput, error) {
	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("get_availability_news: failed to fetch bootstrap: %w", err)
	}

	elementMap := buildElementMap(bootstrap.Elements)
	teamMap := buildTeamMap(bootstrap.Teams)

	var targetIDs []int

	if input.TeamID != nil && *input.TeamID > 0 {
		activeGW := resolveActiveGameweek(bootstrap.Events)
		picksResp, err := a.client.GetPicks(ctx, *input.TeamID, activeGW)
		if err == nil && picksResp != nil {
			for _, p := range picksResp.Picks {
				targetIDs = append(targetIDs, p.Element)
			}
		}
	} else if len(input.PlayerIDs) > 0 {
		targetIDs = input.PlayerIDs
	} else {
		for _, el := range bootstrap.Elements {
			if el.News != "" || el.Status != "a" {
				targetIDs = append(targetIDs, el.ID)
			}
		}
	}

	var alerts []PlayerAvailabilityAlert
	var healthy []PlayerAvailabilityAlert
	var historyRows []db.AvailabilityAlertRow
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

		chance, severity := availabilityFromElement(el)
		switch severity {
		case "warning", "caution":
			doubtCount++
		case "danger":
			injuredCount++
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
			PositionName:    positionNames[el.ElementType],
			Status:          el.Status,
			ChanceOfPlaying: chance,
			Severity:        severity,
			News:            el.News,
			NewsAdded:       newsAdded,
		}

		if severity != "success" || el.News != "" {
			alerts = append(alerts, alert)
			historyRows = append(historyRows, db.AvailabilityAlertRow{
				PlayerID:  el.ID,
				WebName:   el.WebName,
				TeamShort: teamShort,
				Status:    el.Status,
				Chance:    chance,
				Severity:  severity,
				News:      el.News,
			})
		} else {
			healthy = append(healthy, alert)
		}
	}

	if err := a.db.LogAvailabilityAlerts(ctx, historyRows); err != nil {
		log.Printf("get_availability_news: persist alerts: %v", err)
	}

	return &GetAvailabilityNewsOutput{
		TotalPlayersChecked: len(targetIDs),
		DoubtCount:          doubtCount,
		InjuredCount:        injuredCount,
		Alerts:              alerts,
		HealthyPlayers:      healthy,
	}, nil
}

func (a *GetAvailabilityNewsAction) Handle(c *fiber.Ctx) error {
	var input GetAvailabilityNewsInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "bad_request",
			"message": err.Error(),
		})
	}

	res, err := a.Execute(c.UserContext(), input)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "execution_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(res)
}
