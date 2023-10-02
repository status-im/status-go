package common

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

const expectedJPEG = "data:image/jpeg;base64,/9j/2wBDAAMCAgICAgMCAgIDAwMDBAYEBAQEBAgGBgUGCQgKCgkICQkKDA8MCgsOCwkJDRENDg8QEBEQCgwSExIQEw8QEBD/yQALCAABAAEBAREA/8wABgAQEAX/2gAIAQEAAD8A0s8g/9k="
const expectedAAC = "data:audio/aac;base64,//FQgBw//NoATGF2YzUyLjcwLjAAQniptokphEFCg5qs1v9fn48+qz1rfWNhwvz+CqB5dipmq3T2PlT1Ld6sPj+19fUt1C3NKV0KowiqohZVCrdf19WMatvV3YbIvAuy/q2RafA8UiZPmZY7DdmHZtP9ri25kedWSiMKQRt79ttlod55LkuX7/f7/f7/f7/YGBgYGBgYGBgYGBgYGBgYGBgYGBgYGBgYGBgYGBgYGBgYGBgYGBgYHNqo8g5qs1v9fn48+qz1rfWNhwvz+CqAAAAAAAAAAAAAAAAAAAAAAABw//FQgCNf/CFXbUZfDKFRgsYlKDegtXJH9eLkT54uRM1ckDYDcXRzZGF6Kz5Yps5fTeLY6w7gclwly+0PJL3udY3PyekTFI65bdniF3OjvHeafzZfWTs0qRMSkdll1sbb4SNT5e8vX98ytot6jEZ0NhJi2pBVP/tKV2JMyo36n9uxR2tKR+FoLCsP4SVi49kmvaSCWm5bQD96OmVQA9Q40bqnOa7rT8j9N0TlK991XdcenGTLbyS6eUnN2U1ckf14uRPni5EzVyQAAAAAAAAAAx6Q1flBp+KH2LhgH2Xx+14QB2/jcizm6ngck4vB9DoH9/Vcb7E8Dy+D/1ii1pSPwsUUUXCSsXHsk17SBfKwn2uHr6QAAAAAAAHA//FQgBt//CF3VO1KFCFWcd/r04m+O0758/tXHUlvaqEK9lvhUZXEZMXKMV/LQ6B3/mOl/Mrfs6jpD2b7f+n4yt+tm2x5ZmnpD++dZo/V9VgblI3OW/s1b8qt0h1RBiIRIIYIYQIBeCM8yy7etkwt1JAajRSoZGwwNZ07TTFTyMR1mTUVVUTW97vaDaHU5DV1snBf0mN4fraa+rf/vpdZ8FxqatGjNxPh35UuVfpNqc48W4nZ6rOO/16cTfHad8+f2rjqS3tVAAAAAAAAAAAAAAAAAAAAAAAAAAAO//FQgBm//CEXVPU+GiFsPr7x6+N6v+m+q511I4SgtYVyoyWjcMWMxkaxxDGSx1qVcarjDESt8zLQehx/lkil/GrHBy/NfJcHek0XtfanZJLHNXO2rUnFklPAlQSBS4l0pIoXIfORcXx0UYj1nTsSe1/0wXDkkFCfxWHtqRayOmWm3oS6JGdnZdtjesjByefiS8dLW1tVVVC58ijoxN3gmGFYj07+YJ6eth9fePXxvV/031XOupHCUAAAAAAAAAAAAAAAAAAAAAAAAAAA4P/xUIAcf/whN1T9NsMOEK5rxxxxXnid+f0/Ia195vi6oGH1ZVr6kjqScdSF9lt3qXH+Lxf0fo/Oe53r99IUPzybv/YWGZ7Vgk31MGw+DMp05+3y9fPERUTHlt1c9sUyoqCaD5bdXVz2wkG0hnpDmFy8r0fr3VBn/C7Rmg+L0/45EWfdocGq3HQ1uRro0GJK+vsvo837NR82s01l/n97rsWn7RYNBM3WRcDY3cJKosqMJhgdHtj9yflthd65rxxxxXnid+f0/Ia195vi6oAAAAAAAAAAAAAAAAAAAAAAAAAAAABw"

