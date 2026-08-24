package fpl

import "time"

// BootstrapStatic represents the root payload from /api/bootstrap-static/
type BootstrapStatic struct {
	Events       []Event       `json:"events"`
	Teams        []Team        `json:"teams"`
	Elements     []Element     `json:"elements"`
	ElementTypes []ElementType `json:"element_types"`
}

// Event represents a gameweek schedule & state
type Event struct {
	ID                     int        `json:"id"`
	Name                   string     `json:"name"`
	DeadlineTime           time.Time  `json:"deadline_time"`
	AverageEntryScore      int        `json:"average_entry_score"`
	Finished               bool       `json:"finished"`
	DataChecked            bool       `json:"data_checked"`
	HighestScoringEntry    *int       `json:"highest_scoring_entry"`
	DeadlineTimeEpoch      int64      `json:"deadline_time_epoch"`
	DeadlineTimeGameOffset int        `json:"deadline_time_game_offset"`
	HighestScore           *int       `json:"highest_score"`
	IsPrevious             bool       `json:"is_previous"`
	IsCurrent              bool       `json:"is_current"`
	IsNext                 bool       `json:"is_next"`
	CupLeaguesCreated      bool       `json:"cup_leagues_created"`
	H2HkoMatchesCreated    bool       `json:"h2h_ko_matches_created"`
	ChipPlays              []ChipPlay `json:"chip_plays"`
}

type ChipPlay struct {
	ChipName  string `json:"chip_name"`
	NumPlayed int    `json:"num_played"`
}

// Team represents a Premier League club
type Team struct {
	ID                  int    `json:"id"`
	Code                int    `json:"code"`
	Name                string `json:"name"`
	ShortName           string `json:"short_name"`
	Strength            int    `json:"strength"`
	StrengthOverallHome int    `json:"strength_overall_home"`
	StrengthOverallAway int    `json:"strength_overall_away"`
	StrengthAttackHome  int    `json:"strength_attack_home"`
	StrengthAttackAway  int    `json:"strength_attack_away"`
	StrengthDefenceHome int    `json:"strength_defence_home"`
	StrengthDefenceAway int    `json:"strength_defence_away"`
}

// ElementType represents positions: 1=GKP, 2=DEF, 3=MID, 4=FWD
type ElementType struct {
	ID                int    `json:"id"`
	PluralName        string `json:"plural_name"`
	PluralNameShort   string `json:"plural_name_short"`
	SingularName      string `json:"singular_name"`
	SingularNameShort string `json:"singular_name_short"`
	SquadSelect       int    `json:"squad_select"`
	SquadMinPlay      int    `json:"squad_min_play"`
	SquadMaxPlay      int    `json:"squad_max_play"`
}

// Element represents an individual player
type Element struct {
	ID                       int     `json:"id"`
	Code                     int     `json:"code"`
	FirstName                string  `json:"first_name"`
	SecondName               string  `json:"second_name"`
	WebName                  string  `json:"web_name"`
	Team                     int     `json:"team"`
	ElementType              int     `json:"element_type"` // 1=GKP, 2=DEF, 3=MID, 4=FWD
	Status                   string  `json:"status"`       // "a"=available, "d"=doubtful, "i"=injured, "s"=suspended, "u"=unavailable
	NowCost                  int     `json:"now_cost"`     // in tenths (e.g. 125 = £12.5m)
	ChanceOfPlayingNextRound *int    `json:"chance_of_playing_next_round"`
	ChanceOfPlayingThisRound *int    `json:"chance_of_playing_this_round"`
	News                     string  `json:"news"`
	NewsAdded                *string `json:"news_added"`
	Form                     string  `json:"form"`
	PointsPerGame            string  `json:"points_per_game"`
	TotalPoints              int     `json:"total_points"`
	Minutes                  int     `json:"minutes"`
	GoalsScored              int     `json:"goals_scored"`
	Assists                  int     `json:"assists"`
	CleanSheets              int     `json:"clean_sheets"`
	GoalsConceded            int     `json:"goals_conceded"`
	OwnGoals                 int     `json:"own_goals"`
	PenaltiesSaved           int     `json:"penalties_saved"`
	PenaltiesMissed          int     `json:"penalties_missed"`
	YellowCards              int     `json:"yellow_cards"`
	RedCards                 int     `json:"red_cards"`
	Saves                    int     `json:"saves"`
	Bonus                    int     `json:"bonus"`
	BPS                      int     `json:"bps"`
	Influence                string  `json:"influence"`
	Creativity               string  `json:"creativity"`
	Threat                   string  `json:"threat"`
	ICTIndex                 string  `json:"ict_index"`
	ExpectedGoals            string  `json:"expected_goals"`
	ExpectedAssists          string  `json:"expected_assists"`
	ExpectedGoalInvolvements string  `json:"expected_goal_involvements"`
	ExpectedGoalsConceded    string  `json:"expected_goals_conceded"`
	SelectedByPercent        string  `json:"selected_by_percent"`
	EpNext                   *string `json:"ep_next"`
	EpThis                   *string `json:"ep_this"`
}

