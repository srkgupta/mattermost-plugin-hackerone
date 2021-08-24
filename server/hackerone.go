package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

const (
	hackeroneApiUrl = "https://api.hackerone.com/v1/"
)

func (p *Plugin) doHTTPRequest(method string, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, hackeroneApiUrl+url, body)
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(p.getConfiguration().HackeroneApiIdentifier, p.getConfiguration().HackeroneApiKey)
	if err != nil {
		return nil, errors.Wrap(err, "bad request")
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "connection problem")
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, errors.New("non-ok status code")
	}
	return resp, err
}

type Activities struct {
	Activities []Activity `json:"data"`
}

type Activity struct {
	Attributes struct {
		ReportID  string `json:"report_id"`
		CreatedAt string `json:"created_at"`
		Internal  bool   `json:"internal"`
		Message   string `json:"message"`
	}
	ActivityType  string `json:"type"`
	Relationships struct {
		Actor struct {
			Data struct {
				Attributes struct {
					Name     string
					Username string
				}
			}
		}
	}
}

func (p *Plugin) fetchActivities(count string) ([]Activity, error) {
	activitiesEndpoint := "incremental/activities?handle=" + p.getConfiguration().HackeroneProgramHandle + "&page[size]=" + count

	resp, err := p.doHTTPRequest(http.MethodGet, activitiesEndpoint, nil)
	if err != nil {
		p.API.LogWarn("Something went wrong while getting the activities from Hackerone API", "error", err.Error())
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK || resp.Body == nil {
		p.API.LogWarn("Something went wrong while getting the activities from Hackerone API", "error", err.Error())
		return nil, err
	}

	var response Activities
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&response); err != nil {
		p.API.LogWarn("Something went wrong while getting the activities from Hackerone API", "error", err.Error())
		return nil, err
	}
	if len(response.Activities) < 1 {
		p.API.LogWarn("Something went wrong while getting the activities from Hackerone API", "error", err.Error())
		return nil, err
	}
	return response.Activities, err
}

type Reports struct {
	Reports []Report `json:"data"`
}

type ReportResponse struct {
	Report Report `json:"data"`
}

type Report struct {
	Attributes struct {
		Title           string `json:"title"`
		State           string `json:"state"`
		CreatedAt       string `json:"created_at"`
		TriagedAt       string `json:"triaged_at"`
		ClosedAt        string `json:"closed_at"`
		BountyAwardedAt string `json:"bounty_awarded_at"`
		DisclosedAt     string `json:"disclosed_at"`
		SwagAt          string `json:"swag_awarded_at"`
		Info            string `json:"vulnerability_information"`
	}
	Id            string `json:"id"`
	Relationships struct {
		Reporter struct {
			Data struct {
				Attributes struct {
					Name     string
					Username string
				}
			}
		}
	}
}

func (p *Plugin) fetchReports(filters map[string]string) ([]Report, error) {
	program := p.getConfiguration().HackeroneProgramHandle
	pageSize := 100
	reportsEndpoint := fmt.Sprintf("reports?filter[program][]=%s&page[size]=%d", program, pageSize)
	for key, value := range filters {
		if key == "state" || key == "severity" {
			reportsEndpoint += fmt.Sprintf("&filter[%s][]=%s", key, value)
		} else {
			reportsEndpoint += fmt.Sprintf("&filter[%s]=%s", key, value)
		}
	}

	resp, err := p.doHTTPRequest(http.MethodGet, reportsEndpoint, nil)
	if err != nil {
		p.API.LogWarn("Something went wrong while getting the reports from Hackerone API", "error", err.Error())
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK || resp.Body == nil {
		p.API.LogWarn("Something went wrong while getting the reports from Hackerone API", "error", err.Error())
		return nil, err
	}
	var response Reports
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&response); err != nil {
		p.API.LogWarn("Something went wrong while getting the reports from Hackerone API", "error", err.Error())
		return nil, err
	}
	return response.Reports, err
}

type Stats struct {
	NewCount           int
	TriagedCount       int
	PendingBountyCount int
}

func (p *Plugin) fetchStats() (Stats, error) {
	var stats Stats
	stats.NewCount = 2
	stats.TriagedCount = 4
	stats.PendingBountyCount = 10
	return stats, nil
}

func (p *Plugin) fetchReport(reportId string) (Report, error) {
	reportsEndpoint := "reports/" + reportId
	resp, err := p.doHTTPRequest(http.MethodGet, reportsEndpoint, nil)
	if err != nil {
		p.API.LogWarn("Something went wrong while getting the report from Hackerone API", "error", err.Error())
		return Report{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK || resp.Body == nil {
		p.API.LogWarn("Something went wrong while getting the report from Hackerone API", "error", err.Error())
		return Report{}, err
	}
	var response ReportResponse
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&response); err != nil {
		p.API.LogWarn("Something went wrong while getting the report from Hackerone API", "error", err.Error())
		return Report{}, err
	}
	return response.Report, err
}
