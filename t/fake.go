package t

import (
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/contacts"
	"github.com/status-im/status-go/protocol/requests"
)

func FakePrivateKey(t *testing.T) *ecdsa.PrivateKey {
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	return privateKey
}

func FakePublicKey(t *testing.T) *ecdsa.PublicKey {
	pk := FakePrivateKey(t)
	return &pk.PublicKey
}

func FakeContact(t *testing.T, key *ecdsa.PublicKey) *contacts.Contact {
	if key == nil {
		key = FakePublicKey(t)
	}

	var contact *contacts.Contact
	err := gofakeit.Struct(&contact)
	require.NoError(t, err)
	require.NotNil(t, contact)

	contact.ID = contacts.ContactIDFromPublicKey(key)

	return contact
}

type timeSourceStub struct {
}

func (t *timeSourceStub) GetCurrentTime() uint64 {
	return uint64(time.Now().Unix())
}

func FakeCommunity(t *testing.T, options ...FakeCommunityOption) *communities.Community {
	timeSource := timeSourceStub{}

	var config communities.Config
	err := gofakeit.Struct(&config)
	require.NoError(t, err)

	memberKey := FakePrivateKey(t)
	key := FakePrivateKey(t)

	config.ID = &key.PublicKey
	config.PrivateKey = key
	config.ControlNode = &key.PublicKey
	config.Logger = zap.NewNop()
	config.MemberIdentity = memberKey
	config.ControlDevice = true

	request := requests.CreateCommunity{}
	err = gofakeit.Struct(&request)
	require.NoError(t, err)

	// Image has to be real, otherwise the request doesn't pass validation
	request.Image = ""
	request.Banner.ImagePath = ""

	for _, opt := range options {
		opt(&request)
	}

	config.CommunityDescription, err = request.ToCommunityDescription()
	require.NoError(t, err)

	var community *communities.Community
	community, err = communities.New(config, &timeSource, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, community)

	return community
}

type FakeCommunityOption func(*requests.CreateCommunity)

func WithCommunityImage(path string, ax, ay, bx, by int) FakeCommunityOption {
	return func(request *requests.CreateCommunity) {
		request.Image = path
		request.ImageAx = ax
		request.ImageAy = ay
		request.ImageBx = bx
		request.ImageBy = by
	}
}

func WithCommunityBanner(path string, x, y, width, height int) FakeCommunityOption {
	return func(request *requests.CreateCommunity) {
		request.Banner = images.CroppedImage{
			ImagePath: path,
			X:         x,
			Y:         y,
			Width:     width,
			Height:    height,
		}
	}
}