func TestPrepareContentImage(t *testing.T) {
	file, err := os.Open("../../_assets/tests/test.jpg")
	require.NoError(t, err)
	defer file.Close()

	payload, err := ioutil.ReadAll(file)
	require.NoError(t, err)

	message := NewMessage()
	message.ContentType = protobuf.ChatMessage_IMAGE
	image := protobuf.ImageMessage{
		Payload: payload,
		Type:    protobuf.ImageType_JPEG,
	}
	message.Payload = &protobuf.ChatMessage_Image{Image: &image}

	require.NoError(t, message.PrepareContent(""))
	require.Equal(t, expectedJPEG, message.Base64Image)
}

func TestPrepareContentAudio(t *testing.T) {
	file, err := os.Open("../../_assets/tests/test.aac")
	require.NoError(t, err)
	defer file.Close()

	payload, err := ioutil.ReadAll(file)
	require.NoError(t, err)

	message := NewMessage()
	message.ContentType = protobuf.ChatMessage_AUDIO
	audio := protobuf.AudioMessage{
		Payload: payload,
		Type:    protobuf.AudioMessage_AAC,
	}
	message.Payload = &protobuf.ChatMessage_Audio{Audio: &audio}

	require.NoError(t, message.PrepareContent(""))
	require.Equal(t, expectedAAC, message.Base64Audio)
}

func TestGetAudioMessageMIME(t *testing.T) {
	aac := &protobuf.AudioMessage{Type: protobuf.AudioMessage_AAC}
	mime, err := getAudioMessageMIME(aac)
	require.NoError(t, err)
	require.Equal(t, "aac", mime)

	amr := &protobuf.AudioMessage{Type: protobuf.AudioMessage_AMR}
	mime, err = getAudioMessageMIME(amr)
	require.NoError(t, err)
	require.Equal(t, "amr", mime)
}

func TestPrepareContentMentions(t *testing.T) {
	message := NewMessage()
	pk1, err := crypto.GenerateKey()
	require.NoError(t, err)
	pk1String := types.EncodeHex(crypto.FromECDSAPub(&pk1.PublicKey))

	pk2, err := crypto.GenerateKey()
	require.NoError(t, err)
	pk2String := types.EncodeHex(crypto.FromECDSAPub(&pk2.PublicKey))

	message.Text = "hey @" + pk1String + " @" + pk2String

	require.NoError(t, message.PrepareContent(pk2String))
	require.Len(t, message.Mentions, 2)
	require.Equal(t, message.Mentions[0], pk1String)
	require.Equal(t, message.Mentions[1], pk2String)
	require.True(t, message.Mentioned)
}

func TestPrepareContentLinks(t *testing.T) {
	message := NewMessage()

	link1 := "https://github.com/status-im/status-mobile"
	link2 := "https://www.youtube.com/watch?v=6RYO8KCY6YE"

	message.Text = "Just look at that repo " + link1 + " . And watch this video - " + link2

	require.NoError(t, message.PrepareContent(""))
	require.Len(t, message.Links, 2)
	require.Equal(t, message.Links[0], link1)
	require.Equal(t, message.Links[1], link2)
}

func TestPrepareSimplifiedText(t *testing.T) {
	canonicalName1 := "canonical-name-1"
	canonicalName2 := "canonical-name-2"

	message := NewMessage()
	pk1, err := crypto.GenerateKey()
	require.NoError(t, err)
	pk1String := types.EncodeHex(crypto.FromECDSAPub(&pk1.PublicKey))

	pk2, err := crypto.GenerateKey()
	require.NoError(t, err)
	pk2String := types.EncodeHex(crypto.FromECDSAPub(&pk2.PublicKey))

	message.Text = "hey @" + pk1String + " @" + pk2String

	require.NoError(t, message.PrepareContent(""))
	require.Len(t, message.Mentions, 2)
	require.Equal(t, message.Mentions[0], pk1String)
	require.Equal(t, message.Mentions[1], pk2String)

	canonicalNames := make(map[string]string)
	canonicalNames[pk1String] = canonicalName1
	canonicalNames[pk2String] = canonicalName2

	simplifiedText, err := message.GetSimplifiedText("", canonicalNames)
	require.NoError(t, err)
	require.Equal(t, "hey "+canonicalName1+" "+canonicalName2, simplifiedText)
}

