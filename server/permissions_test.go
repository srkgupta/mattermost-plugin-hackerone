package main

import (
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func appError() *model.AppError {
	return model.NewAppError(mock.Anything, mock.Anything, map[string]interface{}{}, mock.Anything, 1)
}
func Test_GetPermissions(t *testing.T) {
	p := &Plugin{}
	t.Run("Error in store", func(t *testing.T) {
		mockPluginAPI := &plugintest.API{}
		mockPluginAPI.On("KVGet", PermissionsKey).Return(nil, appError())
		p.SetAPI(mockPluginAPI)
		_, err := p.GetPermissions()
		assert.Error(t, err)
		assert.Equal(t, "could not get permissions from KVStore: mock.Anything: mock.Anything, mock.Anything", err.Error())
	})
	t.Run("Empty value in store", func(t *testing.T) {
		mockPluginAPI := &plugintest.API{}
		mockPluginAPI.On("KVGet", PermissionsKey).Return(nil, nil)
		p.SetAPI(mockPluginAPI)
		val, err := p.GetPermissions()
		assert.NoError(t, err)
		assert.Equal(t, []string{}, val)
	})
	t.Run("Valid value in store", func(t *testing.T) {
		mockPluginAPI := &plugintest.API{}
		arr := []string{"mock1", "mock2"}
		b, _ := json.Marshal(arr)
		mockPluginAPI.On("KVGet", PermissionsKey).Return(b, nil)
		p.SetAPI(mockPluginAPI)
		val, err := p.GetPermissions()
		assert.NoError(t, err)
		assert.Equal(t, arr, val)
	})
}

func Test_StorePermissions(t *testing.T) {
	p := &Plugin{}
	t.Run("Empty", func(t *testing.T) {
		arr1 := []string{}
		mockPluginAPI := &plugintest.API{}
		mockPluginAPI.On("KVSet", PermissionsKey, mock.AnythingOfType("[]uint8")).Return(nil)
		p.SetAPI(mockPluginAPI)
		err := p.StorePermissions(arr1)
		assert.NoError(t, err)
	})
	t.Run("Valid", func(t *testing.T) {
		arr1 := []string{"mock1", "mock2"}
		mockPluginAPI := &plugintest.API{}
		mockPluginAPI.On("KVSet", PermissionsKey, mock.AnythingOfType("[]uint8")).Return(nil)
		p.SetAPI(mockPluginAPI)
		err := p.StorePermissions(arr1)
		assert.NoError(t, err)
	})

}
