// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-server/v5/model"
)

func TestCondenseSiteURL(t *testing.T) {
	require.Equal(t, "", condenseSiteURL(""))
	require.Equal(t, "mattermost.com", condenseSiteURL("mattermost.com"))
	require.Equal(t, "mattermost.com", condenseSiteURL("mattermost.com/"))
	require.Equal(t, "chat.mattermost.com", condenseSiteURL("chat.mattermost.com"))
	require.Equal(t, "chat.mattermost.com", condenseSiteURL("chat.mattermost.com/"))
	require.Equal(t, "mattermost.com/subpath", condenseSiteURL("mattermost.com/subpath"))
	require.Equal(t, "mattermost.com/subpath", condenseSiteURL("mattermost.com/subpath/"))
	require.Equal(t, "chat.mattermost.com/subpath", condenseSiteURL("chat.mattermost.com/subpath"))
	require.Equal(t, "chat.mattermost.com/subpath", condenseSiteURL("chat.mattermost.com/subpath/"))

	require.Equal(t, "mattermost.com:8080", condenseSiteURL("http://mattermost.com:8080"))
	require.Equal(t, "mattermost.com:8080", condenseSiteURL("http://mattermost.com:8080/"))
	require.Equal(t, "chat.mattermost.com:8080", condenseSiteURL("http://chat.mattermost.com:8080"))
	require.Equal(t, "chat.mattermost.com:8080", condenseSiteURL("http://chat.mattermost.com:8080/"))
	require.Equal(t, "mattermost.com:8080/subpath", condenseSiteURL("http://mattermost.com:8080/subpath"))
	require.Equal(t, "mattermost.com:8080/subpath", condenseSiteURL("http://mattermost.com:8080/subpath/"))
	require.Equal(t, "chat.mattermost.com:8080/subpath", condenseSiteURL("http://chat.mattermost.com:8080/subpath"))
	require.Equal(t, "chat.mattermost.com:8080/subpath", condenseSiteURL("http://chat.mattermost.com:8080/subpath/"))
}

func TestSendInviteEmailRateLimits(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	th.BasicTeam.AllowedDomains = "common.com"
	_, err := th.App.UpdateTeam(th.BasicTeam)
	require.Nilf(t, err, "%v, Should update the team", err)

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.ServiceSettings.EnableEmailInvitations = true
	})

	emailList := make([]string, 22)
	for i := 0; i < 22; i++ {
		emailList[i] = "test-" + strconv.Itoa(i) + "@common.com"
	}
	err = th.App.InviteNewUsersToTeam(emailList, th.BasicTeam.Id, th.BasicUser.Id)
	require.NotNil(t, err)
	assert.Equal(t, "app.email.rate_limit_exceeded.app_error", err.Id)
	assert.Equal(t, http.StatusRequestEntityTooLarge, err.StatusCode)

	_, err = th.App.InviteNewUsersToTeamGracefully(emailList, th.BasicTeam.Id, th.BasicUser.Id)
	require.NotNil(t, err)
	assert.Equal(t, "app.email.rate_limit_exceeded.app_error", err.Id)
	assert.Equal(t, http.StatusRequestEntityTooLarge, err.StatusCode)
}

func TestSendAdminUpgradeRequestEmail(t *testing.T) {
	th := Setup(t).InitBasic()
	defer th.TearDown()

	th.App.Srv().SetLicense(model.NewTestLicense("cloud"))

	mockSubscription := &model.Subscription{
		ID:         "MySubscriptionID",
		CustomerID: "MyCustomer",
		ProductID:  "SomeProductId",
		AddOns:     []string{},
		StartAt:    1000000000,
		EndAt:      2000000000,
		CreateAt:   1000000000,
		Seats:      100,
		DNS:        "some.dns.server",
		IsPaidTier: "false",
	}

	th.App.UpdateConfig(func(cfg *model.Config) {
		*cfg.ExperimentalSettings.CloudUserLimit = 10
	})

	err := th.App.SendAdminUpgradeRequestEmail(th.BasicUser.Username, mockSubscription)
	require.Nil(t, err)

	err = th.App.SendAdminUpgradeRequestEmail(th.BasicUser2.Username, mockSubscription)
	require.Nil(t, err)

	// second attempt by the same user to send emails is blocked by rate limiter
	err = th.App.SendAdminUpgradeRequestEmail(th.BasicUser.Username, mockSubscription)
	require.NotNil(t, err)
	assert.Equal(t, err.Id, "app.email.rate_limit_exceeded.app_error")
}
