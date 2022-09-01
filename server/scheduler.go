// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"sync"
	"time"

	"github.com/mattermost/mattermost-plugin-api/cluster"
	"github.com/pkg/errors"
)

const (
	HackeroneNewActivity    = "new-activity"
	HackeroneMissedDeadline = "missed-deadline"
)

type TaskFunc func()

type JobManager struct {
	registeredJobs sync.Map
	activeJobs     sync.Map
	papi           cluster.JobPluginAPI
}

type RegisteredJob struct {
	id       string
	interval time.Duration
}

type ScheduledTask struct {
	Name      string        `json:"name"`
	Interval  time.Duration `json:"interval"`
	Recurring bool          `json:"recurring"`
	function  func()
	cancel    chan struct{}
	cancelled chan struct{}
}

func (p *Plugin) createHackeroneRecurring() {
	interval := time.Duration(p.getConfiguration().HackeronePollIntervalSeconds) * time.Second
	newActivityJob, err := p.createNewJob(HackeroneNewActivity, func() { p.notifyNewActivity() }, interval)
	if err != nil {
		p.API.LogError("Error while scheduling Hackerone job to notify new activity", "err", err.Error())
	}
	if newActivityJob != nil {
		p.scheduledJobs = append(p.scheduledJobs, newActivityJob)
	}

	slaInterval := time.Duration(p.getConfiguration().HackeroneSLAPollIntervalSeconds) * time.Second
	missedDeadlineJob, err := p.createNewJob(HackeroneMissedDeadline, func() { p.notifyMissedDeadlineReports() }, slaInterval)
	if err != nil {
		p.API.LogError("Error while scheduling Hackerone job to notify missed deadlines", "err", err.Error())
	}
	if missedDeadlineJob != nil {
		p.scheduledJobs = append(p.scheduledJobs, missedDeadlineJob)
	}

}

func (p *Plugin) cancelHackeroneRecurring() {
	for _, job := range p.scheduledJobs {
		job.Close()
	}
	p.scheduledJobs = []*cluster.Job{}
}

func (p *Plugin) createNewJob(name string, function TaskFunc, interval time.Duration) (*cluster.Job, error) {

	job, cronErr := cluster.Schedule(p.API, name, cluster.MakeWaitForRoundedInterval(interval), function)

	if cronErr != nil {
		return nil, errors.Wrap(cronErr, "failed to schedule background job")
	}

	return job, nil
}
