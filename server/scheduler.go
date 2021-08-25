// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

const (
	HackeroneNewActivity    = "new-activity"
	HackeroneMissedDeadline = "missed-deadline"
	MissedDeadlineInterval  = 3600
)

type TaskFunc func()

type ScheduledTask struct {
	Name      string        `json:"name"`
	Interval  time.Duration `json:"interval"`
	Recurring bool          `json:"recurring"`
	function  func()
	cancel    chan struct{}
	cancelled chan struct{}
}

func (p *Plugin) createHackeroneRecurring() {
	interval := time.Duration(p.getConfiguration().HackeronePollIntervalSeconds)
	newActivityTask := createTask(HackeroneNewActivity, func() { p.notifyNewActivity() }, interval, true)
	p.scheduledTasks = append(p.scheduledTasks, newActivityTask)
	missedDeadlineTask := createTask(HackeroneMissedDeadline, func() { p.notifyMissedDeadline() }, MissedDeadlineInterval, true)
	p.scheduledTasks = append(p.scheduledTasks, missedDeadlineTask)
}

func (p *Plugin) cancelHackeroneRecurring() {
	for _, t := range p.scheduledTasks {
		t.Cancel()
	}
	p.scheduledTasks = []*ScheduledTask{}
}

func (p *Plugin) notifyMissedDeadline() error {
	subs, err := p.GetSubscriptions()
	if err != nil {
		return errors.Wrap(err, "could not get subscriptions")
	}
	for _, v := range subs {
		tm := time.Now()
		msg := fmt.Sprintf("Missed Deadline: %s", tm)
		p.sendPostByChannelId(v.ChannelID, msg, nil)
	}
	return nil
}

func createTask(name string, function TaskFunc, interval time.Duration, recurring bool) *ScheduledTask {
	task := &ScheduledTask{
		Name:      name,
		Interval:  interval,
		Recurring: recurring,
		function:  function,
		cancel:    make(chan struct{}),
		cancelled: make(chan struct{}),
	}

	go func() {
		defer close(task.cancelled)

		ticker := time.NewTicker(interval * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				function()
			case <-task.cancel:
				return
			}

			if !task.Recurring {
				break
			}
		}
	}()

	return task
}

func (task *ScheduledTask) Cancel() {
	close(task.cancel)
	<-task.cancelled
}

func (task *ScheduledTask) String() string {
	return fmt.Sprintf(
		"%s\nInterval: %s\nRecurring: %t\n",
		task.Name,
		task.Interval.String(),
		task.Recurring,
	)
}
