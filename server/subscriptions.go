package main

import (
	"bytes"
	"encoding/json"

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
