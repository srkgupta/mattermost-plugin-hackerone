package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
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
	activities, err := p.fetchActivities(count, "")
	if err != nil {
		msg := fmt.Sprintf("Something went wrong while getting the activities from Hackerone API. Error: %s\n", err.Error())
		return p.sendEphemeralResponse(args, msg), nil
	} else {
		for _, activity := range activities.Activities {
			activitiesListString += p.activityTemplate(activity)
		}
		_ = p.sendPost(args, activitiesListString, nil)
	}
	return &model.CommandResponse{}, nil
}

func (p *Plugin) activityTemplate(activity Activity) string {
	activitiesListString := ""
	reportLink := "[report " + activity.Attributes.ReportID + "](https://hackerone.com/reports/" + activity.Attributes.ReportID + ")"
	actorLink := "[" + activity.Relationships.Actor.Data.Attributes.Name + "](https://hackerone.com/" + activity.Relationships.Actor.Data.Attributes.Username + ")"
	activityLink := strings.Replace(getActivityType(activity.ActivityType), "report", reportLink, -1)
	activitiesListString += fmt.Sprintf(
		"> %s %s at %s\n",
		actorLink,
		activityLink,
		p.parseTime(activity.Attributes.CreatedAt),
	)
	if len(activity.Attributes.Message) > 1 {
		activitiesListString += "\n```\n" + activity.Attributes.Message + "\n```\n"
	}
	return activitiesListString
}

func (p *Plugin) notifyNewActivity() error {
	last_updated_at, err := p.GetActivityLastKey()
	if err != nil {
		return errors.Wrap(err, "error while notifying new activity")
	}
	activities, err := p.fetchActivities("100", last_updated_at)
	activitiesListString := ""
	if err != nil {
		return errors.Wrap(err, "Something went wrong while getting the activities from Hackerone API.")
	} else {
		for _, activity := range activities.Activities {
			activitiesListString += p.activityTemplate(activity)
		}

		subs, _ := p.GetSubscriptions()
		for _, v := range subs {
			p.sendPostByChannelId(v.ChannelID, activitiesListString, nil)
		}
		if len(activities.Meta.MaxUpdatedAt) > 1 {
			p.StoreActivityLastKey(activities.Meta.MaxUpdatedAt)
		}
	}
	return nil
}
