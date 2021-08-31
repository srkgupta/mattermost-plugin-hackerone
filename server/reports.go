package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
)

const (
	URLTrigger    = "/trigger"
	contextReport = "report"
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

func (p *Plugin) getReportAttachment(report Report, detailed bool) *model.SlackAttachment {
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
	}

	if len(report.Attributes.TriagedAt) > 1 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Triaged At",
			Value: p.parseTime(report.Attributes.TriagedAt),
			Short: true,
		},
		)
	}

	if len(report.Attributes.BountyAwardedAt) > 1 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Bounty Awarded At",
			Value: p.parseTime(report.Attributes.BountyAwardedAt),
			Short: true,
		},
		)
	}

	if len(report.Attributes.ClosedAt) > 1 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Closed At",
			Value: p.parseTime(report.Attributes.ClosedAt),
			Short: true,
		},
		)
	}

	if len(report.Attributes.DisclosedAt) > 1 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Disclosed At",
			Value: p.parseTime(report.Attributes.DisclosedAt),
			Short: true,
		},
		)
	}

	if detailed {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Report Details",
			Value: report.Attributes.Info,
			Short: false,
		},
		)
	}
	// actionContext := map[string]interface{}{
	// 	contextReport: report.Id,
	// }

	// actions := []*model.PostAction{}
	// actions = append(actions, generateButton("Trigger Incident", URLTrigger, actionContext))

	return &model.SlackAttachment{
		Title:      report.Attributes.Title,
		TitleLink:  "https://hackerone.com/reports/" + report.Id,
		AuthorName: report.Relationships.Reporter.Data.Attributes.Name,
		AuthorLink: "https://hackerone.com/" + report.Relationships.Reporter.Data.Attributes.Username,
		Timestamp:  report.Attributes.CreatedAt,
		Fields:     fields,
	}
}

// Generate an attachment for an action Button that will point to a plugin HTTP handler
func generateButton(name string, urlAction string, context map[string]interface{}) *model.PostAction {
	return &model.PostAction{
		Name: name,
		Type: model.POST_ACTION_TYPE_BUTTON,
		Integration: &model.PostActionIntegration{
			URL:     fmt.Sprintf("/plugins/mattermost-plugin-hackerone/%s", urlAction),
			Context: context,
		},
	}
}

func (p *Plugin) notifyReports(filters map[string]string, title string, description string) error {
	subs, _ := p.GetSubscriptions()
	reports, err := p.fetchReports(filters)
	if err != nil {
		p.API.LogWarn("Error while fetching Reports from Hackerone", "error", err.Error())
		return errors.Wrap(err, "error while notifying missed deadline reports")
	} else {
		reportString := "#### " + title + "\n" + description + "\n\n"
		if len(reports) > 0 {
			postAttachments := []*model.SlackAttachment{}
			for _, report := range reports {
				attachment := p.getReportAttachment(report, false)
				postAttachments = append(postAttachments, attachment)
			}
			for _, v := range subs {
				p.sendPostByChannelId(v.ChannelID, reportString, postAttachments)
			}
		}
	}
	return nil
}

func (p *Plugin) notifyMissedDeadlineReports() error {
	subs, _ := p.GetSubscriptions()
	if len(subs) > 0 {
		// Check for Missed Deadline - New Reports
		filters := getDeadlineReportFilter("new", p.getConfiguration().HackeroneSLANew)
		desc := fmt.Sprintf("These reports have not been triaged for more than %d days and hence have missed SLA deadlines.", p.getConfiguration().HackeroneSLANew)
		p.notifyReports(filters, "Missed SLA Deadline - New Reports:", desc)

		// Check for Missed Deadline - Pending Bounty
		filters = getDeadlineReportFilter("bounty", p.getConfiguration().HackeroneSLABounty)
		desc = fmt.Sprintf("Bounty has not been rewarded for these triaged reports for more than %d days and hence have missed SLA deadlines.", p.getConfiguration().HackeroneSLABounty)
		p.notifyReports(filters, "Missed SLA Deadline - Bounty to be rewarded:", desc)

		// Check for Missed Deadline - Triaged Reports
		filters = getDeadlineReportFilter("triaged", p.getConfiguration().HackeroneSLATriaged)
		desc = fmt.Sprintf("These triaged reports have not been resolved for more than %d days and hence have missed SLA deadlines.", p.getConfiguration().HackeroneSLATriaged)
		p.notifyReports(filters, "Missed SLA Deadline - Triaged reports to be resolved:", desc)
	}
	return nil
}

func getDeadlineReportFilter(filterType string, sla int) map[string]string {
	filters := make(map[string]string)
	now := time.Now().UTC()
	min_created_at := now.AddDate(0, 0, -sla)
	filters["created_at__lt"] = min_created_at.Format(time.RFC3339)
	if filterType == "new" {
		filters["state"] = "new"
		filters["triaged_at__null"] = "true"
	} else if filterType == "bounty" {
		filters["state"] = "triaged"
		filters["bounty_awarded_at__null"] = "true"
	} else if filterType == "triaged" {
		filters["state"] = "triaged"
		filters["bounty_awarded_at__null"] = "false"
		filters["closed_at__null"] = "false"
	}
	return filters
}
