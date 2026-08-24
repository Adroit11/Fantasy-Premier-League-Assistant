package actions

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"fpl-assistant/internal/db"
	"fpl-assistant/internal/fpl"
)

// ─── DTOs ────────────────────────────────────────────────────────────────────

type GetGWScoreHistoryInput struct {
	TeamID int `json:"team_id"`
	Limit  int `json:"limit,omitempty"` // default 38 (full season)
}

type GWScoreHistoryEntry struct {
	Gameweek      int     `json:"gameweek"`
	ActualPoints  int     `json:"actual_points"`
	AIProjectedXP float64 `json:"ai_projected_xp"`
	Delta         float64 `json:"delta"`
	Status        string  `json:"status"` // "beat_ai" | "below_ai" | "on_track"
}

type GetGWScoreHistoryOutput struct {
	TeamID          int                   `json:"team_id"`
	TotalGWsPlayed  int                   `json:"total_gws_played"`
	TotalActual     int                   `json:"total_actual"`
	TotalAIXP       float64               `json:"total_ai_xp"`
	OverallDelta    float64               `json:"overall_delta"`
	History         []GWScoreHistoryEntry `json:"history"`
}

// ─── Action ──────────────────────────────────────────────────────────────────

type GetGWScoreHistoryAction struct {
	client fpl.Client
	db     *db.Database
}

func NewGetGWScoreHistoryAction(client fpl.Client, database *db.Database) *GetGWScoreHistoryAction {
	return &GetGWScoreHistoryAction{
		client: client,
		db:     database,
	}
}

func (a *GetGWScoreHistoryAction) Execute(ctx context.Context, input GetGWScoreHistoryInput) (*GetGWScoreHistoryOutput, error) {
	if input.TeamID <= 0 {
		return nil, fmt.Errorf("invalid team_id: must be greater than zero")
	}

	rows, err := a.db.GetGWScores(ctx, input.TeamID, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("get_gw_score_history: %w", err)
	}

	var history []GWScoreHistoryEntry
	var totalActual int
	var totalXP float64

	for _, r := range rows {
		status := "on_track"
		if r.Delta > 2 {
			status = "beat_ai"
		} else if r.Delta < -2 {
			status = "below_ai"
		}

		history = append(history, GWScoreHistoryEntry{
			Gameweek:      r.Gameweek,
			ActualPoints:  r.ActualPoints,
			AIProjectedXP: r.AIProjectedXP,
			Delta:         r.Delta,
			Status:        status,
		})
		totalActual += r.ActualPoints
		totalXP += r.AIProjectedXP
	}

	return &GetGWScoreHistoryOutput{
		TeamID:         input.TeamID,
		TotalGWsPlayed: len(history),
		TotalActual:    totalActual,
		TotalAIXP:      totalXP,
		OverallDelta:   float64(totalActual) - totalXP,
		History:        history,
	}, nil
}

func (a *GetGWScoreHistoryAction) Handle(c *fiber.Ctx) error {
	teamID, err := c.ParamsInt("team_id", 0)
	if err != nil || teamID <= 0 {
		// Also accept JSON body for consistency
		var input GetGWScoreHistoryInput
		if parseErr := c.BodyParser(&input); parseErr != nil || input.TeamID <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "bad_request",
				"message": "team_id is required",
			})
		}
		teamID = input.TeamID
	}

	limit := c.QueryInt("limit", 38)

	res, err := a.Execute(c.UserContext(), GetGWScoreHistoryInput{
		TeamID: teamID,
		Limit:  limit,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "execution_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(res)
}