// Entry represents a user's manager profile from /api/entry/{team_id}/
type Entry struct {
	ID                         int          `json:"id"`
	PlayerFirstName            string       `json:"player_first_name"`
	PlayerLastName             string       `json:"player_last_name"`
	PlayerRegionName           string       `json:"player_region_name"`
	SummaryOverallPoints       int          `json:"summary_overall_points"`
	SummaryOverallRank         int          `json:"summary_overall_rank"`
	SummaryEventPoints         int          `json:"summary_event_points"`
	SummaryEventRank           int          `json:"summary_event_rank"`
	CurrentEvent               int          `json:"current_event"`
	Name                       string       `json:"name"`
	LastDeadlineBank           int          `json:"last_deadline_bank"`
	LastDeadlineValue          int          `json:"last_deadline_value"`
	LastDeadlineTotalTransfers int          `json:"last_deadline_total_transfers"`
	Leagues                    EntryLeagues `json:"leagues"`
}

type EntryLeagues struct {
	Classic []ClassicLeague `json:"classic"`
	H2H     []H2HLeague     `json:"h2h"`
}

type ClassicLeague struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	EntryRank     int    `json:"entry_rank"`
	EntryLastRank int    `json:"entry_last_rank"`
}

type H2HLeague struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	EntryRank     int    `json:"entry_rank"`
	EntryLastRank int    `json:"entry_last_rank"`
}

// PicksResponse represents the payload from /api/entry/{team_id}/event/{gw}/picks/
type PicksResponse struct {
	ActiveChip   *string      `json:"active_chip"`
	EntryHistory EntryHistory `json:"entry_history"`
	Picks        []Pick       `json:"picks"`
}

type EntryHistory struct {
	Event              int `json:"event"`
	Points             int `json:"points"`
	TotalPoints        int `json:"total_points"`
	Rank               int `json:"rank"`
	RankSort           int `json:"rank_sort"`
	OverallRank        int `json:"overall_rank"`
	Bank               int `json:"bank"`
	Value              int `json:"value"`
	EventTransfers     int `json:"event_transfers"`
	EventTransfersCost int `json:"event_transfers_cost"`
	PointsOnBench      int `json:"points_on_bench"`
}

// EntryHistoryResponse is the payload from /api/entry/{id}/history/
type EntryHistoryResponse struct {
	Current []GWHistoryEntry `json:"current"`
}

// GWHistoryEntry holds the per-gameweek result for a manager
type GWHistoryEntry struct {
	Event              int `json:"event"`
	Points             int `json:"points"`
	TotalPoints        int `json:"total_points"`
	Rank               int `json:"rank"`
	OverallRank        int `json:"overall_rank"`
	Bank               int `json:"bank"`
	Value              int `json:"value"`
	EventTransfers     int `json:"event_transfers"`
	EventTransfersCost int `json:"event_transfers_cost"`
	PointsOnBench      int `json:"points_on_bench"`
}

type Pick struct {
	Element       int  `json:"element"`    // Player ID
	Position      int  `json:"position"`   // 1-15
	Multiplier    int  `json:"multiplier"` // 0=benched, 1=playing, 2=captain, 3=triple captain
	IsCaptain     bool `json:"is_captain"`
	IsViceCaptain bool `json:"is_vice_captain"`
}

// Fixture represents a match from /api/fixtures/
type Fixture struct {
	ID                   int        `json:"id"`
	Code                 int        `json:"code"`
	Event                *int       `json:"event"`
	Finished             bool       `json:"finished"`
	FinishedProvisional  bool       `json:"finished_provisional"`
	KickoffTime          *time.Time `json:"kickoff_time"`
	Minutes              int        `json:"minutes"`
	ProvisionalStartTime bool       `json:"provisional_start_time"`
	Started              bool       `json:"started"`
	TeamA                int        `json:"team_a"`
	TeamH                int        `json:"team_h"`
	TeamADifficulty      int        `json:"team_a_difficulty"`
	TeamHDifficulty      int        `json:"team_h_difficulty"`
	TeamAScore           *int       `json:"team_a_score"`
	TeamHScore           *int       `json:"team_h_score"`
}
