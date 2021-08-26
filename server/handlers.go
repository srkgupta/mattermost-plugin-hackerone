package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
)

func (p *Plugin) executeStats(args *model.CommandArgs, split []string) (*model.CommandResponse, *model.AppError) {
	stats, err := p.fetchStats()
	if err != nil {
		msg := fmt.Sprintf("Something went wrong while getting the stats from Hackerone API. Error: %s\n", err.Error())
		return p.sendEphemeralResponse(args, msg), nil
	} else {
		statsListString := "| New | Triaged | Pending Bounty |\n| :----- | :----- | :----- | \n"
		statsListString += fmt.Sprintf("| %d | %d | %d |\n", stats.NewCount, stats.TriagedCount, stats.PendingBountyCount)
		_ = p.sendPost(args, statsListString, nil)

	}
	return &model.CommandResponse{}, nil
}
func (p *Plugin) contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

func (p *Plugin) parseTime(input string) string {
	if len(input) > 5 {
		layout := "Mon Jan 02 2006 3:04 PM"
		t, _ := time.Parse(time.RFC3339, input)
		return t.Format(layout)
	} else {
		return "-"
	}
}
