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

type SuggestLineupInput struct {
	TeamID   int  `json:"team_id"`
	Gameweek *int `json:"gameweek,omitempty"`
}

type SuggestLineupOutput struct {
	TeamID           int                      `json:"team_id"`
	Gameweek         int                      `json:"gameweek"`
	OptimalLineup    *scoring.LineupSelection `json:"optimal_lineup"`
	CurrentLineupXP  float64                  `json:"current_lineup_xp"`
	OptimizationGain float64                  `json:"optimization_gain"` // XP gained over current pick
	CaptainDelta     string                   `json:"captain_delta"`
	Recommendations  []string                 `json:"recommendations"`
}

type SuggestLineupAction struct {
	client    fpl.Client
	engine    scoring.Engine
	optimizer scoring.Optimizer
	db        *db.Database
}

func NewSuggestLineupAction(client fpl.Client, engine scoring.Engine, optimizer scoring.Optimizer, database *db.Database) *SuggestLineupAction {
	return &SuggestLineupAction{
		client:    client,
		engine:    engine,
		optimizer: optimizer,
		db:        database,
	}
}

func (a *SuggestLineupAction) Execute(ctx context.Context, input SuggestLineupInput) (*SuggestLineupOutput, error) {
	if input.TeamID <= 0 {
		return nil, fmt.Errorf("invalid team_id: must be greater than zero")
	}

	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("suggest_lineup: failed to fetch bootstrap: %w", err)
	}

	fixtures, err := a.client.GetFixtures(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("suggest_lineup: failed to fetch fixtures: %w", err)
	}

	targetGW := resolveGameweek(bootstrap.Events, input.Gameweek)

	picksResp, err := a.client.GetPicks(ctx, input.TeamID, targetGW)
	if err != nil {
		return nil, fmt.Errorf("suggest_lineup: failed to fetch squad picks for GW %d: %w", targetGW, err)
	}

	elementMap := buildElementMap(bootstrap.Elements)
	teamMap := buildTeamMap(bootstrap.Teams)
	fixIndex := scoring.NewFixtureIndex(fixtures)

	var projections []scoring.PlayerProjection
	var currentPicksXP float64

	for _, pick := range picksResp.Picks {
		el, ok := elementMap[pick.Element]
		if !ok {
			continue
		}

		proj := a.engine.CalculatePlayerXPIndexed(&el, teamMap, fixIndex, targetGW)
		projections = append(projections, proj)

		if pick.Position <= 11 {
			multiplier := pick.Multiplier
			if multiplier <= 0 {
				multiplier = 1
			}
			currentPicksXP += proj.ProjectedXP * float64(multiplier)
		}
	}

	if len(projections) < 15 {
		return nil, fmt.Errorf("suggest_lineup: incomplete squad (found %d players, expected 15)", len(projections))
	}

	optimal := a.optimizer.OptimizeSquad(projections)
	gain := optimal.TotalProjectedXP - currentPicksXP
	if gain < 0 {
		gain = 0
	}

	var recs []string
	recs = append(recs, fmt.Sprintf("Recommended formation: %s for optimal expected points.", optimal.Formation))
	captainXP := 0.0
	for _, p := range optimal.StartingXI {
		if p.PlayerID == optimal.CaptainID {
			captainXP = p.ProjectedXP
			break
		}
	}
	recs = append(recs, fmt.Sprintf("Captain %s (Projected: %.1f pts) and Vice-Captain %s.", optimal.CaptainName, captainXP, optimal.ViceCaptainName))
	if len(optimal.Bench) > 1 {
		recs = append(recs, fmt.Sprintf("Priority 1 Bench Substitute: %s (xP: %.1f).", optimal.Bench[1].WebName, optimal.Bench[1].ProjectedXP))
	}

	if err := a.db.SaveProjectionSnapshot(ctx, input.TeamID, targetGW, optimal.Formation, optimal.CaptainName, optimal.ViceCaptainName, optimal.TotalProjectedXP); err != nil {
		log.Printf("suggest_lineup: persist snapshot: %v", err)
	}

	return &SuggestLineupOutput{
		TeamID:           input.TeamID,
		Gameweek:         targetGW,
		OptimalLineup:    optimal,
		CurrentLineupXP:  currentPicksXP,
		OptimizationGain: gain,
		CaptainDelta:     fmt.Sprintf("Recommended Captain: %s", optimal.CaptainName),
		Recommendations:  recs,
	}, nil
}

func (a *SuggestLineupAction) Handle(c *fiber.Ctx) error {
	var input SuggestLineupInput
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
