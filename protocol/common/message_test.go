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
	l := &protobuf.UnfurledLink{
		Description:      "GitHub is where people build software.",
		Title:            "Build software better, together",
		Type:             protobuf.UnfurledLink_LINK,
		Url:              "https://github.com",
		ThumbnailPayload: []byte(""),
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
