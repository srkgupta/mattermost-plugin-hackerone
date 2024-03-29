package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-api/experimental/command"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
)

const (
	hackeroneCommand  = "/hackerone"
	cmdHelpKey        = "help"
	cmdStatsKey       = "stats"
	cmdPermissionsKey = "permissions"
	cmdReportKey      = "report"
	cmdReportsKey     = "reports"
	cmdSubscribeKey   = "subscriptions"
	cmdError          = "Command Error"
)

// type CommandHandlerFunc func(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse

const helpText = "###### Mattermost Hackerone Plugin\n" +
	// "* `/hackerone stats` - Gets stats info like # of new, # of pending bounty, # of pending disclosure, # of triaged reports\n" +
	"* `/hackerone reports <filter>` - Gets list of reports from Hackerone based on the filter supplied.\n" +
	"* `/hackerone report <report_id>` - Gets information about the requested report id\n" +
	"* `/hackerone subscriptions <command>` - Available subcommands: list, add, delete. Subscribe the current channel to receive Hackerone notifications. Once a channel is subscribed, the service will poll Hackerone for new activity and publish it on the subscribed channel\n" +
	"* `/hackerone permissions <command>` - Available subcommands: list, add, delete. Access Control users who can run hackerone slash commands.\n" +
	""

func (p *Plugin) getCommand(config *configuration) (*model.Command, error) {
	iconData, err := command.GetIconData(p.API, "assets/icon.svg")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get icon data")
	}

	return &model.Command{
		Trigger:              "hackerone",
		AutoComplete:         true,
		AutoCompleteDesc:     "Available commands: help, permissions, reports, report, subscriptions",
		AutoCompleteHint:     "[command]",
		AutocompleteData:     getAutocompleteData(config),
		AutocompleteIconData: iconData,
	}, nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	command := ""

	if len(split) > 1 {
		command = split[1]
	}

	if command == cmdHelpKey {
		return p.sendEphemeralResponse(args, helpText), nil
	}

	isAllowed, err := p.IsAuthorized(args.UserId)
	msg := ""
	if err != nil {
		msg = fmt.Sprintf("error occurred while authorizing the command: %v", err)
		return p.sendEphemeralResponse(args, msg), nil
	}
	if !isAllowed {
		msg = "`/hackerone` commands can only be executed by a system administrator or a list of whitelisted users. Please ask your system administrator to run the command, eg: `/hackerone permissions add @user1` to whitelist a specific user."
		return p.sendEphemeralResponse(args, msg), nil
	}

	switch command {
	case cmdReportKey:
		return p.executeReport(args, split[2:])
	case cmdReportsKey:
		return p.executeReports(args, split[2:])
	case cmdSubscribeKey:
		return p.executeSubscriptions(args, split[2:])
	case cmdPermissionsKey:
		return p.executePermissions(args, split[2:])
	default:
		return p.sendEphemeralResponse(args, helpText), nil
	}
}

func getAutocompleteData(config *configuration) *model.AutocompleteData {
	hackerone := model.NewAutocompleteData("hackerone", "[command]", "Available commands: help, reports, report, subscriptions, permissions")
	note := " NOTE: Response will be visible to all in this channel."

	help := model.NewAutocompleteData(cmdHelpKey, "", "Display Slash Command help text")
	hackerone.AddCommand(help)

	reports := model.NewAutocompleteData(cmdReportsKey, "[filters]", "Fetches reports from Hackerone as per the filter criteria specified."+note)

	newReports := model.NewAutocompleteData("new", "", "Fetches new reports from Hackerone."+note)
	reports.AddCommand(newReports)
	triagedReports := model.NewAutocompleteData("triaged", "", "Fetches triaged reports from Hackerone."+note)
	reports.AddCommand(triagedReports)
	moreInfoReports := model.NewAutocompleteData("needs-more-info", "", "Fetches reports which requires more information."+note)
	reports.AddCommand(moreInfoReports)
	bountyReports := model.NewAutocompleteData("bounty", "", "Fetches reports that is triaged & is awaiting for a bounty to be rewarded."+note)
	reports.AddCommand(bountyReports)
	disclosureReports := model.NewAutocompleteData("disclosure", "", "Fetches reports that the researchers have requested for public disclosure"+note)
	reports.AddCommand(disclosureReports)
	disclosedReports := model.NewAutocompleteData("disclosed", "", "Fetches reports that have been disclosed."+note)
	reports.AddCommand(disclosedReports)
	resolvedReports := model.NewAutocompleteData("resolved", "", "Fetches reports that have been resolved."+note)
	reports.AddCommand(resolvedReports)

	hackerone.AddCommand(reports)

	report := model.NewAutocompleteData(cmdReportKey, "[report-id]", "Gets detailed info about a Hackerone report."+note)
	hackerone.AddCommand(report)

	subscriptions := model.NewAutocompleteData(cmdSubscribeKey, "[command]", "Available commands: list, add, delete")

	subscribeAdd := model.NewAutocompleteData("add", "<report_id>(optional)", "The current channel will receive notifications when there are any activity on your Hackerone program. If report_id is not specified, it will subscribe to all the Hackerone reports")
	subscriptions.AddCommand(subscribeAdd)

	subscribeDelete := model.NewAutocompleteData("delete", "[subscriptionId]", "The specified channel will stop receiving any notifications for any events from Hackerone. You can run the command '/hackerone subscriptions list' to get the subscriptionId.")
	subscriptions.AddCommand(subscribeDelete)

	subscribeList := model.NewAutocompleteData("list", "", "Lists all the channels which has been set to receive Hackerone notifications")
	subscriptions.AddCommand(subscribeList)

	hackerone.AddCommand(subscriptions)

	permissions := model.NewAutocompleteData(cmdPermissionsKey, "[command]", "Available commands: list, allow, remove")

	permissionAdd := model.NewAutocompleteData("add", "@username", "Whitelist the user to run the Hackerone slash commands. "+permissionNote)
	permissions.AddCommand(permissionAdd)

	permissionsRemove := model.NewAutocompleteData("delete", "@username", "Remove the user from running the Hackerone slash commands. "+permissionNote)
	permissions.AddCommand(permissionsRemove)

	permissionsList := model.NewAutocompleteData("list", "", "List all the users who are allowed to run the Hackerone slash commands. "+permissionNote)
	permissions.AddCommand(permissionsList)

	hackerone.AddCommand(permissions)

	return hackerone
}