func TestConvertLinkPreviewsToProto(t *testing.T) {
	msg := Message{
		LinkPreviews: []LinkPreview{
			{
				Type:        protobuf.UnfurledLink_LINK,
				Description: "GitHub is where people build software.",
				Hostname:    "github.com",
				Title:       "Build software better, together",
				URL:         "https://github.com",
				Thumbnail: LinkPreviewThumbnail{
					Width:   100,
					Height:  200,
					URL:     "http://localhost:9999",
					DataURI: "data:image/png;base64,iVBORw0KGgoAAAANSUg=",
				},
			},
		},
	}

	unfurledLinks, err := msg.ConvertLinkPreviewsToProto()
	require.NoError(t, err)
	require.Len(t, unfurledLinks, 1)

	l := unfurledLinks[0]
	validPreview := msg.LinkPreviews[0]
	require.Equal(t, validPreview.Type, l.Type)
	require.Equal(t, validPreview.Description, l.Description)
	require.Equal(t, validPreview.Title, l.Title)
	require.Equal(t, uint32(validPreview.Thumbnail.Width), l.ThumbnailWidth)
	require.Equal(t, uint32(validPreview.Thumbnail.Height), l.ThumbnailHeight)

	expectedPayload, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUg=")
	require.NoError(t, err)
	require.Equal(t, expectedPayload, l.ThumbnailPayload)

	// Test any invalid link preview causes an early return.
	invalidPreview := validPreview
	invalidPreview.Title = ""
	msg.LinkPreviews = []LinkPreview{invalidPreview}
	_, err = msg.ConvertLinkPreviewsToProto()
	require.ErrorContains(t, err, "invalid link preview, url='https://github.com'")

	// Test invalid data URI invalidates a preview.
	invalidPreview = validPreview
	invalidPreview.Thumbnail.DataURI = "data:hello/png,iVBOR"
	msg.LinkPreviews = []LinkPreview{invalidPreview}
	_, err = msg.ConvertLinkPreviewsToProto()
	require.ErrorContains(t, err, "could not get data URI payload, url='https://github.com': wrong uri format")

	// Test thumbnail is optional.
	somePreview := validPreview
	somePreview.Thumbnail.DataURI = ""
	somePreview.Thumbnail.Width = 0
	somePreview.Thumbnail.Height = 0
	msg.LinkPreviews = []LinkPreview{somePreview}
	unfurledLinks, err = msg.ConvertLinkPreviewsToProto()
	require.NoError(t, err)
	require.Len(t, unfurledLinks, 1)
	require.Nil(t, unfurledLinks[0].ThumbnailPayload)
}

func TestConvertFromProtoToLinkPreviews(t *testing.T) {

	thumbnailPayload, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUg=")
	require.NoError(t, err)

	l := &protobuf.UnfurledLink{
		Description:      "GitHub is where people build software.",
		Title:            "Build software better, together",
		Type:             protobuf.UnfurledLink_LINK,
		Url:              "https://github.com",
		ThumbnailPayload: thumbnailPayload,
		ThumbnailWidth:   100,
		ThumbnailHeight:  200,
	}
	msg := Message{
		ID: "42",
		ChatMessage: &protobuf.ChatMessage{
			UnfurledLinks: []*protobuf.UnfurledLink{l},
		},
	}

	urlMaker := func(msgID string, linkURL string) string {
		return "https://localhost:6666/" + msgID + "-" + linkURL
	}

	previews := msg.ConvertFromProtoToLinkPreviews(urlMaker)
	require.Len(t, previews, 1)
	p := previews[0]
	require.Equal(t, l.Type, p.Type)
	require.Equal(t, "github.com", p.Hostname)
	require.Equal(t, l.Description, p.Description)
	require.Equal(t, l.Url, p.URL)
	require.Equal(t, int(l.ThumbnailHeight), p.Thumbnail.Height)
	require.Equal(t, int(l.ThumbnailWidth), p.Thumbnail.Width)
	// Important, don't build up a data URI because the thumbnail should be
	// fetched from the media server.
	require.Equal(t, "", p.Thumbnail.DataURI)
	require.Equal(t, "https://localhost:6666/42-https://github.com", p.Thumbnail.URL)

	// Test when the URL is not parseable by url.Parse.
	l.Url = "postgres://user:abc{DEf1=ghi@example.com:5432/db?sslmode=require"
	msg.ChatMessage.UnfurledLinks = []*protobuf.UnfurledLink{l}
	previews = msg.ConvertFromProtoToLinkPreviews(urlMaker)
	require.Len(t, previews, 1)
	p = previews[0]
	require.Equal(t, l.Url, p.Hostname)

	// Test when there's no thumbnail payload.
	l = &protobuf.UnfurledLink{
		Description: "GitHub is where people build software.",
		Title:       "Build software better, together",
		Url:         "https://github.com",
	}
	msg.ChatMessage.UnfurledLinks = []*protobuf.UnfurledLink{l}
	previews = msg.ConvertFromProtoToLinkPreviews(urlMaker)
	require.Len(t, previews, 1)
	p = previews[0]
	require.Equal(t, 0, p.Thumbnail.Height)
	require.Equal(t, 0, p.Thumbnail.Width)
	require.Equal(t, "", p.Thumbnail.URL)
}

