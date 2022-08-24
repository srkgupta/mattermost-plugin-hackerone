package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

const (
	SubscriptionsKey = "subscriptions"
)

type Subscription struct {
	ID        string
	ChannelID string
	CreatorID string
	ReportID  string
}

type Subscriptions struct {
	Subscriptions []*Subscription
}

func generateUUIDName() string {
	id := uuid.New()
	return (id.String())
}

func (p *Plugin) Subscribe(userID string, channelID string, reportID string) error {
	sub := &Subscription{
		ID:        generateUUIDName(),
		ChannelID: channelID,
		CreatorID: userID,
		ReportID:  reportID,
	}

	if err := p.AddSubscription(sub); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) GetSubscriptionsByChannel(channelID string) ([]*Subscription, error) {
	var filteredSubs []*Subscription
	subs, err := p.GetSubscriptions()
	if err != nil {
		return nil, errors.Wrap(err, "could not get subscriptions")
	}

	for _, sub := range subs {
		if sub.ChannelID == channelID {
			filteredSubs = append(filteredSubs, sub)
		}
	}

	return filteredSubs, nil
}

func (p *Plugin) AddSubscription(sub *Subscription) error {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return errors.Wrap(err, "could not get subscriptions")
	}
	for _, v := range subs {
		if v.ChannelID == sub.ChannelID {
			if len(sub.ReportID) > 0 && len(v.ReportID) > 0 && v.ReportID == sub.ReportID {
				return errors.New(fmt.Sprintf("This channel is already subscribed to receive notifications for the report ID: %s", v.ReportID))
			}

			if len(v.ReportID) == 0 {
				return errors.New("This channel is already subscribed to receive notifications for all reports")
			}

			// If user is trying to add subscription for all reports when existing subscription exists for individual reports, then delete the previous subscriptions
			if len(sub.ReportID) == 0 {
				newSubs := []*Subscription{}

				for _, newSub := range subs {
					if sub.ChannelID != newSub.ChannelID {
						newSubs = append(newSubs, newSub)
					}
				}
				subs = newSubs
				break
			}

		}
	}

	subs = append(subs, sub)

	err = p.StoreSubscriptions(subs)
	if err != nil {
		return err
	}

	return nil
}

func (p *Plugin) GetSubscriptions() ([]*Subscription, error) {
	var subscriptions []*Subscription

	value, appErr := p.API.KVGet(SubscriptionsKey)
	if appErr != nil {
		return nil, errors.Wrap(appErr, "could not get subscriptions from KVStore")
	}

	if value == nil {
		return []*Subscription{}, nil
	}

	err := json.NewDecoder(bytes.NewReader(value)).Decode(&subscriptions)
	if err != nil {
		return nil, errors.Wrap(err, "could not properly decode subscriptions key")
	}

	return subscriptions, nil
}

func (p *Plugin) StoreSubscriptions(s []*Subscription) error {
	b, err := json.Marshal(s)
	if err != nil {
		return errors.Wrap(err, "error while converting subscriptions to json")
	}

	if appErr := p.API.KVSet(SubscriptionsKey, b); appErr != nil {
		return errors.Wrap(appErr, "could not store subscriptions in KV store")
	}

	return nil
}

func (p *Plugin) Unsubscribe(id string) error {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return errors.Wrap(err, "could not get subscriptions")
	}

	newSubs := []*Subscription{}

	for _, sub := range subs {
		if sub.ID != id {
			newSubs = append(newSubs, sub)
		}
	}

	if err := p.StoreSubscriptions(newSubs); err != nil {
		return errors.Wrap(err, "could not store subscriptions")
	}

	return nil
}

func (p *Plugin) executeSubscriptions(args *model.CommandArgs, split []string) (*model.CommandResponse, *model.AppError) {
	if 0 >= len(split) {
		msg := "Invalid subscribe command. Available commands are 'list', 'add' and 'delete'."
		return p.sendEphemeralResponse(args, msg), nil
	}

	command := split[0]

	switch {
	case command == "list":
		return p.handleSubscriptionsList(args)
	case command == "add":
		reportId := ""
		if len(split) >= 2 {
			reportId = split[1]
		}
		return p.handleSubscribesAdd(args, reportId)
	case command == "delete":
		if len(split) < 2 {
			msg := "Please specify the subscriptionId to be removed. You can run the command '/hackerone subscriptions list' to get the subscriptionId."
			return p.sendEphemeralResponse(args, msg), nil
		} else {
			return p.handleUnsubscribe(args, split[1])
		}
	default:
		msg := "Unknown subcommand for subscribe command. Available commands are 'list', 'add' and 'delete'."
		return p.sendEphemeralResponse(args, msg), nil
	}
}

func (p *Plugin) handleSubscribesAdd(args *model.CommandArgs, reportID string) (*model.CommandResponse, *model.AppError) {
	err := p.Subscribe(args.UserId, args.ChannelId, reportID)
	if err != nil {
		msg := err.Error()
		return p.sendEphemeralResponse(args, msg), nil
	}
	msg := "Subscription successful for all Hackerone reports."
	if len(reportID) > 0 {
		msg = "Subscription successful for Hackerone report id: " + reportID
	}
	return p.sendEphemeralResponse(args, msg), nil
}

func (p *Plugin) handleUnsubscribe(args *model.CommandArgs, ID string) (*model.CommandResponse, *model.AppError) {
	err := p.Unsubscribe(ID)
	if err != nil {
		msg := fmt.Sprintf("Something went wrong while unsubscribing. Error: %s\n", err.Error())
		return p.sendEphemeralResponse(args, msg), nil
	}
	msg := "Successfully unsubscribed! The specified channel will not receive Hackerone notifications."
	return p.sendEphemeralResponse(args, msg), nil
}

func (p *Plugin) handleSubscriptionsList(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	subs, err := p.GetSubscriptions()
	msg := ""
	if err != nil {
		msg = fmt.Sprintf("Something went wrong while checking for subscriptions. Error: %s\n", err.Error())
		return p.sendEphemeralResponse(args, msg), nil
	}
	if len(subs) == 0 {
		msg = "Currently there are no channels subscribed to receive Hackerone notifications."
	} else {
		msg = "##### Channels subscribed to receive Hackerone notifications:\n\n"
		msg += "| Channel | Type | Subscription ID |\n"
		msg += "| ----------- | ----------- | ----------- | \n"
		for _, v := range subs {
			channel, _ := p.API.GetChannel(v.ChannelID)
			if len(v.ReportID) > 0 {
				msg += fmt.Sprintf("| ~%s | Report ID =%s | %s |\n", channel.Name, v.ReportID, v.ID)
			} else {
				msg += fmt.Sprintf("| ~%s | All Reports | %s |\n", channel.Name, v.ID)
			}
		}
	}
	return p.sendEphemeralResponse(args, msg), nil

}
