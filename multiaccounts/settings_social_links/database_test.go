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

func profileSocialLinks() identity.SocialLinks {
	return identity.SocialLinks{
		{
			Text: identity.TwitterID,
			URL:  "https://twitter.com/ethstatus",
		},
		{
			Text: identity.TwitterID,
			URL:  "https://twitter.com/StatusIMBlog",
		},
		{
			Text: identity.TelegramID,
			URL:  "dummy.telegram",
		},
		{
			Text: identity.YoutubeID,
			URL:  "https://www.youtube.com/@Statusim",
		},
		{
			Text: identity.YoutubeID,
			URL:  "https://www.youtube.com/@EthereumProtocol",
		},
		{
			Text: "customLink",
			URL:  "customLink.com",
		},
	}
}

func TestProfileSocialLinksSaveAndGet(t *testing.T) {
	socialLinkSettings, stop := openTestDB(t)
	defer stop()

	// db is empty at the beginning
	links, err := socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, links, 0)

	clock := uint64(1)
	// add profile social links with new clock
	profileSocialLinks1 := profileSocialLinks()[:2]
	err = socialLinkSettings.AddOrReplaceSocialLinksIfNewer(profileSocialLinks1, clock)
	require.NoError(t, err)

	// check social links
	links, err = socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, links, len(profileSocialLinks1))
	require.True(t, profileSocialLinks1.Equal(links))

	oldClock := uint64(0)
	// delete add profile social links with old clock
	profileSocialLinks2 := profileSocialLinks()
	err = socialLinkSettings.AddOrReplaceSocialLinksIfNewer(profileSocialLinks2, oldClock)
	require.NoError(t, err)

	// check social links
	links, err = socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, links, len(profileSocialLinks1))
	require.True(t, profileSocialLinks1.Equal(links))

	// check clock
	dbClock, err := socialLinkSettings.GetSocialLinksClock()
	require.NoError(t, err)
	require.Equal(t, clock, dbClock)
}

func TestProfileSocialLinksUpdate(t *testing.T) {
	socialLinkSettings, stop := openTestDB(t)
	defer stop()

	// db is empty at the beginning
	links, err := socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, links, 0)

	clock := uint64(1)
	// add profile social links
	profileSocialLinks := profileSocialLinks()
	err = socialLinkSettings.AddOrReplaceSocialLinksIfNewer(profileSocialLinks, clock)
	require.NoError(t, err)

	clock = 2
	// test social link update
	updateLinkAtIndex := 2
	profileSocialLinks[updateLinkAtIndex].Text = identity.GithubID
	profileSocialLinks[updateLinkAtIndex].URL = "https://github.com/status-im"

	err = socialLinkSettings.AddOrReplaceSocialLinksIfNewer(profileSocialLinks, clock)
	require.NoError(t, err)

	// check social links
	links, err = socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, links, len(profileSocialLinks))
	require.True(t, profileSocialLinks.Equal(links))

	// check clock
	dbClock, err := socialLinkSettings.GetSocialLinksClock()
	require.NoError(t, err)
	require.Equal(t, clock, dbClock)
}

func TestProfileSocialLinksDelete(t *testing.T) {
	socialLinkSettings, stop := openTestDB(t)
	defer stop()

	// db is empty at the beginning
	links, err := socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, links, 0)

	clock := uint64(1)
	// add profile social links
	profileSocialLinks := profileSocialLinks()
	totalLinks := len(profileSocialLinks)
	err = socialLinkSettings.AddOrReplaceSocialLinksIfNewer(profileSocialLinks, clock)
	require.NoError(t, err)

	// check
	links, err = socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, links, totalLinks)
	require.True(t, profileSocialLinks.Equal(links))

	// prepare new links to save
	removeLinkAtIndex := 2
	removedLink := profileSocialLinks[removeLinkAtIndex]
	profileSocialLinks = append(profileSocialLinks[:removeLinkAtIndex], profileSocialLinks[removeLinkAtIndex+1:]...)

	oldClock := uint64(0)
	// test delete with old clock
	err = socialLinkSettings.AddOrReplaceSocialLinksIfNewer(profileSocialLinks, oldClock)
	require.NoError(t, err)

	// check
	links, err = socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, links, totalLinks)
	require.True(t, links.Contains(removedLink))

	clock = 2
	// test delete link new clock
	err = socialLinkSettings.AddOrReplaceSocialLinksIfNewer(profileSocialLinks, clock)
	require.NoError(t, err)

	// check social links
	links, err = socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, links, totalLinks-1)
	require.True(t, profileSocialLinks.Equal(links))
	require.False(t, links.Contains(removedLink))

	// check clock
	dbClock, err := socialLinkSettings.GetSocialLinksClock()
	require.NoError(t, err)
	require.Equal(t, clock, dbClock)
}

func TestProfileSocialLinksReorder(t *testing.T) {
	socialLinkSettings, stop := openTestDB(t)
	defer stop()

	// db is empty at the beginning
	links, err := socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, links, 0)

	clock := uint64(1)
	// add profile social links
	profileSocialLinks := profileSocialLinks()
	totalLinks := len(profileSocialLinks)
	err = socialLinkSettings.AddOrReplaceSocialLinksIfNewer(profileSocialLinks, clock)
	require.NoError(t, err)

	// check social links
	links, err = socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, links, len(profileSocialLinks))
	require.True(t, profileSocialLinks.Equal(links))

	var randomLinksOrder identity.SocialLinks
	for i := len(profileSocialLinks) - 1; i >= 3; i-- {
		randomLinksOrder = append(randomLinksOrder, profileSocialLinks[i])
	}
	randomLinksOrder = append(randomLinksOrder, profileSocialLinks[:3]...)

	clock = 2
	// test reorder links
	err = socialLinkSettings.AddOrReplaceSocialLinksIfNewer(randomLinksOrder, clock)
	require.NoError(t, err)

	// check social links
	links, err = socialLinkSettings.GetSocialLinks()
	require.NoError(t, err)
	require.Len(t, randomLinksOrder, totalLinks)
	require.True(t, randomLinksOrder.Equal(links))

	// check clock
	dbClock, err := socialLinkSettings.GetSocialLinksClock()
	require.NoError(t, err)
	require.Equal(t, clock, dbClock)
}
