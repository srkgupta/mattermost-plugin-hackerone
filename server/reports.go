package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

const (
	URLTrigger    = "/trigger"
	contextReport = "report"
)

func (p *Plugin) executeReport(args *model.CommandArgs, split []string) (*model.CommandResponse, *model.AppError) {
	if len(split) <= 0 {
		msg := "Report Id should be specified while fetching the report information"
		return p.sendEphemeralResponse(args, msg), nil
	}

	reportId := split[0]
	report, err := p.fetchReport(reportId)
	if err != nil {
		msg := fmt.Sprintf("Something went wrong while getting the report from Hackerone API. Error: %s\n", err.Error())
		return p.sendEphemeralResponse(args, msg), nil
	}

	postAttachments := []*model.SlackAttachment{}
	attachment := p.getReportAttachment(report, true)
	postAttachments = append(postAttachments, attachment)
	_ = p.sendPost(args, "", postAttachments)
	return &model.CommandResponse{}, nil
}

func (p *Plugin) executeReports(args *model.CommandArgs, split []string) (*model.CommandResponse, *model.AppError) {
	state := ""
	if len(split) <= 0 {
		msg := "Filter not provided. Please select a valid option from the autocomplete."
		return p.sendEphemeralResponse(args, msg), nil
	}

	state = split[0]
	allowedStates := []string{"new", "triaged", "needs-more-info", "bounty", "disclosure", "disclosed", "resolved"}
	if !contains(allowedStates, state) {
		msg := "Incorrect filter option applied. Please select a valid option from the autocomplete."
		return p.sendEphemeralResponse(args, msg), nil
	}

	filters := make(map[string]string)
	title := ""
	hackeroneStates := []string{"new", "triaged", "needs-more-info", "resolved"}
	if contains(hackeroneStates, state) {
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
	}

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
			Value: parseTime(report.Attributes.CreatedAt),
			Short: true,
		},
	}

	if len(report.Attributes.TriagedAt) > 0 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Triaged At",
			Value: parseTime(report.Attributes.TriagedAt),
			Short: true,
		},
		)
	}

	if len(report.Attributes.BountyAwardedAt) > 0 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Bounty Awarded At",
			Value: parseTime(report.Attributes.BountyAwardedAt),
			Short: true,
		},
		)
	}

	if len(report.Attributes.ClosedAt) > 0 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Closed At",
			Value: parseTime(report.Attributes.ClosedAt),
			Short: true,
		},
		)
	}

	if len(report.Attributes.DisclosedAt) > 0 {
		fields = append(fields, &model.SlackAttachmentField{
			Title: "Disclosed At",
			Value: parseTime(report.Attributes.DisclosedAt),
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
		Type: model.PostActionTypeButton,
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
	}

	if len(reports) == 0 {
		return nil
	}

	reportString := "#### " + title + "\n" + description + "\n\n"
	// Each subscription can either be for a single reportId or for all reports
	for _, s := range subs {
		found := false
		postAttachments := []*model.SlackAttachment{}
		for _, report := range reports {
			// If subscription has a report Id, only notify the subscription's report ID
			if len(s.ReportID) > 0 {
				// Notify only if subscription's report ID is equal to report fetched from Hackerone
				if s.ReportID == report.Id {
					attachment := p.getReportAttachment(report, false)
					postAttachments = append(postAttachments, attachment)
					found = true
					break
				}
			} else {
				// Else it means it is subscribed to receive all reports info
				attachment := p.getReportAttachment(report, false)
				postAttachments = append(postAttachments, attachment)
				found = true
			}
		}
		if found {
			p.sendPostByChannelId(s.ChannelID, reportString, postAttachments)
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
	switch filterType {
	case "new":
		filters["state"] = "new"
		filters["triaged_at__null"] = "true"
	case "bounty":
		filters["state"] = "triaged"
		filters["bounty_awarded_at__null"] = "true"
	case "triaged":
		filters["state"] = "triaged"
		filters["bounty_awarded_at__null"] = "false"
		filters["closed_at__null"] = "false"
	}
	return filters
}
