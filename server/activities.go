package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

const (
	ActivityLastKey = "activities-last"
)

func (p *Plugin) GetActivityLastKey() (string, error) {
	value, appErr := p.API.KVGet(ActivityLastKey)
	tm := "1970-01-01T00:00:00Z"

	if appErr != nil {
		return tm, errors.Wrap(appErr, "could not get activity last key from KVStore")
	}

	if value == nil {
		return tm, nil
	}

	return string(value), nil
}

func (p *Plugin) StoreActivityLastKey(value string) error {
	_, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return errors.Wrap(err, "invalid format of activity last key. It should be in the format: 2017-02-02T04:05:06.000Z")
	}

	if appErr := p.API.KVSet(ActivityLastKey, []byte(value)); appErr != nil {
		return errors.Wrap(appErr, "could not store activity last key in KV store")
	}

	return nil
}

func (p *Plugin) activityTemplate(activity Activity) string {
	activitiesListString := ""
	name := activity.Relationships.Actor.Data.Attributes.Name
	if len(name) < 1 {
		name = activity.Relationships.Actor.Data.Attributes.Username
	}
	actorLink := "[" + name + "](https://hackerone.com/" + activity.Relationships.Actor.Data.Attributes.Username + ")"
	activitiesListString += fmt.Sprintf(
		"%s %s\n",
		actorLink,
		getActivityType(activity.ActivityType),
	)
	if len(activity.Attributes.Message) > 1 {
		activitiesListString += "\n```\n" + activity.Attributes.Message + "\n```\n"
	}
	return activitiesListString
}

func (p *Plugin) notifyNewActivity() error {
	subs, _ := p.GetSubscriptions()
	if len(subs) == 0 {
		return nil
	}

	last_updated_at, err := p.GetActivityLastKey()
	if err != nil {
		p.API.LogWarn("Error while notifying new activity", "error", err.Error())
		return errors.Wrap(err, "error while notifying new activity")
	}
	activities, err := p.fetchActivities("100", last_updated_at)
	if err != nil {
		p.API.LogWarn("Something went wrong while getting the activities from Hackerone API", "error", err.Error())
		return errors.Wrap(err, "Something went wrong while getting the activities from Hackerone API.")
	}

	if len(activities.Activities) == 0 {
		return nil
	}

	// do not notify all previous activities, store the last activity timestamp and only display new activities
	if last_updated_at == "1970-01-01T00:00:00Z" {
		p.StoreActivityLastKey(activities.Meta.MaxUpdatedAt)
		return nil
	}

	for _, activity := range activities.Activities {
		activitiesListString := p.activityTemplate(activity)
		postAttachments := []*model.SlackAttachment{}
		report, err := p.fetchReport(activity.Attributes.ReportID)
		if err != nil {
			p.API.LogWarn("Something went wrong while getting the report from Hackerone API", "error", err.Error())
		} else {
			var attachment = &model.SlackAttachment{}
			if activity.ActivityType == "activity-bug-filed" {
				attachment = p.getReportAttachment(report, true)
			} else {
				attachment = p.getReportAttachment(report, false)
			}
			postAttachments = append(postAttachments, attachment)
		}
		for _, v := range subs {
			if (len(v.ReportID) == 0) || (v.ReportID == activity.Attributes.ReportID) {
				p.sendPostByChannelId(v.ChannelID, activitiesListString, postAttachments)
			}
		}
	}

	if len(activities.Meta.MaxUpdatedAt) > 0 {
		p.StoreActivityLastKey(activities.Meta.MaxUpdatedAt)
	}
	return nil
}

func getActivityType(activityType string) string {
	switch activityType {
	case "activity-agreed-on-going-public":
		return "agreed on going public on the report"
	case "activity-bounty-awarded":
		return "awarded a bounty on the report"
	case "activity-bounty-suggested":
		return "suggested a bounty on the report"
	case "activity-bug-cloned":
		return "cloned the report"
	case "activity-comment":
		return "commented on the report"
	case "activity-bug-duplicate":
		return "closed the report as Duplicate"
	case "activity-bug-triaged":
		return "triaged the report"
	case "activity-bug-inactive":
		return "marked the report as Inactive"
	case "activity-bug-new":
		return "marked the report as New"
	case "activity-bug-retesting":
		return "marked the report for Retesting"
	case "activity-bug-spam":
		return "marked the report as Spam"
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
	case "activity-changed-scope":
		return "changed the scope on the report"
	case "activity-comments-closed":
		return "locked the report"
	case "activity-external-user-invited":
		return "invited an external user on the report"
	case "activity-external-user-joined":
		return "joined the report as an external user"
	case "activity-external-user-removed":
		return "removed an external user from the report"
	case "activity-group-assigned-to-bug":
		return "assigned the report to a group"
	case "activity-hacker-requested-mediation":
		return "requested for mediation"
	case "activity-manually-disclosed":
		return "manually disclosed"
	case "activity-mediation-requested":
		return "requested for mediation"
	case "activity-nobody-assigned-to-bug":
		return "removed the asignee on the report"
	case "activity-not-eligible-for-bounty":
		return "marked the report as not eligible for bounty"
	case "activity-program-inactive":
		return "marked the status as program inactive on the report"
	case "activity-reference-id-added":
		return "added a reference id on the report"
	case "activity-report-became-public":
		return "disclosed the report as public"
	case "activity-report-custom-field-value-updated":
		return "updated the custom field on the report"
	case "activity-report-retest-approved":
		return "approved the retesting on the report"
	case "activity-report-retest-rejected":
		return "rejected the retesting on the report"
	case "activity-report-severity-updated":
		return "updated the severity on the report"
	case "activity-report-title-updated":
		return "updated the title on the report"
	case "activity-report-vulnerability-types-updated":
		return "updated the vulnerability type of the report"
	case "activity-retest-user-expired":
		return "retesting request was timed out on the report"
	case "activity-swag-awarded":
		return "awarded a swag on the report"
	case "activity-user-banned-from-program":
		return "banned a user from the program on the report"
	case "activity-user-completed-retest":
		return "completed retesting on the report"
	case "activity-user-left-retest":
		return "left the retesting on the report"
	}
	return activityType
}
