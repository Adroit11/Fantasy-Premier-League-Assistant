package ai

import (
	"fmt"
	"strings"

	"fpl-assistant/internal/scoring"
)

// BuildSelectionPrompt constructs the structured prompt sent to the AI for
// weekly team selection. It embeds xP projections, injury news, and fixture
// context as JSON so the model has precise numerical grounding.
func BuildSelectionPrompt(
	gameweek int,
	managerName string,
	projections []scoring.PlayerProjection,
	injuryAlerts []string,
	bankMillions float64,
	freeTransfers int,
) string {
	var sb strings.Builder

	sb.WriteString("You are an expert Fantasy Premier League (FPL) assistant for the 2026/2027 season.\n")
	sb.WriteString("Your task: analyse the squad data below and provide a gameweek team selection recommendation.\n\n")

	sb.WriteString(fmt.Sprintf("## Gameweek %d — %s's Squad\n\n", gameweek, managerName))
	sb.WriteString(fmt.Sprintf("Bank available: £%.1fm | Free transfers: %d\n\n", bankMillions, freeTransfers))

	// Injury / availability alerts
	if len(injuryAlerts) > 0 {
		sb.WriteString("### ⚠️ Injury & Availability Alerts\n")
		for _, alert := range injuryAlerts {
			sb.WriteString(fmt.Sprintf("- %s\n", alert))
		}
		sb.WriteString("\n")
	}

	// xP projections table
	sb.WriteString("### Squad xP Projections (sorted by projected xP descending)\n")
	sb.WriteString("| # | Player | Pos | Team | vs | H/A | FDR | xP | Avail% | Status |\n")
	sb.WriteString("|---|--------|-----|------|----|-----|-----|-----|--------|--------|\n")

	posNames := map[int]string{1: "GKP", 2: "DEF", 3: "MID", 4: "FWD"}
	for i, p := range projections {
		ha := "A"
		if p.IsHome {
			ha = "H"
		}
		sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s | %d | %.1f | %d%% | %s |\n",
			i+1,
			p.WebName,
			posNames[p.ElementType],
			p.TeamShortName,
			p.OpponentShortName,
			ha,
			p.FDR,
			p.ProjectedXP,
			p.AvailabilityChance,
			p.StatusBadge,
		))
	}
	sb.WriteString("\n")

	sb.WriteString("### Instructions\n")
	sb.WriteString("Based on the xP projections and availability data above, provide:\n\n")
	sb.WriteString("1. **Recommended Starting XI** (11 players, valid formation: 1 GKP, 3-5 DEF, 2-5 MID, 1-3 FWD)\n")
	sb.WriteString("2. **Captain & Vice-Captain** with a clear reason referencing the xP data and fixture\n")
	sb.WriteString("3. **Bench Order** (4 remaining players, priority order)\n")
	sb.WriteString("4. **Key Risks** — flag any doubtful players in the XI\n")
	sb.WriteString("5. **One-line Summary** — a punchy headline for this gameweek's strategy\n\n")
	sb.WriteString("Be concise. Use the xP numbers explicitly. Format with markdown headings.\n")

	return sb.String()
}

// BuildImprovementPrompt constructs the prompt for the post-GW improvement suggestion.
// It compares AI's pre-GW recommendation against what the user actually played.
func BuildImprovementPrompt(
	gameweek int,
	managerName string,
	aiSuggestion string,
	actualPoints int,
	aiProjectedXP float64,
	transferRecommendations string,
	nextGW int,
) string {
	var sb strings.Builder

	sb.WriteString("You are an expert Fantasy Premier League (FPL) coach for the 2026/2027 season.\n\n")
	sb.WriteString(fmt.Sprintf("## Post-GW %d Analysis for %s\n\n", gameweek, managerName))

	sb.WriteString(fmt.Sprintf("- **Actual GW points scored**: %d\n", actualPoints))
	sb.WriteString(fmt.Sprintf("- **AI projected xP (pre-GW)**: %.1f\n", aiProjectedXP))
	delta := float64(actualPoints) - aiProjectedXP
	direction := "above"
	if delta < 0 {
		direction = "below"
	}
	sb.WriteString(fmt.Sprintf("- **Performance vs projection**: %.1f pts %s projection\n\n", absF(delta), direction))

	sb.WriteString("### AI Pre-GW Recommendation (what was suggested before the GW):\n")
	sb.WriteString(aiSuggestion)
	sb.WriteString("\n\n")

	if transferRecommendations != "" {
		sb.WriteString("### Current Transfer Recommendations for GW ")
		sb.WriteString(fmt.Sprintf("%d:\n", nextGW))
		sb.WriteString(transferRecommendations)
		sb.WriteString("\n\n")
	}

	sb.WriteString("### Your Task\n")
	sb.WriteString("Provide a concise improvement plan for the manager:\n\n")
	sb.WriteString("1. **What went well / what went wrong** — reference specific players or the captain choice\n")
	sb.WriteString("2. **Priority transfers for GW ")
	sb.WriteString(fmt.Sprintf("%d** — max 2 transfers, justify with xP gain\n", nextGW))
	sb.WriteString("3. **Chip advice** — should they play Wildcard, Triple Captain, Bench Boost, or Free Hit this GW?\n")
	sb.WriteString("4. **Motivation** — end with a short motivational line\n\n")
	sb.WriteString("Be specific, data-driven, and concise. Format with markdown.\n")

	return sb.String()
}

func absF(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
