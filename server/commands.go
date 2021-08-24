package main

import (
	"strings"

	"github.com/mattermost/mattermost-plugin-api/experimental/command"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
)

const (
	hackeroneCommand  = "/hackerone"
	helpCmdKey        = "help"
	statsCmdKey       = "stats"
	activityCmdKey    = "activities"
	pendingCmdKey     = "pending_reports"
	reportCmdKey      = "report"
	reportsCmdKey     = "reports"
	deadlineCmdKey    = "deadline_reports"
	subscribeCmdKey   = "subscribe"
	unsubscribeCmdKey = "unsubscribe"
	cmdError          = "Command Error"
)

// type CommandHandlerFunc func(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse

const helpText = "###### Mattermost Hackerone Plugin\n" +
	// "* `/hackerone stats` - Gets stats info like # of new, # of pending bounty, # of pending disclosure, # of triaged reports\n" +
	"* `/hackerone activities <count>` - Gets most recent activities of your program\n" +
	"* `/hackerone reports <status>` - Gets list of reports from Hackerone based on the status filter.\n" +
	"* `/hackerone report <report_id>` - Gets information about the requested report id\n" +
	"* `/hackerone pending_reports <new/triaged/disclosure>` - Gets a list of pending reports according to the criteria specified - new, triaged, disclosure\n" +
	"* `/hackerone deadline_reports <new/triaged>` - Gets a list of reports which have exceeded the time deadline. Example: it will list all the reports which have been triaged for more than 1 month\n" +
	"* `/hackerone subscribe` - Subscribe the current channel to receive Hackerone notifications. Once a channel is subscribed, the service will poll Hackerone every 30 seconds and check for new content. If any new content is found, it will be shown on the channel\n" +
	"* `/hackerone unsubscribe` - Once unsubscribed, the channel will stop receiving any notifications for any events from Hackerone\n" +
	""

func (p *Plugin) getCommand(config *configuration) (*model.Command, error) {
	iconData, err := command.GetIconData(p.API, "assets/icon.svg")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get icon data")
	}

	return &model.Command{
		Trigger:              "hackerone",
		AutoComplete:         true,
		AutoCompleteDesc:     "Available commands: stats, activities, reports, report, pending_reports, deadline_reports, help, subscribe, triggers, unsubscribe",
		AutoCompleteHint:     "[command]",
		AutocompleteData:     getAutocompleteData(config),
		AutocompleteIconData: iconData,
	}, nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	command := ""

	if 1 < len(split) {
		command = split[1]
	}

	if command == helpCmdKey {
		return p.sendEphemeralResponse(args, helpText), nil
	}

	switch command {
	case activityCmdKey:
		return p.executeActivities(args, split[2:])
	case statsCmdKey:
		return p.executeStats(args, split[2:])
	case reportCmdKey:
		return p.executeReport(args, split[2:])
	case reportsCmdKey:
		return p.executeReports(args, split[2:])
	default:
		return p.sendEphemeralResponse(args, helpText), nil
	}
}

func getAutocompleteData(config *configuration) *model.AutocompleteData {
	hackerone := model.NewAutocompleteData("hackerone", "[command]", "Available commands: stats, pending, deadline, help, subscribe, triggers, unsubscribe")
	note := " NOTE: Response will be visible to all in this channel."

	help := model.NewAutocompleteData(helpCmdKey, "", "Display Slash Command help text")
	hackerone.AddCommand(help)

	activities := model.NewAutocompleteData(activityCmdKey, "[positive integer]", "Gets most recent activities of your program."+note)
	hackerone.AddCommand(activities)

	// stats := model.NewAutocompleteData(statsCmdKey, "", "Gets stats info like # of new, # of pending bounty, # of pending disclosure, # of triaged reports. NOTE: Response will be visible to all users in this channel.")
	// hackerone.AddCommand(stats)

	reports := model.NewAutocompleteData(reportsCmdKey, "[filters]", "Fetches reports from Hackerone as per the filter criteria specified."+note)

	allReports := model.NewAutocompleteData("all", "", "Fetches all reports from Hackerone."+note)
	reports.AddCommand(allReports)
	newReports := model.NewAutocompleteData("new", "", "Fetches new reports from Hackerone."+note)
	reports.AddCommand(newReports)
	triagedReports := model.NewAutocompleteData("triaged", "", "Fetches triaged reports from Hackerone."+note)
	reports.AddCommand(triagedReports)
	moreInfoReports := model.NewAutocompleteData("needs-more-info", "", "Fetches reports which requires more information."+note)
	reports.AddCommand(moreInfoReports)
	bountyReports := model.NewAutocompleteData("bounty", "", "Fetches reports that is triaged & is awaiting for a bounty to be rewarded."+note)
	reports.AddCommand(bountyReports)
	disclosureReports := model.NewAutocompleteData("disclosure", "", "Fetches reports that have the hacker disclosure request"+note)
	reports.AddCommand(disclosureReports)
	disclosedReports := model.NewAutocompleteData("disclosed", "", "Fetches reports that have been disclosed."+note)
	reports.AddCommand(disclosedReports)
	resolvedReports := model.NewAutocompleteData("resolved", "", "Fetches reports that have been resolved."+note)
	reports.AddCommand(resolvedReports)

	hackerone.AddCommand(reports)

	report := model.NewAutocompleteData(reportCmdKey, "[report-id]", "Gets detailed info about a Hackerone report."+note)
	hackerone.AddCommand(report)

	subscribe := model.NewAutocompleteData(subscribeCmdKey, "", "Subscribe the current channel to receive Hackerone notifications.")
	hackerone.AddCommand(subscribe)

	unsubscribe := model.NewAutocompleteData(unsubscribeCmdKey, "", "Once unsubscribed, the channel will stop receiving any notifications for any events from Hackerone.")
	hackerone.AddCommand(unsubscribe)

	return hackerone
}
