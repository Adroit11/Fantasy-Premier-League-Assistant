package actions

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"

	"fpl-assistant/internal/ai"
	"fpl-assistant/internal/db"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
)

// ─── DTOs ────────────────────────────────────────────────────────────────────

type GetAIImprovementSuggestionInput struct {
	TeamID   int  `json:"team_id"`
	Gameweek *int `json:"gameweek,omitempty"` // defaults to previous GW
}

type GetAIImprovementSuggestionOutput struct {
	TeamID                int    `json:"team_id"`
	Gameweek              int    `json:"gameweek"`    // GW being reviewed
	NextGameweek          int    `json:"next_gameweek"`
	ImprovementNarrative  string `json:"improvement_narrative"` // AI markdown
	ActualPoints          int    `json:"actual_points"`
	AIProjectedXP         float64 `json:"ai_projected_xp"`
	Delta                 float64 `json:"delta"`
	GeneratedAt           string `json:"generated_at"`
}

// ─── Action ──────────────────────────────────────────────────────────────────

type GetAIImprovementSuggestionAction struct {
	client  fpl.Client
	engine  scoring.Engine
	db      *db.Database
	ai      ai.AIClient
	aiModel string
}

func NewGetAIImprovementSuggestionAction(
	client fpl.Client,
	engine scoring.Engine,
	database *db.Database,
	aiClient ai.AIClient,
	aiModel string,
) *GetAIImprovementSuggestionAction {
	return &GetAIImprovementSuggestionAction{
		client:  client,
		engine:  engine,
		db:      database,
		ai:      aiClient,
		aiModel: aiModel,
	}
}

func (a *GetAIImprovementSuggestionAction) Execute(ctx context.Context, input GetAIImprovementSuggestionInput) (*GetAIImprovementSuggestionOutput, error) {
	if input.TeamID <= 0 {
		return nil, fmt.Errorf("invalid team_id: must be greater than zero")
	}

	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("get_ai_improvement: fetch bootstrap: %w", err)
	}

	entry, err := a.client.GetEntry(ctx, input.TeamID)
	if err != nil {
		return nil, fmt.Errorf("get_ai_improvement: fetch entry: %w", err)
	}

	// Resolve current GW; improvement looks at the *previous* GW result
	currentGW := resolveGameweek(bootstrap.Events, nil)
	reviewGW := currentGW - 1
	if input.Gameweek != nil && *input.Gameweek > 0 {
		reviewGW = *input.Gameweek
	}
	if reviewGW < 1 {
		reviewGW = 1
	}
	nextGW := reviewGW + 1

	managerName := fmt.Sprintf("%s %s", entry.PlayerFirstName, entry.PlayerLastName)

	// Load stored GW score (actual points + AI xP)
	scoreRows, dbErr := a.db.GetGWScores(ctx, input.TeamID, 1)
	if dbErr != nil {
		log.Printf("[get_ai_improvement] load scores: %v", dbErr)
	}

	var actualPoints int
	var aiProjectedXP, delta float64

	if len(scoreRows) > 0 {
		// Use the most recently recorded score as the review GW
		row := scoreRows[0]
		actualPoints = row.ActualPoints
		aiProjectedXP = row.AIProjectedXP
		delta = row.Delta
		reviewGW = row.Gameweek
		nextGW = reviewGW + 1
	} else {
		// Fall back: fetch actual points directly from picks API
		picksResp, pickErr := a.client.GetPicks(ctx, input.TeamID, reviewGW)
		if pickErr == nil {
			actualPoints = picksResp.EntryHistory.Points
		}
	}

	// Load the stored AI suggestion for context
	var aiSuggestionText string
	if suggestion, sErr := a.db.GetAISuggestion(ctx, input.TeamID, reviewGW); sErr == nil && suggestion != nil {
		aiSuggestionText = suggestion.Suggestion
		aiProjectedXP = suggestion.ProjectedXP
	}

	// Get current transfer suggestions as context for next GW
	var transferContext string
	fixtures, _ := a.client.GetFixtures(ctx, nil)
	if fixtures != nil {
		transferContext = a.buildTransferContext(ctx, input.TeamID, nextGW, bootstrap, fixtures)
	}

	// Build and call AI
	prompt := ai.BuildImprovementPrompt(
		reviewGW,
		managerName,
		aiSuggestionText,
		actualPoints,
		aiProjectedXP,
		transferContext,
		nextGW,
	)

	narrative, err := a.ai.GenerateTeamSuggestion(ctx, prompt)
	if err != nil {
		log.Printf("[AI] improvement generation error (non-fatal): %v", err)
		narrative = fmt.Sprintf("⚠️ AI improvement advice temporarily unavailable. GW %d: %d pts scored (xP was %.1f).", reviewGW, actualPoints, aiProjectedXP)
	}

	return &GetAIImprovementSuggestionOutput{
		TeamID:               input.TeamID,
		Gameweek:             reviewGW,
		NextGameweek:         nextGW,
		ImprovementNarrative: narrative,
		ActualPoints:         actualPoints,
		AIProjectedXP:        aiProjectedXP,
		Delta:                delta,
		GeneratedAt:          "now",
	}, nil
}

// buildTransferContext produces a short text summary of top transfer targets for the prompt.
func (a *GetAIImprovementSuggestionAction) buildTransferContext(
	ctx context.Context,
	teamID, nextGW int,
	bootstrap *fpl.BootstrapStatic,
	fixtures []fpl.Fixture,
) string {
	picksResp, err := a.client.GetPicks(ctx, teamID, nextGW-1)
	if err != nil {
		return ""
	}

	elementMap := buildElementMap(bootstrap.Elements)
	teamMap := buildTeamMap(bootstrap.Teams)
	fixIndex := scoring.NewFixtureIndex(fixtures)

	squadIDs := make(map[int]bool)
	var worstOut scoring.PlayerProjection
	for _, pick := range picksResp.Picks {
		squadIDs[pick.Element] = true
		if el, ok := elementMap[pick.Element]; ok {
			proj := a.engine.CalculatePlayerXPIndexed(&el, teamMap, fixIndex, nextGW)
			if proj.ProjectedXP < worstOut.ProjectedXP || worstOut.PlayerID == 0 {
				worstOut = proj
			}
		}
	}

	var sb strings.Builder
	count := 0
	for _, el := range bootstrap.Elements {
		if squadIDs[el.ID] || el.Status == "i" || el.Status == "u" {
			continue
		}
		proj := a.engine.CalculatePlayerXPIndexed(&el, teamMap, fixIndex, nextGW)
		if proj.ProjectedXP > 5.5 && count < 5 {
			sb.WriteString(fmt.Sprintf("- %s (%s, xP: %.1f vs %s)\n", proj.WebName, proj.TeamShortName, proj.ProjectedXP, proj.OpponentShortName))
			count++
		}
	}
	return sb.String()
}

func (a *GetAIImprovementSuggestionAction) Handle(c *fiber.Ctx) error {
	var input GetAIImprovementSuggestionInput
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
