package scoring

import (
	"testing"
)

func TestOptimizeSquad_ValidFormationAndCaptain(t *testing.T) {
	optimizer := NewSquadOptimizer()

	// Create a mock 15-player squad
	projections := []PlayerProjection{
		// 2 GKPs
		{PlayerID: 1, WebName: "Raya", ElementType: 1, ProjectedXP: 5.5},
		{PlayerID: 2, WebName: "Turner", ElementType: 1, ProjectedXP: 1.5},
		// 5 DEFs
		{PlayerID: 3, WebName: "Gabriel", ElementType: 2, ProjectedXP: 6.0},
		{PlayerID: 4, WebName: "Saliba", ElementType: 2, ProjectedXP: 5.8},
		{PlayerID: 5, WebName: "Gvardiol", ElementType: 2, ProjectedXP: 5.2},
		{PlayerID: 6, WebName: "Konsa", ElementType: 2, ProjectedXP: 3.5},
		{PlayerID: 7, WebName: "Barco", ElementType: 2, ProjectedXP: 2.0},
		// 5 MIDs
		{PlayerID: 8, WebName: "Salah", ElementType: 3, ProjectedXP: 8.5},
		{PlayerID: 9, WebName: "Saka", ElementType: 3, ProjectedXP: 7.2},
		{PlayerID: 10, WebName: "Palmer", ElementType: 3, ProjectedXP: 8.0},
		{PlayerID: 11, WebName: "Rogers", ElementType: 3, ProjectedXP: 4.8},
		{PlayerID: 12, WebName: "Winks", ElementType: 3, ProjectedXP: 2.5},
		// 3 FWDs
		{PlayerID: 13, WebName: "Haaland", ElementType: 4, ProjectedXP: 9.0},
		{PlayerID: 14, WebName: "Watkins", ElementType: 4, ProjectedXP: 6.5},
		{PlayerID: 15, WebName: "Muniz", ElementType: 4, ProjectedXP: 4.2},
	}

	result := optimizer.OptimizeSquad(projections)

	if len(result.StartingXI) != 11 {
		t.Fatalf("expected 11 starting players, got %d", len(result.StartingXI))
	}
	if len(result.Bench) != 4 {
		t.Fatalf("expected 4 bench players, got %d", len(result.Bench))
	}
	if result.CaptainName != "Haaland" {
		t.Fatalf("expected Haaland as Captain (highest xP 9.0), got %s", result.CaptainName)
	}
	if result.ViceCaptainName != "Salah" {
		t.Fatalf("expected Salah as Vice-Captain (second highest xP 8.5), got %s", result.ViceCaptainName)
	}
	if result.TotalProjectedXP <= 0 {
		t.Fatalf("expected positive total xP, got %.2f", result.TotalProjectedXP)
	}
}
