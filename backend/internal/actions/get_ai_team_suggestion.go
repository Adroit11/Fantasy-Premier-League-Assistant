package actions

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/gofiber/fiber/v2"

	"fpl-assistant/internal/ai"
	"fpl-assistant/internal/db"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
)

// ─── DTOs ────────────────────────────────────────────────────────────────────

type GetAITeamSuggestionInput struct {
	TeamID   int  `json:"team_id"`
	Gameweek *int `json:"gameweek,omitempty"`
}

type GetAITeamSuggestionOutput struct {
	TeamID       int                      `json:"team_id"`
	Gameweek     int                      `json:"gameweek"`
	AISuggestion string                   `json:"ai_suggestion"`  // full markdown narrative
	ProjectedXP  float64                  `json:"projected_xp"`
	Formation    string                   `json:"formation"`
	CaptainName  string                   `json:"captain_name"`
	OptimalLineup *scoring.LineupSelection `json:"optimal_lineup"` // raw xP data
	GeneratedAt  string                   `json:"generated_at"`
	AIModel      string                   `json:"ai_model"`
}

// ─── Action ──────────────────────────────────────────────────────────────────

type GetAITeamSuggestionAction struct {
	client    fpl.Client
	engine    scoring.Engine
	optimizer scoring.Optimizer
	db        *db.Database
	ai        ai.AIClient
	aiModel   string
}

func NewGetAITeamSuggestionAction(
	client fpl.Client,
	engine scoring.Engine,
	optimizer scoring.Optimizer,
	database *db.Database,
	aiClient ai.AIClient,
	aiModel string,
) *GetAITeamSuggestionAction {
	return &GetAITeamSuggestionAction{
		client:    client,
		engine:    engine,
		optimizer: optimizer,
		db:        database,
		ai:        aiClient,
		aiModel:   aiModel,
	}
}

func (a *GetAITeamSuggestionAction) Execute(ctx context.Context, input GetAITeamSuggestionInput) (*GetAITeamSuggestionOutput, error) {
	if input.TeamID <= 0 {
		return nil, fmt.Errorf("invalid team_id: must be greater than zero")
	}

	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("get_ai_team_suggestion: fetch bootstrap: %w", err)
	}

	entry, err := a.client.GetEntry(ctx, input.TeamID)
	if err != nil {
		return nil, fmt.Errorf("get_ai_team_suggestion: fetch entry: %w", err)
	}

	fixtures, err := a.client.GetFixtures(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("get_ai_team_suggestion: fetch fixtures: %w", err)
	}

	targetGW := resolveGameweek(bootstrap.Events, input.Gameweek)

	picksResp, err := a.client.GetPicks(ctx, input.TeamID, targetGW)
	if err != nil {
		return nil, fmt.Errorf("get_ai_team_suggestion: fetch picks GW %d: %w", targetGW, err)
	}

	elementMap := buildElementMap(bootstrap.Elements)
	teamMap := buildTeamMap(bootstrap.Teams)
	fixIndex := scoring.NewFixtureIndex(fixtures)

	// Calculate xP projections for squad
	var projections []scoring.PlayerProjection
	var injuryAlerts []string

	for _, pick := range picksResp.Picks {
		el, ok := elementMap[pick.Element]
		if !ok {
			continue
		}
		proj := a.engine.CalculatePlayerXPIndexed(&el, teamMap, fixIndex, targetGW)
		projections = append(projections, proj)

		if el.Status != "a" && el.News != "" {
			injuryAlerts = append(injuryAlerts, fmt.Sprintf("%s (%s): %s", el.WebName, proj.TeamShortName, el.News))
		}
	}

	// Sort projections by xP descending for prompt clarity
	sort.Slice(projections, func(i, j int) bool {
		return projections[i].ProjectedXP > projections[j].ProjectedXP
	})

	// Run optimizer to get formation + captain
	optimal := a.optimizer.OptimizeSquad(projections)

	// Build AI prompt and call LLM
	bankMillions := float64(entry.LastDeadlineBank) / 10.0
	prompt := ai.BuildSelectionPrompt(
		targetGW,
		fmt.Sprintf("%s %s", entry.PlayerFirstName, entry.PlayerLastName),
		projections,
		injuryAlerts,
		bankMillions,
		1, // FPL always provides at least 1 free transfer; extend if entry exposes it
	)

	suggestion, err := a.ai.GenerateTeamSuggestion(ctx, prompt)
	if err != nil {
		log.Printf("[AI] suggestion generation error (non-fatal): %v", err)
		suggestion = "⚠️ AI suggestion temporarily unavailable. Using xP engine recommendation above."
	}

	// Persist to DB (non-fatal if DB is offline)
	if dbErr := a.db.SaveAISuggestion(ctx, db.AISuggestionRow{
		TeamID:      input.TeamID,
		Gameweek:    targetGW,
		Suggestion:  suggestion,
		Formation:   optimal.Formation,
		CaptainName: optimal.CaptainName,
		ProjectedXP: optimal.TotalProjectedXP,
		AIModel:     a.aiModel,
	}); dbErr != nil {
		log.Printf("[AI] persist suggestion: %v", dbErr)
	}

	return &GetAITeamSuggestionOutput{
		TeamID:        input.TeamID,
		Gameweek:      targetGW,
		AISuggestion:  suggestion,
		ProjectedXP:   optimal.TotalProjectedXP,
		Formation:     optimal.Formation,
		CaptainName:   optimal.CaptainName,
		OptimalLineup: optimal,
		GeneratedAt:   "now",
		AIModel:       a.aiModel,
	}, nil
}

func (a *GetAITeamSuggestionAction) Handle(c *fiber.Ctx) error {
	var input GetAITeamSuggestionInput
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
