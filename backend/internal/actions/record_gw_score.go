package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"

	"fpl-assistant/internal/db"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
)

// ─── DTOs ────────────────────────────────────────────────────────────────────

type RecordGWScoreInput struct {
	TeamID   int  `json:"team_id"`
	Gameweek *int `json:"gameweek,omitempty"`
}

type RecordGWScoreOutput struct {
	TeamID        int     `json:"team_id"`
	Gameweek      int     `json:"gameweek"`
	ActualPoints  int     `json:"actual_points"`
	AIProjectedXP float64 `json:"ai_projected_xp"`
	Delta         float64 `json:"delta"`        // actual - ai_projected
	RecordedAt    string  `json:"recorded_at"`
	Message       string  `json:"message"`
}

// ─── Action ──────────────────────────────────────────────────────────────────

type RecordGWScoreAction struct {
	client fpl.Client
	engine scoring.Engine
	db     *db.Database
}

func NewRecordGWScoreAction(client fpl.Client, engine scoring.Engine, database *db.Database) *RecordGWScoreAction {
	return &RecordGWScoreAction{
		client: client,
		engine: engine,
		db:     database,
	}
}

func (a *RecordGWScoreAction) Execute(ctx context.Context, input RecordGWScoreInput) (*RecordGWScoreOutput, error) {
	if input.TeamID <= 0 {
		return nil, fmt.Errorf("invalid team_id: must be greater than zero")
	}

	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("record_gw_score: fetch bootstrap: %w", err)
	}

	targetGW := resolveGameweek(bootstrap.Events, input.Gameweek)

	// Fetch picks for the target GW — EntryHistory embedded contains actual points
	picksResp, err := a.client.GetPicks(ctx, input.TeamID, targetGW)
	if err != nil {
		return nil, fmt.Errorf("record_gw_score: fetch picks GW %d: %w", targetGW, err)
	}

	actualPoints := picksResp.EntryHistory.Points

	// Load AI projected xP from DB (may be zero if suggestion was never generated)
	var aiProjectedXP float64
	if suggestion, dbErr := a.db.GetAISuggestion(ctx, input.TeamID, targetGW); dbErr != nil {
		log.Printf("[record_gw_score] load AI suggestion: %v", dbErr)
	} else if suggestion != nil {
		aiProjectedXP = suggestion.ProjectedXP
	}

	delta := float64(actualPoints) - aiProjectedXP

	// Persist the score record
	if dbErr := a.db.UpsertGWScore(ctx, db.GWScoreRow{
		TeamID:        input.TeamID,
		Gameweek:      targetGW,
		ActualPoints:  actualPoints,
		AIProjectedXP: aiProjectedXP,
		Delta:         delta,
	}); dbErr != nil {
		log.Printf("[record_gw_score] persist: %v", dbErr)
	}

	msg := fmt.Sprintf("GW %d recorded: %d pts actual vs %.1f xP projected (Δ %.1f)", targetGW, actualPoints, aiProjectedXP, delta)

	return &RecordGWScoreOutput{
		TeamID:        input.TeamID,
		Gameweek:      targetGW,
		ActualPoints:  actualPoints,
		AIProjectedXP: aiProjectedXP,
		Delta:         delta,
		RecordedAt:    "now",
		Message:       msg,
	}, nil
}

func (a *RecordGWScoreAction) Handle(c *fiber.Ctx) error {
	var input RecordGWScoreInput
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
