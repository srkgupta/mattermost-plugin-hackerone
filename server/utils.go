package main

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
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

// HTTP Utils below

func (p *Plugin) respondAndLogErr(w http.ResponseWriter, code int, err error) {
	http.Error(w, err.Error(), code)
	p.API.LogError(err.Error())
}

func (p *Plugin) respondJSON(w http.ResponseWriter, obj interface{}) {
	data, err := json.Marshal(obj)
	if err != nil {
		p.respondAndLogErr(w, http.StatusInternalServerError, errors.WithMessage(err, "failed to marshal response"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		p.respondAndLogErr(w, http.StatusInternalServerError, errors.WithMessage(err, "failed to write response"))
		return
	}

	w.WriteHeader(http.StatusOK)
}
