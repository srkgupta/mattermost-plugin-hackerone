package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_GetActivityLastKey(t *testing.T) {
	p := &Plugin{}
	t.Run("Error in store", func(t *testing.T) {
		mockPluginAPI := &plugintest.API{}
		mockPluginAPI.On("KVGet", ActivityLastKey).Return(nil, model.NewAppError(mock.Anything, mock.Anything, map[string]interface{}{}, mock.Anything, 1))
		p.SetAPI(mockPluginAPI)
		_, err := p.GetActivityLastKey()
		assert.Error(t, err)
		assert.Equal(t, "could not get activity last key from KVStore: mock.Anything: mock.Anything, mock.Anything", err.Error())
	})
	t.Run("Empty value in store", func(t *testing.T) {
		mockPluginAPI := &plugintest.API{}
		mockPluginAPI.On("KVGet", ActivityLastKey).Return(nil, nil)
		p.SetAPI(mockPluginAPI)
		val, err := p.GetActivityLastKey()
		assert.NoError(t, err)
		assert.Equal(t, "1970-01-01T00:00:00Z", val)
	})
	t.Run("Valid value in store", func(t *testing.T) {
		mockPluginAPI := &plugintest.API{}
		mockPluginAPI.On("KVGet", ActivityLastKey).Return([]byte("2001-11-21T00:00:00Z"), nil)
		p.SetAPI(mockPluginAPI)
		val, err := p.GetActivityLastKey()
		assert.NoError(t, err)
		assert.Equal(t, "2001-11-21T00:00:00Z", val)
	})
}

func Test_StoreActivityLastKey(t *testing.T) {
	p := &Plugin{}
	t.Run("Invalid Format", func(t *testing.T) {
		err := p.StoreActivityLastKey("test")
		assert.Error(t, err)
		assert.Equal(t, "invalid format of activity last key. It should be in the format: 2017-02-02T04:05:06.000Z: parsing time \"test\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"test\" as \"2006\"", err.Error())
	})
	t.Run("Invalid Format2", func(t *testing.T) {
		err := p.StoreActivityLastKey("2006-01-02T15:04:05Z07:00")
		assert.Error(t, err)
		assert.Equal(t, "invalid format of activity last key. It should be in the format: 2017-02-02T04:05:06.000Z: parsing time \"2006-01-02T15:04:05Z07:00\": extra text: \"07:00\"", err.Error())
	})
	t.Run("Valid Format", func(t *testing.T) {
		mockPluginAPI := &plugintest.API{}
		mockPluginAPI.On("KVSet", ActivityLastKey, mock.AnythingOfType("[]uint8")).Return(nil)
		p.SetAPI(mockPluginAPI)
		err := p.StoreActivityLastKey("2006-01-02T15:04:05Z")
		assert.NoError(t, err)
	})
}