func TestConvertStatusLinkPreviewsToProto(t *testing.T) {
	contact := &StatusContactLinkPreview{
		PublicKey:   "PublicKey_1",
		DisplayName: "DisplayName_2",
		Description: "Description_3",
		Icon: LinkPreviewThumbnail{
			Width:   10,
			Height:  20,
			DataURI: "data:image/png;base64,iVBORw0KGgoAAAANSUg=",
		},
	}

	community := &StatusCommunityLinkPreview{
		CommunityID:  "CommunityID_4",
		DisplayName:  "DisplayName_5",
		Description:  "Description_6",
		MembersCount: 7,
		Color:        "Color_8",
		TagIndices:   []uint32{9, 10},
		Icon: LinkPreviewThumbnail{
			Width:   30,
			Height:  40,
			DataURI: "data:image/png;base64,iVBORw0KGgoAAAANSUg=",
		},
		Banner: LinkPreviewThumbnail{
			Width:   50,
			Height:  60,
			DataURI: "data:image/png;base64,iVBORw0KGgoAAAANSUg=",
		},
	}

	channel := &StatusCommunityChannelLinkPreview{
		ChannelUUID: "ChannelUUID_11",
		Emoji:       "Emoji_12",
		DisplayName: "DisplayName_13",
		Description: "Description_14",
		Color:       "Color_15",
		Community: &StatusCommunityLinkPreview{
			CommunityID:  "CommunityID_16",
			DisplayName:  "DisplayName_17",
			Description:  "Description_18",
			MembersCount: 19,
			Color:        "Color_20",
			TagIndices:   []uint32{21, 22},
			Icon: LinkPreviewThumbnail{
				Width:   70,
				Height:  80,
				DataURI: "data:image/png;base64,iVBORw0KGgoAAAANSUg=",
			},
			Banner: LinkPreviewThumbnail{
				Width:   90,
				Height:  100,
				DataURI: "data:image/png;base64,iVBORw0KGgoAAAANSUg=",
			},
		},
	}

	message := Message{
		StatusLinkPreviews: []StatusLinkPreview{
			{
				URL:     "https://status.app/u/",
				Contact: contact,
			},
			{
				URL:       "https://status.app/c/",
				Community: community,
			},
			{
				URL:     "https://status.app/cc/",
				Channel: channel,
			},
		},
	}

	expectedThumbnailPayload, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUg=")
	require.NoError(t, err)

	unfurledLinks, err := message.ConvertStatusLinkPreviewsToProto()
	require.NoError(t, err)
	require.Len(t, unfurledLinks.UnfurledStatusLinks, 3)

	// Contact link

	l1 := unfurledLinks.UnfurledStatusLinks[0]
	require.Equal(t, "https://status.app/u/", l1.Url)
	require.NotNil(t, l1.GetContact())
	require.Nil(t, l1.GetCommunity())
	require.Nil(t, l1.GetChannel())
	c1 := l1.GetContact()
	require.Equal(t, contact.PublicKey, c1.PublicKey)
	require.Equal(t, contact.DisplayName, c1.DisplayName)
	require.Equal(t, contact.Description, c1.Description)
	require.NotNil(t, c1.Icon)
	require.Equal(t, uint32(contact.Icon.Width), c1.Icon.Width)
	require.Equal(t, uint32(contact.Icon.Height), c1.Icon.Height)
	require.Equal(t, expectedThumbnailPayload, c1.Icon.Payload)

	// Community link

	l2 := unfurledLinks.UnfurledStatusLinks[1]
	require.Equal(t, "https://status.app/c/", l2.Url)
	require.NotNil(t, l2.GetCommunity())
	require.Nil(t, l2.GetContact())
	require.Nil(t, l2.GetChannel())
	c2 := l2.GetCommunity()
	require.Equal(t, community.CommunityID, c2.CommunityId)
	require.Equal(t, community.DisplayName, c2.DisplayName)
	require.Equal(t, community.Description, c2.Description)
	require.Equal(t, community.MembersCount, c2.MembersCount)
	require.Equal(t, community.Color, c2.Color)
	require.Equal(t, community.TagIndices, c2.TagIndices)
	require.NotNil(t, c2.Icon)
	require.Equal(t, uint32(community.Icon.Width), c2.Icon.Width)
	require.Equal(t, uint32(community.Icon.Height), c2.Icon.Height)
	require.Equal(t, expectedThumbnailPayload, c2.Icon.Payload)
	require.NotNil(t, c2.Banner)
	require.Equal(t, uint32(community.Banner.Width), c2.Banner.Width)
	require.Equal(t, uint32(community.Banner.Height), c2.Banner.Height)
	require.Equal(t, expectedThumbnailPayload, c2.Banner.Payload)

	// Channel link

	l3 := unfurledLinks.UnfurledStatusLinks[2]
	require.Equal(t, "https://status.app/cc/", l3.Url)
	require.NotNil(t, l3.GetChannel())
	require.Nil(t, l3.GetContact())
	require.Nil(t, l3.GetCommunity())

	c3 := l3.GetChannel()
	require.Equal(t, channel.ChannelUUID, c3.ChannelUuid)
	require.Equal(t, channel.Emoji, c3.Emoji)
	require.Equal(t, channel.DisplayName, c3.DisplayName)
	require.Equal(t, channel.Description, c3.Description)
	require.Equal(t, channel.Color, c3.Color)

	require.NotNil(t, c3.Community)
	require.Equal(t, channel.Community.CommunityID, c3.Community.CommunityId)
	require.Equal(t, channel.Community.DisplayName, c3.Community.DisplayName)
	require.Equal(t, channel.Community.Color, c3.Community.Color)
	require.Equal(t, channel.Community.Description, c3.Community.Description)
	require.Equal(t, channel.Community.MembersCount, c3.Community.MembersCount)
	require.NotNil(t, c3.Community.Icon)
	require.Equal(t, uint32(channel.Community.Icon.Width), c3.Community.Icon.Width)
	require.Equal(t, uint32(channel.Community.Icon.Height), c3.Community.Icon.Height)
	require.Equal(t, expectedThumbnailPayload, c3.Community.Icon.Payload)
	require.NotNil(t, c3.Community.Banner)
	require.Equal(t, uint32(channel.Community.Banner.Width), c3.Community.Banner.Width)
	require.Equal(t, uint32(channel.Community.Banner.Height), c3.Community.Banner.Height)
	require.Equal(t, expectedThumbnailPayload, c3.Community.Banner.Payload)

	// Test any invalid link preview causes an early return.
	invalidContactPreview := contact
	invalidContactPreview.PublicKey = ""
	invalidPreview := message.StatusLinkPreviews[0]
	invalidPreview.Contact = invalidContactPreview
	message.StatusLinkPreviews = []StatusLinkPreview{invalidPreview}
	_, err = message.ConvertStatusLinkPreviewsToProto()
	require.ErrorContains(t, err, "invalid status link preview, url='https://status.app/u/'")
}

