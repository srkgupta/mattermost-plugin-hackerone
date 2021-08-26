package main

import (
	"fmt"

	"github.com/mattermost/mattermost-server/v5/model"
)

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
			postAttachments := []*model.SlackAttachment{}
			attachment := p.getReportAttachment(report, true)
			postAttachments = append(postAttachments, attachment)
			_ = p.sendPost(args, "", postAttachments)
		}
	}
	return &model.CommandResponse{}, nil
}

func (p *Plugin) executeReports(args *model.CommandArgs, split []string) (*model.CommandResponse, *model.AppError) {
	state := ""
	if len(split) > 0 {
		state = split[0]
		allowedStates := []string{"all", "new", "triaged", "needs-more-info", "bounty", "disclosure", "disclosed", "resolved"}
		if !p.contains(allowedStates, state) {
			msg := "Incorrect filter option applied. Please select a valid option from the autocomplete."
			return p.sendEphemeralResponse(args, msg), nil
		}
	} else {
		msg := "Filter not provided. Please select a valid option from the autocomplete."
		return p.sendEphemeralResponse(args, msg), nil
	}

	filters := make(map[string]string)
	title := ""
	hackeroneStates := []string{"new", "triaged", "needs-more-info", "resolved"}
	if p.contains(hackeroneStates, state) {
		filters["state"] = state
		title = "Displaying reports with the state: `" + state + "`"
	} else if state == "bounty" {
		filters["state"] = "triaged"
		filters["bounty_awarded_at__null"] = "true"
		title = "Displaying reports that is `triaged` & is awaiting for a `bounty to be rewarded`:"
	} else if state == "disclosure" {
		filters["reporter_agreed_on_going_public"] = "true"
		title = "Displaying reports that the researchers have requested for `public disclosure`:"
	} else if state == "disclosed" {
		filters["disclosed_at__null"] = "false"
		title = "Displaying reports that have been `disclosed`:"
	} else {
		title = "Displaying all reports:"
	}

	reports, err := p.fetchReports(filters)
	if err != nil {
		msg := fmt.Sprintf("Something went wrong while getting the reports from Hackerone API. Error: %s\n", err.Error())
		return p.sendEphemeralResponse(args, msg), nil
	} else {
		reportString := "#### " + title + "\n\n"
		if len(reports) > 0 {
			postAttachments := []*model.SlackAttachment{}
			for _, report := range reports {
				attachment := p.getReportAttachment(report, false)
				postAttachments = append(postAttachments, attachment)
			}
			_ = p.sendPost(args, reportString, postAttachments)
		} else {
			msg := "No reports found matching the filter criteria you have specified."
			return p.sendEphemeralResponse(args, reportString+msg), nil
		}
	}

	return &model.CommandResponse{}, nil
}

func (p *Plugin) getReportAttachment(report Report, description bool) *model.SlackAttachment {
	fields := []*model.SlackAttachmentField{
		{
			Title: "Report Id",
			Value: report.Id,
			Short: true,
		},
		{
			Title: "State",
			Value: report.Attributes.State,
			Short: true,
		},
		{
			Title: "Created At",
			Value: p.parseTime(report.Attributes.CreatedAt),
			Short: true,
		},
		{
			Title: "Triaged At",
			Value: p.parseTime(report.Attributes.TriagedAt),
			Short: true,
		},
		{
			Title: "Bounty Awarded At",
			Value: p.parseTime(report.Attributes.BountyAwardedAt),
			Short: true,
		},
		{
			Title: "Closed At",
			Value: p.parseTime(report.Attributes.ClosedAt),
			Short: true,
		},
		{
			Title: "Disclosed At",
			Value: p.parseTime(report.Attributes.DisclosedAt),
			Short: true,
		},
	}
	if description {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Report Details",
			Value: report.Attributes.Info,
			Short: false,
		},
		)
	}
	return &model.SlackAttachment{
		Title:      report.Attributes.Title,
		TitleLink:  "https://hackerone.com/reports/" + report.Id,
		AuthorName: report.Relationships.Reporter.Data.Attributes.Name,
		AuthorLink: "https://hackerone.com/" + report.Relationships.Reporter.Data.Attributes.Username,
		AuthorIcon: report.Relationships.Reporter.Data.Attributes.ProfilePicture.Photo,
		Timestamp:  report.Attributes.CreatedAt,
		Fields:     fields,
	}
}
