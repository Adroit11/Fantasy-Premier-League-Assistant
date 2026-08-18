package actions

import (
	"context"
	"fmt"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
	"github.com/gofiber/fiber/v2"
	"math"
	"sort"
)

type SuggestTransfersInput struct {
	TeamID       int  `json:"team_id"`
	MaxTransfers *int `json:"max_transfers,omitempty"`
	Gameweek     *int `json:"gameweek,omitempty"`
}

type TransferRecommendation struct {
	PlayerOutID    int     `json:"player_out_id"`
	PlayerOutName  string  `json:"player_out_name"`
	PlayerOutTeam  string  `json:"player_out_team"`
	PlayerOutCost  float64 `json:"player_out_cost"`
	PlayerOutXP    float64 `json:"player_out_xp"`
	PlayerOutNews  string  `json:"player_out_news"`
	PlayerInID     int     `json:"player_in_id"`
	PlayerInName   string  `json:"player_in_name"`
	PlayerInTeam   string  `json:"player_in_team"`
	PlayerInCost   float64 `json:"player_in_cost"`
	PlayerInXP     float64 `json:"player_in_xp"`
	NetXPGain      float64 `json:"net_xp_gain"`
	CostDifference float64 `json:"cost_difference"`
	PositionName   string  `json:"position_name"`
	Reason         string  `json:"reason"`
}

type SuggestTransfersOutput struct {
	TeamID          int                      `json:"team_id"`
	Gameweek        int                      `json:"gameweek"`
	BankAvailable   float64                  `json:"bank_available"`
	Recommendations []TransferRecommendation `json:"recommendations"`
	Message         string                   `json:"message"`
}

type SuggestTransfersAction struct {
	client fpl.Client
	engine scoring.Engine
}

func NewSuggestTransfersAction(client fpl.Client, engine scoring.Engine) *SuggestTransfersAction {
	return &SuggestTransfersAction{
		client: client,
		engine: engine,
	}
}

func (a *SuggestTransfersAction) Execute(ctx context.Context, input SuggestTransfersInput) (*SuggestTransfersOutput, error) {
	if input.TeamID <= 0 {
		return nil, fmt.Errorf("invalid team_id: must be greater than zero")
	}

	entry, err := a.client.GetEntry(ctx, input.TeamID)
	if err != nil {
		return nil, fmt.Errorf("suggest_transfers: failed to fetch entry: %w", err)
	}

	bootstrap, err := a.client.GetBootstrapStatic(ctx)
	if err != nil {
		return nil, fmt.Errorf("suggest_transfers: failed to fetch bootstrap: %w", err)
	}

	fixtures, err := a.client.GetFixtures(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("suggest_transfers: failed to fetch fixtures: %w", err)
	}

	targetGW := resolveGameweek(bootstrap.Events, input.Gameweek)

	picksResp, err := a.client.GetPicks(ctx, input.TeamID, targetGW)
	if err != nil {
		return nil, fmt.Errorf("suggest_transfers: failed to fetch squad picks for GW %d: %w", targetGW, err)
	}

	elementMap := buildElementMap(bootstrap.Elements)
	teamMap := buildTeamMap(bootstrap.Teams)
	fixIndex := scoring.NewFixtureIndex(fixtures)

	bankTenths := entry.LastDeadlineBank
	bankMillions := float64(bankTenths) / 10.0

	squadIDs := make(map[int]bool)
	var squadProjections []scoring.PlayerProjection
	for _, p := range picksResp.Picks {
		squadIDs[p.Element] = true
		if el, ok := elementMap[p.Element]; ok {
			proj := a.engine.CalculatePlayerXPIndexed(&el, teamMap, fixIndex, targetGW)
			squadProjections = append(squadProjections, proj)
		}
	}

	candidatesByPos := make(map[int][]scoring.PlayerProjection)
	for _, el := range bootstrap.Elements {
		if squadIDs[el.ID] {
			continue
		}
		if el.Status == "i" || el.Status == "u" || el.Status == "s" {
			continue
		}
		proj := a.engine.CalculatePlayerXPIndexed(&el, teamMap, fixIndex, targetGW)
		if proj.ProjectedXP > 3.5 {
			candidatesByPos[el.ElementType] = append(candidatesByPos[el.ElementType], proj)
		}
	}

	const maxCandidatesPerPos = 25
	for pos := range candidatesByPos {
		sort.Slice(candidatesByPos[pos], func(i, j int) bool {
			return candidatesByPos[pos][i].ProjectedXP > candidatesByPos[pos][j].ProjectedXP
		})
		if len(candidatesByPos[pos]) > maxCandidatesPerPos {
			candidatesByPos[pos] = candidatesByPos[pos][:maxCandidatesPerPos]
		}
	}

	var recs []TransferRecommendation

	for _, outProj := range squadProjections {
		outEl := elementMap[outProj.PlayerID]
		maxAffordableTenths := bankTenths + outEl.NowCost

		candidates := candidatesByPos[outProj.ElementType]
		for _, inProj := range candidates {
			inEl := elementMap[inProj.PlayerID]
			if inEl.NowCost <= maxAffordableTenths {
				xpGain := inProj.ProjectedXP - outProj.ProjectedXP
				if xpGain >= 1.2 || (outEl.Status != "a" && xpGain > 0) {
					reason := fmt.Sprintf("+%.1f projected xP upgrade over upcoming fixture against %s", xpGain, inProj.OpponentShortName)
					if outEl.Status != "a" {
						reason = fmt.Sprintf("Replace injured/doubtful player (%s) with fit starter", outEl.News)
					}

					recs = append(recs, TransferRecommendation{
						PlayerOutID:    outProj.PlayerID,
						PlayerOutName:  outProj.WebName,
						PlayerOutTeam:  outProj.TeamShortName,
						PlayerOutCost:  float64(outEl.NowCost) / 10.0,
						PlayerOutXP:    outProj.ProjectedXP,
						PlayerOutNews:  outEl.News,
						PlayerInID:     inProj.PlayerID,
						PlayerInName:   inProj.WebName,
						PlayerInTeam:   inProj.TeamShortName,
						PlayerInCost:   float64(inEl.NowCost) / 10.0,
						PlayerInXP:     inProj.ProjectedXP,
						NetXPGain:      math.Round(xpGain*10) / 10,
						CostDifference: float64(inEl.NowCost-outEl.NowCost) / 10.0,
						PositionName:   positionNames[outProj.ElementType],
						Reason:         reason,
					})
					break
				}
			}
		}
	}

	sort.Slice(recs, func(i, j int) bool {
		return recs[i].NetXPGain > recs[j].NetXPGain
	})

	maxRecs := 5
	if input.MaxTransfers != nil && *input.MaxTransfers > 0 {
		maxRecs = *input.MaxTransfers
	}
	if len(recs) > maxRecs {
		recs = recs[:maxRecs]
	}

	return &SuggestTransfersOutput{
		TeamID:          input.TeamID,
		Gameweek:        targetGW,
		BankAvailable:   bankMillions,
		Recommendations: recs,
		Message:         fmt.Sprintf("Found %d high-value transfer recommendations", len(recs)),
	}, nil
}

func (a *SuggestTransfersAction) Handle(c *fiber.Ctx) error {
	var input SuggestTransfersInput
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