func TestConvertFromProtoToStatusLinkPreviews(t *testing.T) {

	contact := &protobuf.UnfurledStatusContactLink{
		PublicKey:   "PublicKey_1",
		DisplayName: "DisplayName_2",
		Description: "Description_3",
		Icon: &protobuf.UnfurledLinkThumbnail{
			Width:   10,
			Height:  20,
			Payload: []byte(""),
		},
	}

	community := &protobuf.UnfurledStatusCommunityLink{
		CommunityId:  "CommunityId_4",
		DisplayName:  "DisplayName_5",
		Description:  "Description_6",
		MembersCount: 7,
		Color:        "Color_8",
		TagIndices:   []uint32{9, 10},
		Icon: &protobuf.UnfurledLinkThumbnail{
			Width:   30,
			Height:  40,
			Payload: []byte(""),
		},
		Banner: &protobuf.UnfurledLinkThumbnail{
			Width:   50,
			Height:  60,
			Payload: []byte(""),
		},
	}

	channel := &protobuf.UnfurledStatusChannelLink{
		ChannelUuid: "ChannelUuid_11",
		Emoji:       "Emoji_12",
		DisplayName: "DisplayName_13",
		Description: "Description_14",
		Color:       "Color_15",
		Community: &protobuf.UnfurledStatusCommunityLink{
			CommunityId:  "CommunityId_16",
			DisplayName:  "DisplayName_17",
			Description:  "Description_18",
			MembersCount: 19,
			Color:        "Color_20",
			TagIndices:   []uint32{21, 22},
			Icon: &protobuf.UnfurledLinkThumbnail{
				Width:   70,
				Height:  80,
				Payload: []byte(""),
			},
			Banner: &protobuf.UnfurledLinkThumbnail{
				Width:   90,
				Height:  100,
				Payload: []byte(""),
			},
		},
	}

	msg := Message{
		ID: "42",
		ChatMessage: &protobuf.ChatMessage{
			UnfurledStatusLinks: &protobuf.UnfurledStatusLinks{
				UnfurledStatusLinks: []*protobuf.UnfurledStatusLink{
					{
						Url: "https://status.app/u/",
						Payload: &protobuf.UnfurledStatusLink_Contact{
							Contact: contact,
						},
					},
					{
						Url: "https://status.app/c/",
						Payload: &protobuf.UnfurledStatusLink_Community{
							Community: community,
						},
					},
					{
						Url: "https://status.app/cc/",
						Payload: &protobuf.UnfurledStatusLink_Channel{
							Channel: channel,
						},
					},
				},
			},
		},
	}

	urlMaker := func(msgID string, linkURL string, imageID string) string {
		return "https://localhost:6666/" + msgID + "-" + linkURL + "-" + imageID
	}

	previews := msg.ConvertFromProtoToStatusLinkPreviews(urlMaker)
	require.Len(t, previews, 3)

	// Contact preview

	p1 := previews[0]
	require.Equal(t, "https://status.app/u/", p1.URL)
	require.NotNil(t, p1.Contact)
	require.Nil(t, p1.Community)
	require.Nil(t, p1.Channel)

	c1 := p1.Contact
	require.NotNil(t, c1)
	require.Equal(t, contact.PublicKey, c1.PublicKey)
	require.Equal(t, contact.DisplayName, c1.DisplayName)
	require.Equal(t, contact.Description, c1.Description)
	require.NotNil(t, c1.Icon)
	require.Equal(t, int(contact.Icon.Width), c1.Icon.Width)
	require.Equal(t, int(contact.Icon.Height), c1.Icon.Height)
	require.Equal(t, "", c1.Icon.DataURI)
	require.Equal(t, "https://localhost:6666/42-https://status.app/u/-contact-icon", c1.Icon.URL)

	// Community preview

	p2 := previews[1]
	require.Equal(t, "https://status.app/c/", p2.URL)
	require.NotNil(t, p2.Community)
	require.Nil(t, p2.Contact)
	require.Nil(t, p2.Channel)

	c2 := p2.Community
	require.Equal(t, community.CommunityId, c2.CommunityID)
	require.Equal(t, community.DisplayName, c2.DisplayName)
	require.Equal(t, community.Description, c2.Description)
	require.Equal(t, community.MembersCount, c2.MembersCount)
	require.Equal(t, community.Color, c2.Color)
	require.Equal(t, community.TagIndices, c2.TagIndices)
	require.NotNil(t, c2.Icon)
	require.Equal(t, int(community.Icon.Width), c2.Icon.Width)
	require.Equal(t, int(community.Icon.Height), c2.Icon.Height)
	require.Equal(t, "", c2.Icon.DataURI)
	require.Equal(t, "https://localhost:6666/42-https://status.app/c/-community-icon", c2.Icon.URL)
	require.NotNil(t, c2.Banner)
	require.Equal(t, int(community.Banner.Width), c2.Banner.Width)
	require.Equal(t, int(community.Banner.Height), c2.Banner.Height)
	require.Equal(t, "", c2.Banner.DataURI)
	require.Equal(t, "https://localhost:6666/42-https://status.app/c/-community-banner", c2.Banner.URL)

	// Channel preview

	p3 := previews[2]
	require.Equal(t, "https://status.app/cc/", p3.URL)
	require.NotNil(t, p3.Channel)
	require.Nil(t, p3.Contact)
	require.Nil(t, p3.Community)

	c3 := previews[2].Channel
	require.Equal(t, channel.ChannelUuid, c3.ChannelUUID)
	require.Equal(t, channel.Emoji, c3.Emoji)
	require.Equal(t, channel.DisplayName, c3.DisplayName)
	require.Equal(t, channel.Description, c3.Description)
	require.Equal(t, channel.Color, c3.Color)

	require.NotNil(t, p3.Channel.Community)
	require.Equal(t, channel.Community.CommunityId, c3.Community.CommunityID)
	require.Equal(t, channel.Community.DisplayName, c3.Community.DisplayName)
	require.Equal(t, channel.Community.Color, c3.Community.Color)
	require.Equal(t, channel.Community.Description, c3.Community.Description)
	require.Equal(t, channel.Community.MembersCount, c3.Community.MembersCount)
	require.NotNil(t, c3.Community.Icon)
	require.Equal(t, int(channel.Community.Icon.Width), c3.Community.Icon.Width)
	require.Equal(t, int(channel.Community.Icon.Height), c3.Community.Icon.Height)
	require.Equal(t, "", c3.Community.Icon.DataURI)
	require.Equal(t, "https://localhost:6666/42-https://status.app/cc/-channel-community-icon", c3.Community.Icon.URL)
	require.NotNil(t, c3.Community.Banner)
	require.Equal(t, int(channel.Community.Banner.Width), c3.Community.Banner.Width)
	require.Equal(t, int(channel.Community.Banner.Height), c3.Community.Banner.Height)
	require.Equal(t, "", c3.Community.Banner.DataURI)
	require.Equal(t, "https://localhost:6666/42-https://status.app/cc/-channel-community-banner", c3.Community.Banner.URL)

}

