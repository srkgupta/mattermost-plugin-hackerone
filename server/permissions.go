package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
)

const (
	PermissionsKey = "permissions"
	permissionNote = "Note: By default, all system administrators can run the `/hackerone` commands."
)

type Permissions struct {
	Permissions []string
}

func (p *Plugin) AllowPermission(userID string) error {
	perms, err := p.GetPermissions()
	if err != nil {
		return errors.Wrap(err, "could not get permissions")
	}
	exists := false
	for _, v := range perms {
		if v == userID {
			exists = true
			break
		}
	}

	if !exists {
		perms = append(perms, userID)
	}

	err = p.StorePermissions(perms)
	if err != nil {
		return errors.Wrap(err, "could not store permissions")
	}

	return nil
}

func (p *Plugin) RemovePermission(userID string) error {
	newPerms := []string{}
	perms, err := p.GetPermissions()
	if err != nil {
		return errors.Wrap(err, "could not get permissions")
	}

	for _, v := range perms {
		if v != userID {
			newPerms = append(newPerms, v)
		}
	}

	err = p.StorePermissions(newPerms)
	if err != nil {
		return errors.Wrap(err, "could not store permissions")
	}

	return nil
}

func (p *Plugin) GetPermissions() ([]string, error) {
	permissions := []string{}
	value, appErr := p.API.KVGet(PermissionsKey)
	if appErr != nil {
		return nil, errors.Wrap(appErr, "could not get permissions from KVStore")
	}

	if value == nil {
		return []string{}, nil
	}

	err := json.NewDecoder(bytes.NewReader(value)).Decode(&permissions)
	if err != nil {
		return nil, errors.Wrap(err, "could not properly decode permissions key")
	}

	return permissions, nil
}

func (p *Plugin) StorePermissions(perm []string) error {
	b, err := json.Marshal(perm)
	if err != nil {
		return errors.Wrap(err, "error while converting permissions to json")
	}

	if appErr := p.API.KVSet(PermissionsKey, b); appErr != nil {
		return errors.Wrap(appErr, "could not store permissions in KV store")
	}

	return nil
}

func (p *Plugin) IsAdmin(userID string) (bool, error) {
	user, appErr := p.API.GetUser(userID)
	if appErr != nil {
		return false, fmt.Errorf(
			"failed to obtain information about user `%s`: %w", userID, appErr)
	}
	if strings.Contains(user.Roles, "system_admin") {
		p.API.LogDebug(
			fmt.Sprintf("UserID `%s` is authorized on basis of the sysadmin role membership", userID))
		return true, nil
	}
	return false, nil
}

func (p *Plugin) IsAuthorized(userID string) (bool, error) {
	isAdmin, appErr := p.IsAdmin(userID)
	if appErr != nil {
		return false, appErr
	}

	if isAdmin {
		return true, nil
	}

	perms, err := p.GetPermissions()
	if err != nil {
		return false, fmt.Errorf(
			"failed to obtain information about user `%s`: %w", userID, err)
	}

	for _, u := range perms {
		if u == userID {
			p.API.LogDebug(
				fmt.Sprintf("UserID `%s` is authorized on basis of plugin allowed users list", userID))
			return true, nil
		}
	}

	return false, nil
}

func (p *Plugin) executePermissions(args *model.CommandArgs, split []string) (*model.CommandResponse, *model.AppError) {
	if 0 >= len(split) {
		msg := "Invalid permissions command. Available commands are 'list', 'add' and 'delete'."
		return p.sendEphemeralResponse(args, msg), nil
	}

	command := split[0]

	switch {
	case command == "list":
		return p.handlePermissionsList(args)
	case command == "add":
		if len(split) < 2 {
			msg := "Please specify the user who needs to be whitelisted to run the hackerone slash command. Run the command, eg: `/hackerone permissions add @user1` to whitelist a specific user."
			return p.sendEphemeralResponse(args, msg), nil
		} else {
			return p.handlePermissionsAdd(args, split[1])
		}
	case command == "delete":
		if len(split) < 2 {
			msg := "Please specify the user who needs to be removed from running the /hackerone slash command. Run the command, eg: `/hackerone permissions delete @user1` to remove a specific user."
			return p.sendEphemeralResponse(args, msg), nil
		} else {
			return p.handlePermissionsDelete(args, split[1])
		}
	default:
		msg := "Unknown subcommand for permissions command. Available commands are 'list', 'add' and 'delete'."
		return p.sendEphemeralResponse(args, msg), nil
	}
}

func (p *Plugin) handlePermissionsAdd(args *model.CommandArgs, username string) (*model.CommandResponse, *model.AppError) {
	username = strings.TrimPrefix(username, "@")
	user, userErr := p.API.GetUserByUsername(username)
	if userErr != nil {
		p.API.LogError(
			fmt.Sprintf("Something went wrong while adding permissions. Error: %s", userErr.Error()))
		msg := "Something went wrong while adding permissions. Please check the username (or) check the server logs"
		return p.sendEphemeralResponse(args, msg), nil
	}

	err := p.AllowPermission(user.Id)
	if err != nil {
		p.API.LogError(
			fmt.Sprintf("Something went wrong while adding permissions. Error: %s", err.Error()))
		msg := "Something went wrong while adding permissions. Please check the username (or) check the server logs"
		return p.sendEphemeralResponse(args, msg), nil
	}
	msg := fmt.Sprintf("User `%s` was successfully whitelisted to run `/hackerone` commands. %s", user.Username, permissionNote)
	return p.sendEphemeralResponse(args, msg), nil
}

func (p *Plugin) handlePermissionsDelete(args *model.CommandArgs, username string) (*model.CommandResponse, *model.AppError) {
	username = strings.TrimPrefix(username, "@")
	user, userErr := p.API.GetUserByUsername(username)
	if userErr != nil {
		p.API.LogError(
			fmt.Sprintf("Something went wrong while removing permissions. Error: %s", userErr.Error()))
		msg := "Something went wrong while removing permissions. Please check the username (or) check the server logs"
		return p.sendEphemeralResponse(args, msg), nil
	}

	err := p.RemovePermission(user.Id)
	if err != nil {
		p.API.LogError(
			fmt.Sprintf("Something went wrong while removing permissions. Error: %s", err.Error()))
		msg := "Something went wrong while removing permissions. Please check the username (or) check the server logs"
		return p.sendEphemeralResponse(args, msg), nil
	}
	msg := fmt.Sprintf("User `%s` is now removed from the list of users allowed to run `/hackerone` commands. %s", user.Username, permissionNote)
	return p.sendEphemeralResponse(args, msg), nil
}

func (p *Plugin) handlePermissionsList(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	perms, err := p.GetPermissions()
	msg := ""
	if err != nil {
		p.API.LogError(
			fmt.Sprintf("Something went wrong while listing permissions. Error: %s", err.Error()))
		msg := "Something went wrong while listing permissions. Please check the server logs"
		return p.sendEphemeralResponse(args, msg), nil
	}

	if len(perms) == 0 {
		msg = "Currently there are no users whitelisted to run `/hackerone` commands. " + permissionNote
	} else {
		msg = "Users whitelisted to run `/hackerone` slash commands:\n"
		for i, u := range perms {
			user, appErr := p.API.GetUser(u)
			if appErr != nil {
				p.API.LogWarn(
					fmt.Sprintf("User was whitelisted but user info for userID `%s` could not be obtained. %s", u, appErr.Error()))

			} else {
				msg += fmt.Sprintf("%d. @%s\n", i+1, user.Username)
			}
		}
	}
	return p.sendEphemeralResponse(args, msg), nil
}
