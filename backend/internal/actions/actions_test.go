package actions

import (
	"context"
	"fpl-assistant/internal/fpl"
	"fpl-assistant/internal/scoring"
	"testing"
)

type MockFPLClient struct {
	Entry     *fpl.Entry
	Bootstrap *fpl.BootstrapStatic
	Picks     *fpl.PicksResponse
	Fixtures  []fpl.Fixture
}

func (m *MockFPLClient) GetBootstrapStatic(ctx context.Context) (*fpl.BootstrapStatic, error) {
	return m.Bootstrap, nil
}

func (m *MockFPLClient) GetEntry(ctx context.Context, teamID int) (*fpl.Entry, error) {
	return m.Entry, nil
}

func (m *MockFPLClient) GetPicks(ctx context.Context, teamID int, event int) (*fpl.PicksResponse, error) {
	return m.Picks, nil
}

func (m *MockFPLClient) GetFixtures(ctx context.Context, event *int) ([]fpl.Fixture, error) {
	return m.Fixtures, nil
}

func createTestFixtureClient() *MockFPLClient {
	elements := make([]fpl.Element, 15)
	picks := make([]fpl.Pick, 15)
	posList := []int{1, 1, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 4, 4, 4}

	for i := 0; i < 15; i++ {
		elements[i] = fpl.Element{
			ID:          i + 1,
			WebName:     "Player" + string(rune('A'+i)),
			Team:        1,
			ElementType: posList[i],
			NowCost:     60,
			Status:      "a",
			Form:        "5.0",
		}
		picks[i] = fpl.Pick{
			Element:    i + 1,
			Position:   i + 1,
			Multiplier: 1,
		}
	}
	picks[12].IsCaptain = true
	picks[12].Multiplier = 2

	return &MockFPLClient{
		Entry: &fpl.Entry{
			ID:                   12345,
			PlayerFirstName:      "John",
			PlayerLastName:       "Doe",
			Name:                 "All Stars FC",
			SummaryOverallRank:   5420,
			SummaryOverallPoints: 1250,
			LastDeadlineBank:     15,
			LastDeadlineValue:    1025,
		},
		Bootstrap: &fpl.BootstrapStatic{
			Events: []fpl.Event{
				{ID: 1, IsCurrent: true},
			},
			Teams: []fpl.Team{
				{ID: 1, ShortName: "ARS"},
			},
			Elements: elements,
		},
		Picks: &fpl.PicksResponse{
			Picks: picks,
		},
		Fixtures: []fpl.Fixture{},
	}
}

func TestConnectTeamAction(t *testing.T) {
	mockClient := createTestFixtureClient()
	action := NewConnectTeamAction(mockClient, nil)

	out, err := action.Execute(context.Background(), ConnectTeamInput{TeamID: 12345})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.TeamName != "All Stars FC" {
		t.Fatalf("expected team name 'All Stars FC', got %s", out.TeamName)
	}
	if out.ManagerName != "John Doe" {
		t.Fatalf("expected manager name 'John Doe', got %s", out.ManagerName)
	}
	if out.Bank != 1.5 {
		t.Fatalf("expected bank 1.5, got %.1f", out.Bank)
	}
}

func TestGetAvailabilityNewsAction(t *testing.T) {
	mockClient := createTestFixtureClient()
	// Add an injury
	mockClient.Bootstrap.Elements[0].Status = "i"
	mockClient.Bootstrap.Elements[0].News = "Knee strain"

	action := NewGetAvailabilityNewsAction(mockClient, nil)
	teamID := 12345
	out, err := action.Execute(context.Background(), GetAvailabilityNewsInput{TeamID: &teamID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.InjuredCount != 1 {
		t.Fatalf("expected 1 injured player, got %d", out.InjuredCount)
	}
	if len(out.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(out.Alerts))
	}
}

func TestSuggestLineupAction(t *testing.T) {
	mockClient := createTestFixtureClient()
	engine := scoring.NewXPEngine()
	optimizer := scoring.NewSquadOptimizer()
	action := NewSuggestLineupAction(mockClient, engine, optimizer, nil)

	out, err := action.Execute(context.Background(), SuggestLineupInput{TeamID: 12345})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.OptimalLineup == nil {
		t.Fatalf("expected non-nil optimal lineup")
	}
	if len(out.OptimalLineup.StartingXI) != 11 {
		t.Fatalf("expected 11 starting players, got %d", len(out.OptimalLineup.StartingXI))
	}
}
