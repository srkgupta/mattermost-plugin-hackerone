package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sync"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/cluster"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
)

const (
	hackeroneUsernameKey = "_hackeroneusername"
)

var hackeroneToUsernameMappingCallback func(string) string

func registerHackeroneToUsernameMappingCallback(callback func(string) string) {
	hackeroneToUsernameMappingCallback = callback
}

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	BotUserID string

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	httpClient http.Client

	scheduledJobs []*cluster.Job
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, world!")
}

func (p *Plugin) OnActivate() error {
	if p.API.GetConfig().ServiceSettings.SiteURL == nil {
		return errors.New("siteURL is not set. Please set a siteURL and restart the plugin")
	}
	client := pluginapi.NewClient(p.API, p.Driver)
	botID, err := client.Bot.EnsureBot(&model.Bot{
		Username:    "hackerone",
		DisplayName: "Hackerone",
		Description: "Created by the Hackerone plugin.",
	})
	if err != nil {
		return errors.Wrap(err, "failed to ensure hackerone bot")
	}
	p.BotUserID = botID

	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "couldn't get bundle path")
	}

	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile.png"))
	if err != nil {
		return errors.Wrap(err, "couldn't read profile image")
	}

	appErr := p.API.SetProfileImage(botID, profileImage)
	if appErr != nil {
		return errors.Wrap(appErr, "couldn't set profile image")
	}

	registerHackeroneToUsernameMappingCallback(p.getHackeroneToUsernameMapping)

	return nil
}

// getHackeroneToUsernameMapping maps a Hackerone username to the corresponding Mattermost username, if any.
func (p *Plugin) getHackeroneToUsernameMapping(hackeroneUsername string) string {
	user, _ := p.API.GetUser(p.getHackeroneToUserIDMapping(hackeroneUsername))
	if user == nil {
		return ""
	}

	return user.Username
}

func (p *Plugin) getHackeroneToUserIDMapping(hackeroneUsername string) string {
	userID, _ := p.API.KVGet(hackeroneUsername + hackeroneUsernameKey)
	return string(userID)
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
