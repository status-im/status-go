package sociallinkssettings

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/protocol/identity"
)

func openTestDB(t *testing.T) (*SocialLinksSettings, func()) {
	db, stop, err := appdatabase.SetupTestSQLDB("settings-social-links-tests-")
	if err != nil {
		require.NoError(t, stop())
	}
	require.NoError(t, err)

	socialLinkSettings := NewSocialLinksSettings(db)
	if err != nil {
		require.NoError(t, stop())
	}
	require.NoError(t, err)

	return socialLinkSettings, func() {
		require.NoError(t, stop())
	}
}

func socialLinksWithDefaults(twitter, personalSite, github, youtube, discord, telegram string) identity.SocialLinks {
	return identity.SocialLinks{
		{
			Text: identity.TwitterID,
			URL:  twitter,
		},
		{
			Text: identity.PersonalSiteID,
			URL:  personalSite,
		},
		{
			Text: identity.GithubID,
			URL:  github,
		},
		{
			Text: identity.YoutubeID,
			URL:  youtube,
		},
		{
			Text: identity.DiscordID,
			URL:  discord,
		},
		{
			Text: identity.TelegramID,
			URL:  telegram,
		},
	}
}

func TestDatabase(t *testing.T) {
	socialLinkSettings, stop := openTestDB(t)
	defer stop()

	// fresh database should have default rows
	links, err := socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.True(t, links.Equals(socialLinksWithDefaults("", "", "", "", "", "")))

	// cleaning db should not remove default rows
	links = identity.SocialLinks{}
	err = socialLinkSettings.SetSocialLinks(&links)
	require.NoError(t, err)
	links, err = socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.True(t, links.Equals(identity.SocialLinks{}))

	// custom links
	links = identity.SocialLinks{
		{
			Text: identity.TwitterID,
			URL:  "Status_ico",
		},
		{
			Text: identity.TelegramID,
			URL:  "dummy.telegram",
		},
		{
			Text: "customLink",
			URL:  "customLink.com",
		},
	}
	err = socialLinkSettings.SetSocialLinks(&links)
	require.NoError(t, err)

	expected := identity.SocialLinks{
		{
			Text: identity.TwitterID,
			URL:  "Status_ico",
		},
		{
			Text: identity.TelegramID,
			URL:  "dummy.telegram",
		},
	}
	expected = append(expected, identity.SocialLink{Text: "customLink", URL: "customLink.com"})

	links, err = socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.True(t, links.Equals(expected))

	// cleaning database with defaults should remove custom links
	links = socialLinksWithDefaults("", "", "", "", "", "")
	err = socialLinkSettings.SetSocialLinks(&links)
	require.NoError(t, err)
	links, err = socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.True(t, links.Equals(socialLinksWithDefaults("", "", "", "", "", "")))
}
