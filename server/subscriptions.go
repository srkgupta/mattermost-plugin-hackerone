package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
)

const (
	SubscriptionsKey = "subscriptions"
)

type Subscription struct {
	ChannelID string
	CreatorID string
}

type Subscriptions struct {
	Subscriptions []*Subscription
}

func (p *Plugin) Subscribe(userID string, channelID string) error {
	sub := &Subscription{
		ChannelID: channelID,
		CreatorID: userID,
	}

	if err := p.AddSubscription(sub); err != nil {
		return errors.Wrap(err, "could not add subscription")
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

	// sort.Slice(filteredSubs, func(i, j int) bool {
	// 	return filteredSubs[i] < filteredSubs[j]
	// })

	return filteredSubs, nil
}

func (p *Plugin) AddSubscription(sub *Subscription) error {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return errors.Wrap(err, "could not get subscriptions")
	}
	exists := false
	for _, v := range subs {
		if v.ChannelID == sub.ChannelID {
			exists = true
			break
		}
	}

	if !exists {
		subs = append(subs, sub)
	}

	err = p.StoreSubscriptions(subs)
	if err != nil {
		return errors.Wrap(err, "could not store subscriptions")
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

func (p *Plugin) Unsubscribe(channelID string) error {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return errors.Wrap(err, "could not get subscriptions")
	}

	removed := false
	for index, sub := range subs {
		if sub.ChannelID == channelID {
			subs = append(subs[:index], subs[index+1:]...)
			removed = true
			break
		}
	}

	if removed {
		if err := p.StoreSubscriptions(subs); err != nil {
			return errors.Wrap(err, "could not store subscriptions")
		}
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
	case command == "check":
		return p.handleSubscriptionsList(args)
	case command == "add":
		return p.handleSubscribesAdd(args)
	case command == "delete":
		return p.handleUnsubscribe(args)
	default:
		msg := "Unknown subcommand for subscribe command. Available commands are 'list', 'add' and 'delete'."
		return p.sendEphemeralResponse(args, msg), nil
	}
}

func (p *Plugin) handleSubscribesAdd(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	err := p.Subscribe(args.UserId, args.ChannelId)
	if err != nil {
		msg := fmt.Sprintf("Something went wrong while subscribing. Error: %s\n", err.Error())
		return p.sendEphemeralResponse(args, msg), nil
	} else {
		msg := "Subscription successful! The channel will now receive notifications whenever there are any activity on your Hackerone program."
		return p.sendEphemeralResponse(args, msg), nil
	}
}

func (p *Plugin) handleUnsubscribe(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	err := p.Unsubscribe(args.ChannelId)
	if err != nil {
		msg := fmt.Sprintf("Something went wrong while unsubscribing. Error: %s\n", err.Error())
		return p.sendEphemeralResponse(args, msg), nil
	} else {
		msg := "Successfully unsubscribed! The channel will not receive notifications whenever there are any activity on your Hackerone program."
		return p.sendEphemeralResponse(args, msg), nil
	}
}

func (p *Plugin) handleSubscriptionsList(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	subs, err := p.GetSubscriptionsByChannel(args.ChannelId)
	if err != nil {
		msg := fmt.Sprintf("Something went wrong while checking for subscriptions. Error: %s\n", err.Error())
		return p.sendEphemeralResponse(args, msg), nil
	} else {
		msg := ""
		if len(subs) > 0 {
			msg = "The channel is subscribed to receive Hackerone notifications."
		} else {
			msg = "The channel is not subscribed to receive any Hackerone notifications."
		}
		return p.sendEphemeralResponse(args, msg), nil
	}
}
