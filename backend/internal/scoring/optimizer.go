package scoring

import (
	"math"
	"sort"
)

// LineupSelection represents the optimized starting XI, bench, and captaincy
type LineupSelection struct {
	Formation        string             `json:"formation"` // e.g. "3-5-2", "4-3-3"
	TotalProjectedXP float64            `json:"total_projected_xp"`
	CaptainID        int                `json:"captain_id"`
	CaptainName      string             `json:"captain_name"`
	ViceCaptainID    int                `json:"vice_captain_id"`
	ViceCaptainName  string             `json:"vice_captain_name"`
	StartingXI       []PlayerProjection `json:"starting_xi"`
	Bench            []PlayerProjection `json:"bench"`
	Goalkeepers      []PlayerProjection `json:"goalkeepers"`
	Defenders        []PlayerProjection `json:"defenders"`
	Midfielders      []PlayerProjection `json:"midfielders"`
	Forwards         []PlayerProjection `json:"forwards"`
}

type Optimizer interface {
	OptimizeSquad(projections []PlayerProjection) *LineupSelection
}

type SquadOptimizer struct{}

func NewSquadOptimizer() *SquadOptimizer {
	return &SquadOptimizer{}
}

// Legal formations in FPL: (DEF, MID, FWD)
var legalFormations = [][]int{
	{3, 5, 2},
	{3, 4, 3},
	{4, 4, 2},
	{4, 3, 3},
	{4, 5, 1},
	{5, 3, 2},
	{5, 4, 1},
	{5, 2, 3},
}

// OptimizeSquad finds the optimal starting XI and bench order
func (o *SquadOptimizer) OptimizeSquad(projections []PlayerProjection) *LineupSelection {
	var gks, defs, mids, fwds []PlayerProjection

	for _, p := range projections {
		switch p.ElementType {
		case 1:
			gks = append(gks, p)
		case 2:
			defs = append(defs, p)
		case 3:
			mids = append(mids, p)
		case 4:
			fwds = append(fwds, p)
		}
	}

	// Sort each position by projected xP descending
	sortByXP := func(list []PlayerProjection) {
		sort.Slice(list, func(i, j int) bool {
			return list[i].ProjectedXP > list[j].ProjectedXP
		})
	}
	sortByXP(gks)
	sortByXP(defs)
	sortByXP(mids)
	sortByXP(fwds)

	bestFormation := "3-4-3"
	bestTotalXP := -1.0
	var bestXI []PlayerProjection
	var bestBench []PlayerProjection

	// 1 starting GK
	startingGK := gks[0]
	benchGK := gks[1]

	for _, form := range legalFormations {
		reqDef, reqMid, reqFwd := form[0], form[1], form[2]
		if len(defs) < reqDef || len(mids) < reqMid || len(fwds) < reqFwd {
			continue
		}

		currentXI := []PlayerProjection{startingGK}
		currentXI = append(currentXI, defs[:reqDef]...)
		currentXI = append(currentXI, mids[:reqMid]...)
		currentXI = append(currentXI, fwds[:reqFwd]...)

		currentBench := []PlayerProjection{benchGK}
		currentBench = append(currentBench, defs[reqDef:]...)
		currentBench = append(currentBench, mids[reqMid:]...)
		currentBench = append(currentBench, fwds[reqFwd:]...)

		// Sort outfield bench by xP descending
		sort.Slice(currentBench[1:], func(i, j int) bool {
			return currentBench[1:][i].ProjectedXP > currentBench[1:][j].ProjectedXP
		})

		// Calculate total XP (including 2x for highest player as captain)
		totalXP := 0.0
		highestXP := 0.0
		for _, p := range currentXI {
			totalXP += p.ProjectedXP
			if p.ProjectedXP > highestXP {
				highestXP = p.ProjectedXP
			}
		}
		// Add captain bonus
		totalWithCaptain := totalXP + highestXP

		if totalWithCaptain > bestTotalXP {
			bestTotalXP = totalWithCaptain
			bestFormation = formatFormation(form)
			bestXI = currentXI
			bestBench = currentBench
		}
	}

	// Find Captain (highest xP in XI) and Vice-Captain (second highest)
	sortedXI := make([]PlayerProjection, len(bestXI))
	copy(sortedXI, bestXI)
	sort.Slice(sortedXI, func(i, j int) bool {
		return sortedXI[i].ProjectedXP > sortedXI[j].ProjectedXP
	})

	captain := sortedXI[0]
	viceCaptain := sortedXI[1]

	// Categorize Starting XI by position for UI presentation
	var xiGK, xiDef, xiMid, xiFwd []PlayerProjection
	for _, p := range bestXI {
		switch p.ElementType {
		case 1:
			xiGK = append(xiGK, p)
		case 2:
			xiDef = append(xiDef, p)
		case 3:
			xiMid = append(xiMid, p)
		case 4:
			xiFwd = append(xiFwd, p)
		}
	}

	return &LineupSelection{
		Formation:        bestFormation,
		TotalProjectedXP: math.Round(bestTotalXP*10) / 10,
		CaptainID:        captain.PlayerID,
		CaptainName:      captain.WebName,
		ViceCaptainID:    viceCaptain.PlayerID,
		ViceCaptainName:  viceCaptain.WebName,
		StartingXI:       bestXI,
		Bench:            bestBench,
		Goalkeepers:      xiGK,
		Defenders:        xiDef,
		Midfielders:      xiMid,
		Forwards:         xiFwd,
	}
}

func formatFormation(f []int) string {
	return string([]byte{byte('0' + f[0]), '-', byte('0' + f[1]), '-', byte('0' + f[2])})
}
