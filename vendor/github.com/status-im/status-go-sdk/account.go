package sdk

import (
	"encoding/hex"
)

// Account represents a logged in user on statusd node
type Account struct {
	conn     *SDK
	Address  string
	PubKey   string
	Mnemonic string
	Username string
	channels []*Channel
}

// JoinPublicChannel joins a status public channel
func (a *Account) JoinPublicChannel(name string) (*Channel, error) {
	return a.createAndJoin(name, name)
}

// CreatePrivateChannel creates and joins a private channel
func (a *Account) CreatePrivateChannel(name, password string) (*Channel, error) {
	return a.createAndJoin(name, password)
}

func (a *Account) createAndJoin(name, password string) (*Channel, error) {
	symkeyResponse, err := shhGenerateSymKeyFromPasswordRequest(a.conn, []string{password})
	if err != nil {
		return nil, err
	}
	symKey := symkeyResponse.Key

	topicID, err := a.calculatePublicChannelTopicID(name, symkeyResponse.ID)
	if err != nil {
		return nil, err
	}

	return a.Join(name, topicID, symKey)
}

// Join joins a status channel
func (a *Account) Join(channelName, topicID, symKey string) (*Channel, error) {
	newMessageFilterResponse, err := newShhMessageFilterFormatRequest(a.conn, []string{topicID}, symKey)
	if err != nil {
		return nil, err
	}

	filterID := newMessageFilterResponse.FilterID

	ch := &Channel{
		account:    a,
		name:       channelName,
		filterID:   filterID,
		TopicID:    topicID,
		ChannelKey: symKey,
	}
	a.channels = append(a.channels, ch)

	return ch, nil
}

func (a *Account) calculatePublicChannelTopicID(name string, symkey int) (topicID string, err error) {
	p := "0x" + hex.EncodeToString([]byte(name))
	web3ShaResponse, err := web3Sha3Request(a.conn, symkey, []string{p})
	if err != nil {
		return
	}
	topicID = web3ShaResponse.Result[0:10]

	return
}

// Close all channels you're subscribed to
func (a *Account) Close() {
	for _, ch := range a.channels {
		ch.Close()
	}
}