func assertMarshalAndUnmarshalJSON[T any](t *testing.T, obj *T, msgAndArgs ...any) {
	rawJSON, err := json.Marshal(obj)
	require.NoError(t, err, msgAndArgs...)

	var unmarshalled T
	err = json.Unmarshal(rawJSON, &unmarshalled)
	require.NoError(t, err, msgAndArgs...)
	require.Equal(t, obj, &unmarshalled, msgAndArgs...)
}

func TestMarshalMessageJSON(t *testing.T) {
	msg := &Message{
		ID:   "1",
		From: "0x04c51631b3354242d5a56f044c3b7703bcc001e8c725c4706928b3fac3c2a12ec9019e1e224d487f5c893389405bcec998bc687307f290a569d6a97d24b711bca8",
		LinkPreviews: []LinkPreview{
			{
				Type:        protobuf.UnfurledLink_LINK,
				Description: "GitHub is where people build software.",
				Hostname:    "github.com",
				Title:       "Build software better, together",
				URL:         "https://github.com",
				Thumbnail: LinkPreviewThumbnail{
					Width:   100,
					Height:  200,
					URL:     "http://localhost:9999",
					DataURI: "data:image/png;base64,iVBORw0KGgoAAAANSUg=",
				},
			},
		},
	}

	assertMarshalAndUnmarshalJSON(t, msg, "message ID='%s'", msg.ID)
}
