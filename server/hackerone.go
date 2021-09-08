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
	p.API.LogDebug("Making HTTP request to Hackerone API:" + hackeroneApiUrl + url)
	req, err := http.NewRequest(method, hackeroneApiUrl+url, body)
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(p.getConfiguration().HackeroneApiIdentifier, p.getConfiguration().HackeroneApiKey)
	if err != nil {
		return nil, errors.Wrap(err, "bad request for url:"+url)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "connection problem for url:"+url)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, errors.New("non-ok status code for url:" + url)
	}
	return resp, err
}

type Activities struct {
	Activities []Activity `json:"data"`
	Meta       struct {
		MaxUpdatedAt string `json:"max_updated_at"`
	}
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
					Name           string
					Username       string
					ProfilePicture struct {
						Photo string `json:"260x260"`
					}
				}
			}
		}
	}
}

func (p *Plugin) fetchActivities(count string, last_updated_at string) (Activities, error) {
	activitiesEndpoint := "incremental/activities?handle=" + p.getConfiguration().HackeroneProgramHandle + "&page[size]=" + count

	if len(last_updated_at) > 1 {
		activitiesEndpoint += "&updated_at_after=" + last_updated_at
	}
	resp, err := p.doHTTPRequest(http.MethodGet, activitiesEndpoint, nil)
	errorMsg := "Something went wrong while getting the activities from Hackerone API: " + activitiesEndpoint
	if err != nil {
		p.API.LogWarn(errorMsg, "error", err.Error())
		return Activities{}, errors.Wrap(err, errorMsg)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK || resp.Body == nil {
		p.API.LogWarn(errorMsg, "error", err.Error())
		return Activities{}, errors.Wrap(err, errorMsg)
	}

	var response Activities
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&response); err != nil {
		p.API.LogWarn(errorMsg, "error", err.Error())
		return Activities{}, errors.Wrap(err, errorMsg)
	}
	return response, errors.Wrap(err, errorMsg)
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
					Name           string
					Username       string
					ProfilePicture struct {
						Photo string `json:"260x260"`
					}
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
