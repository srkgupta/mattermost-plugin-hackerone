package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
)

func (p *Plugin) executeActivities(args *model.CommandArgs, split []string) (*model.CommandResponse, *model.AppError) {
	count := "10"
	if 0 < len(split) {
		s := split[0]
		i, err := strconv.Atoi(s)
		if err != nil || i > 100 || i < 1 {
			count = "10"
		} else {
			count = s
		}
	}
	activitiesListString := ""
	activities, err := p.fetchActivities(count)
	if err != nil {
		msg := fmt.Sprintf("Something went wrong while getting the activities from Hackerone API. Error: %s\n", err.Error())
		return p.sendEphemeralResponse(args, msg), nil
	} else {
		for _, activity := range activities {
			reportLink := "[report " + activity.Attributes.ReportID + "](https://hackerone.com/reports/" + activity.Attributes.ReportID + ")"
			actorLink := "[" + activity.Relationships.Actor.Data.Attributes.Name + "](https://hackerone.com/" + activity.Relationships.Actor.Data.Attributes.Username + ")"
			activityLink := strings.Replace(getActivityType(activity.ActivityType), "report", reportLink, -1)
			activitiesListString += fmt.Sprintf(
				"> %s %s at %s\n",
				actorLink,
				activityLink,
				parseTime(activity.Attributes.CreatedAt),
			)
			if len(activity.Attributes.Message) > 1 {
				activitiesListString += "\n```\n" + activity.Attributes.Message + "\n```\n"
			}
		}
		_ = p.sendPost(args, activitiesListString, nil)
	}

	return &model.CommandResponse{}, nil
}

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

func (p *Plugin) executeReport(args *model.CommandArgs, split []string) (*model.CommandResponse, *model.AppError) {
	if 0 >= len(split) {
		msg := "Report Id should be specified while fetching the report information"
		return p.sendEphemeralResponse(args, msg), nil
	} else {
		reportId := split[0]
		report, err := p.fetchReport(reportId)
		if err != nil {
			msg := fmt.Sprintf("Something went wrong while getting the report from Hackerone API. Error: %s\n", err.Error())
			return p.sendEphemeralResponse(args, msg), nil
		} else {
			reportString := getReportString(report, true)
			_ = p.sendPost(args, reportString, nil)
		}
	}
	return &model.CommandResponse{}, nil
}

func (p *Plugin) executeReports(args *model.CommandArgs, split []string) (*model.CommandResponse, *model.AppError) {
	state := ""
	if len(split) > 0 {
		state = split[0]
		allowedStates := []string{"all", "new", "triaged", "needs-more-info", "bounty", "disclosure", "disclosed", "resolved"}
		if !contains(allowedStates, state) {
			msg := "Incorrect filter option applied. Please select a valid option from the autocomplete."
			return p.sendEphemeralResponse(args, msg), nil
		}
	} else {
		msg := "Filter not provided. Please select a valid option from the autocomplete."
		return p.sendEphemeralResponse(args, msg), nil
	}

	filters := make(map[string]string)
	hackeroneStates := []string{"new", "triaged", "needs-more-info", "resolved"}
	if contains(hackeroneStates, state) {
		filters["state"] = state
	} else if state == "bounty" {
		filters["state"] = "triaged"
		filters["bounty_awarded_at__null"] = "true"
	} else if state == "disclosure" {
		filters["reporter_agreed_on_going_public"] = "true"
	} else if state == "disclosed" {
		filters["disclosed_at__null"] = "false"
	}

	reports, err := p.fetchReports(filters)
	if err != nil {
		msg := fmt.Sprintf("Something went wrong while getting the reports from Hackerone API. Error: %s\n", err.Error())
		return p.sendEphemeralResponse(args, msg), nil
	} else {
		reportString := ""
		if len(reports) > 0 {
			for _, report := range reports {
				reportString += getReportString(report, false)
				reportString += "\n\n"
			}
			_ = p.sendPost(args, reportString, nil)
		} else {
			msg := "No reports found matching the filter criteria you have specified."
			return p.sendEphemeralResponse(args, msg), nil
		}
	}

	return &model.CommandResponse{}, nil
}

func getReportString(report Report, description bool) string {
	reportLink := fmt.Sprintf("[%s](https://hackerone.com/reports/%s)", report.Attributes.Title, report.Id)
	actorLink := "[" + report.Relationships.Reporter.Data.Attributes.Name + "](https://hackerone.com/" + report.Relationships.Reporter.Data.Attributes.Username + ")"
	reportString := fmt.Sprintf("### %s\n\n", reportLink)
	reportString += "| ID | Reporter | State | Created At | Triaged At | Bounty Awarded At | Closed At | Disclosed At |\n| :----- | :----- | :----- | :----- | :----- | :----- | :----- | :----- |\n"
	reportString += fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |\n", report.Id, actorLink, report.Attributes.State, report.Attributes.CreatedAt, report.Attributes.TriagedAt, report.Attributes.BountyAwardedAt, report.Attributes.ClosedAt, report.Attributes.DisclosedAt)
	if description && len(report.Attributes.Info) > 0 {
		reportString += fmt.Sprintf("#### Report Description:\n %s\n\n", report.Attributes.Info)
	}
	return reportString
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}

	return false
}

func parseTime(input string) string {
	layout := "Mon Jan 02 2006 3:04 PM"
	t, _ := time.Parse(time.RFC3339, input)
	return t.Format(layout)
}

func getActivityType(activityType string) string {
	switch activityType {
	case "activity-agreed-on-going-public":
		return "agreed on going public on the report"
	case "activity-bounty-awarded":
		return "awarded a bounty on the report"
	case "activity-comment":
		return "commented on the report"
	case "activity-bug-triaged":
		return "triaged the report"
	case "activity-bug-resolved":
		return "resolved the report"
	case "activity-bug-filed":
		return "filed a new report"
	case "activity-bug-informative":
		return "closed the report as Informative"
	case "activity-bug-needs-more-info":
		return "requested more info"
	case "activity-bug-not-applicable":
		return "closed the report as Not Applicable"
	case "activity-bug-reopened":
		return "reopened the report"
	case "activity-cancelled-disclosure-request":
		return "cancelled the disclosure request"
	case "activity-user-assigned-to-bug":
		return "assigned a user to the report"
	}
	return activityType
}
