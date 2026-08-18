package actions

import "fpl-assistant/internal/fpl"

var positionNames = map[int]string{1: "GKP", 2: "DEF", 3: "MID", 4: "FWD"}

func resolveActiveGameweek(events []fpl.Event) int {
	active := 1
	for _, ev := range events {
		if ev.IsCurrent {
			return ev.ID
		}
		if ev.IsNext {
			active = ev.ID
		}
	}
	return active
}

func resolveGameweek(events []fpl.Event, override *int) int {
	if override != nil && *override > 0 {
		return *override
	}
	return resolveActiveGameweek(events)
}

func buildElementMap(elements []fpl.Element) map[int]fpl.Element {
	m := make(map[int]fpl.Element, len(elements))
	for _, el := range elements {
		m[el.ID] = el
	}
	return m
}

func buildTeamMap(teams []fpl.Team) map[int]fpl.Team {
	m := make(map[int]fpl.Team, len(teams))
	for _, tm := range teams {
		m[tm.ID] = tm
	}
	return m
}

func availabilityFromElement(el fpl.Element) (chance int, badge string) {
	chance = 100
	badge = "success"
	if el.ChanceOfPlayingNextRound != nil {
		chance = *el.ChanceOfPlayingNextRound
		switch chance {
		case 100:
			badge = "success"
		case 75:
			badge = "warning"
		case 50, 25:
			badge = "caution"
		case 0:
			badge = "danger"
		}
		return chance, badge
	}
	if el.Status == "i" || el.Status == "s" || el.Status == "u" {
		return 0, "danger"
	}
	if el.Status == "d" {
		return 50, "caution"
	}
	return chance, badge
}
