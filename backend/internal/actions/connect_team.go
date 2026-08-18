package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"

	"fpl-assistant/internal/db"
	"fpl-assistant/internal/fpl"
)

// ConnectTeamInput represents the input request for connecting an FPL team
type ConnectTeamInput struct {
	TeamID int `json:"team_id"`
}

// ConnectTeamOutput represents the manager summary and squad info
type ConnectTeamOutput struct {
	TeamID          int                 `json:"team_id"`
	ManagerName     string              `json:"manager_name"`
	TeamName        string              `json:"team_name"`
	OverallRank     int                 `json:"overall_rank"`
	TotalPoints     int                 `json:"total_points"`
	CurrentGameweek int                 `json:"current_gameweek"`
	Bank            float64             `json:"bank"`       // in millions (e.g. 1.5m)
	TeamValue       float64             `json:"team_value"` // in millions (e.g. 102.5m)
	ActiveChip      *string             `json:"active_chip"`
	ClassicLeagues  []fpl.ClassicLeague `json:"classic_leagues"`
	PicksCount      int                 `json:"picks_count"`
	Message         string              `json:"message"`
}

// ConnectTeamAction handles manager authentication, initial squad sync, and MySQL persistence
type ConnectTeamAction struct {
	client fpl.Client
	db     *db.Database
}

func NewConnectTeamAction(client fpl.Client, database *db.Database) *ConnectTeamAction {
	return &ConnectTeamAction{
		client: client,
		db:     database,
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

	activeGW := resolveActiveGameweek(bootstrap.Events)

	picksResp, err := a.client.GetPicks(ctx, input.TeamID, activeGW)
	var activeChip *string
	picksCount := 0
	if err == nil && picksResp != nil {
		activeChip = picksResp.ActiveChip
		picksCount = len(picksResp.Picks)
	}

	bankMillions := float64(entry.LastDeadlineBank) / 10.0
	valueMillions := float64(entry.LastDeadlineValue) / 10.0
	managerName := fmt.Sprintf("%s %s", entry.PlayerFirstName, entry.PlayerLastName)

	if err := a.db.SaveManagerTeam(ctx, input.TeamID, managerName, entry.Name, entry.SummaryOverallRank, entry.SummaryOverallPoints, activeGW, bankMillions, valueMillions); err != nil {
		log.Printf("connect_team: persist manager: %v", err)
	}

	return &ConnectTeamOutput{
		TeamID:          input.TeamID,
		ManagerName:     managerName,
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

// Handle Fiber HTTP Endpoint Adapter
func (a *ConnectTeamAction) Handle(c *fiber.Ctx) error {
	var input ConnectTeamInput
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
