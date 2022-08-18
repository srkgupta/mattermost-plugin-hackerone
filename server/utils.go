package main

import (
	"time"

	"github.com/mattermost/mattermost-server/v6/model"
)

// Plugin utils
func (p *Plugin) sendPost(args *model.CommandArgs, message string, attachments []*model.SlackAttachment) *model.Post {
	channelPost := &model.Post{
		ChannelId: args.ChannelId,
		UserId:    p.BotUserID,
		Message:   message,
	}

	if attachments != nil {
		channelPost.AddProp("attachments", attachments)
	}

	if _, appErr := p.API.CreatePost(channelPost); appErr != nil {
		p.API.LogError("Unable to create post", "appError", appErr)
	}

	return channelPost
}

func (p *Plugin) sendEphemeralPost(args *model.CommandArgs, message string, attachments []*model.SlackAttachment) *model.Post {
	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: args.ChannelId,
		Message:   message,
	}

	if attachments != nil {
		post.AddProp("attachments", attachments)
	}

	return p.API.SendEphemeralPost(
		args.UserId,
		post,
	)
}

func (p *Plugin) sendPostByChannelId(channelId string, message string, attachments []*model.SlackAttachment) *model.Post {
	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: channelId,
		Message:   message,
	}

	if attachments != nil {
		post.AddProp("attachments", attachments)
	}

	if _, appErr := p.API.CreatePost(post); appErr != nil {
		p.API.LogError("Unable to create post", "appError", appErr)
	}

	return post
}

// Wrapper of p.sendEphemeralPost() to one-line the return statements in all executeCommand functions
func (p *Plugin) sendEphemeralResponse(args *model.CommandArgs, message string) *model.CommandResponse {
	p.sendEphemeralPost(args, message, nil)
	return &model.CommandResponse{}
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

func parseTime(input string) string {
	if len(input) > 5 {
		layout := "Mon Jan 02 2006 3:04 PM"
		t, _ := time.Parse(time.RFC3339, input)
		output := t.Format(layout)
		if output == "Mon Jan 01 0001 12:00 AM" {
			// If the input is not in the RFC3339 format, return input as is
			return input
		} else {
			return output
		}
	} else {
		return "-"
	}
}
