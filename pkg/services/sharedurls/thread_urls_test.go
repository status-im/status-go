package sharedurls

import (
	"crypto/ecdsa"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/internal/crypto"
)

func TestThreadURLRoundTrip(t *testing.T) {
	parent, err := ParseSharedURL(channelURL)
	if err != nil {
		t.Fatal(err)
	}
	communityChatID := parent.Community.CommunityID + parent.Channel.ChannelUUID
	service := NewService(nil)
	for _, chatID := range []string{"group-id", communityChatID, "chat /?#&+%"} {
		const threadID = "thread /?#&+%"
		link, err := service.ShareThreadURL(chatID, threadID)
		if err != nil {
			t.Fatal(err)
		}
		for _, prefix := range []string{"https://status.app/", "http://status.app/", "status-app://"} {
			candidate := strings.Replace(link, baseShareURL+"/", prefix, 1)
			if !IsStatusSharedURL(candidate) || !IsThreadURL(candidate) {
				t.Fatalf("unrecognized: %s", candidate)
			}
			parsed, err := service.ParseThreadURL(candidate)
			if err != nil {
				t.Fatal(err)
			}
			communityID := ""
			if chatID == communityChatID {
				communityID = parent.Community.CommunityID
			}
			if parsed.ChatID != chatID || parsed.ThreadID != threadID || parsed.CommunityID != communityID {
				t.Fatalf("unexpected parsed data: %+v", parsed)
			}
		}
	}
}

func TestThreadURLPreservesParentFormat(t *testing.T) {
	service := NewService(nil)
	link, err := service.ShareThreadURL("parent", "thread")
	if err != nil || link != "https://status.app/g/?a2=parent&thread-id=thread" {
		t.Fatalf("unexpected group link %q: %v", link, err)
	}
	for _, parentLink := range []string{channelURL, channelURLWithData} {
		parent, err := ParseSharedURL(parentLink)
		if err != nil {
			t.Fatal(err)
		}
		parsedURL, err := url.Parse(parentLink)
		if err != nil {
			t.Fatal(err)
		}
		query := parsedURL.Query()
		query.Set("thread-id", "thread")
		parsedURL.RawQuery = query.Encode()
		extended := parsedURL.String()
		data, err := service.ParseThreadURL(extended)
		if err != nil {
			t.Fatal(err)
		}
		if data.ChatID != parent.Community.CommunityID+parent.Channel.ChannelUUID || data.ThreadID != "thread" {
			t.Fatalf("incorrect channel destination: %+v", data)
		}
		if strings.Index(extended, "?thread-id=") > strings.Index(extended, "#") {
			t.Fatalf("query was appended inside the community fragment: %s", extended)
		}
		link, err := service.ShareThreadURL(data.ChatID, "thread")
		if err != nil {
			t.Fatal(err)
		}
		generated, err := url.Parse(link)
		if err != nil {
			t.Fatal(err)
		}
		generated.RawQuery = ""
		expected, err := service.ShareCommunityChannelURLWithChatKey(mustDecodeCommunityID(t, data.CommunityID), parent.Channel.ChannelUUID)
		if err != nil || generated.String() != expected {
			t.Fatalf("changed parent link: %s, %v", generated, err)
		}
	}
}

func mustDecodeCommunityID(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := hexutil.Decode(value)
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func TestInvalidThreadURLs(t *testing.T) {
	for _, link := range []string{
		"", "https://example.org/g/?a2=chat&thread-id=thread",
		"https://status.app.evil/g/?a2=chat&thread-id=thread",
		"https://status.app/t/chat?thread-id=thread",
		"https://status.app/g/?thread-id=thread", "https://status.app/g/?a2=chat",
		"https://status.app/g/?a2=chat&thread-id=", "https://status.app/g/?a2=chat&thread-id=+",
		"https://status.app/g/?a2=chat&thread-id=a&thread-id=b",
		"https://status.app/g/?a2=chat&a2=other&thread-id=thread",
		"https://status.app/g/%zz?a2=chat&thread-id=thread",
		"https://status.app/g/?a2=chat&thread-id=%zz",
		"https://status.app/g/?a2=chat&thread-id=thread&sender-id=",
		"https://status.app/g/?a2=chat&thread-id=thread&sender-id=a&sender-id=b",
		"https://status.app/p/not-a-key?thread-id=thread",
		"https://status.app/cc/not-a-channel?thread-id=thread#not-a-key",
		"https://status.app/cc/003cdcd5-e065-48f9-b166-b1a94ac75a11?thread-id=thread",
		"https://status.app/u/key?thread-id=thread", "https://status.app/m/chat?message-id=message",
	} {
		if _, err := ParseThreadURL(link); err == nil {
			t.Errorf("accepted invalid URL: %q", link)
		}
	}
	for _, ids := range [][2]string{{"", "thread"}, {"chat", ""}, {" ", "thread"}, {"chat", " "}} {
		if _, err := NewService(nil).ShareThreadURL(ids[0], ids[1]); err == nil {
			t.Errorf("accepted empty identifiers: %q", ids)
		}
	}
}

func TestThreadURLClassification(t *testing.T) {
	for _, link := range []string{
		"https://status.app/g/?a2=chat&thread-id=thread",
		"https://status.app/cc/channel?thread-id=#key",
		"status-app://p/peer?%74hread-id=thread", "http://status.app/p/peer?thread-id",
		"https://status.app/p/peer?thread-id=%zz",
	} {
		if !IsThreadURL(link) {
			t.Errorf("missed thread destination: %s", link)
		}
	}
	for _, link := range []string{
		channelURL, channelURLWithData, "https://status.app/p/peer", "status-app://g/?a2=chat",
		"https://status.app/g/?a2=chat&other=thread-id", "https://status.app/g/?a2=chat#thread-id=thread",
		"https://status.app/u/key?thread-id=thread", "https://example.org/g/?a2=chat&thread-id=thread",
	} {
		if IsThreadURL(link) {
			t.Errorf("parent misclassified as thread destination: %s", link)
		}
	}
}

type threadIdentityProvider struct {
	NopDataProvider
	identity *ecdsa.PublicKey
	chats    map[string]string
}

func (p *threadIdentityProvider) IdentityPublicKey() *ecdsa.PublicKey {
	return p.identity
}

func (p *threadIdentityProvider) ResolveDirectThreadChatID(threadID string) (string, error) {
	if chatID := p.chats[threadID]; chatID != "" {
		return chatID, nil
	}
	return "", fmt.Errorf("thread unavailable")
}

func TestDirectThreadURLAcrossParticipants(t *testing.T) {
	alice, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	bob, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	outsider, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	aliceID := crypto.PubkeyToHex(&alice.PublicKey)
	bobID := crypto.PubkeyToHex(&bob.PublicKey)
	aliceAPI := &PublicAPI{service: NewService(&threadIdentityProvider{identity: &alice.PublicKey, chats: map[string]string{"thread": bobID, "self-thread": aliceID}})}
	bobAPI := &PublicAPI{service: NewService(&threadIdentityProvider{identity: &bob.PublicKey, chats: map[string]string{"thread": aliceID}})}
	link, err := aliceAPI.ShareThreadURL(bobID, "thread")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(link, aliceID) || strings.Contains(link, bobID) || strings.Contains(link, "sender-id") {
		t.Fatalf("direct thread link exposes participants: %s", link)
	}
	if link != baseShareURL+"/p/?thread-id=thread" {
		t.Fatalf("unexpected private link: %s", link)
	}
	for _, scheme := range []string{"https://status.app/", "http://status.app/", "status-app://"} {
		candidate := strings.Replace(link, "https://status.app/", scheme, 1)
		for _, tc := range []struct {
			api         *PublicAPI
			localChatID string
		}{{aliceAPI, bobID}, {bobAPI, aliceID}} {
			data, err := tc.api.ParseThreadURL(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if data.ChatID != tc.localChatID || data.ThreadID != "thread" || data.CommunityID != "" {
				t.Fatalf("incorrect local destination: %+v", data)
			}
		}
	}
	outsiderService := NewService(&threadIdentityProvider{identity: &outsider.PublicKey})
	if _, err := outsiderService.ParseThreadURL(link); err == nil {
		t.Fatal("third party accepted direct thread link")
	}
	reverseLink, err := bobAPI.ShareThreadURL(aliceID, "thread")
	if err != nil {
		t.Fatal(err)
	}
	if reverseLink != link {
		t.Fatal("participants generated different thread references")
	}
	if _, err := aliceAPI.ShareThreadURL(aliceID, "thread"); err == nil {
		t.Fatal("generated link for a thread in a different chat")
	}
	if _, err := aliceAPI.ShareThreadURL(bobID, "missing"); err == nil {
		t.Fatal("generated link for a missing thread")
	}
	reverse, err := aliceAPI.ParseThreadURL(reverseLink)
	if err != nil || reverse.ChatID != bobID {
		t.Fatalf("incorrect reverse destination: %+v, %v", reverse, err)
	}
	legacy := baseShareURL + "/p/" + bobID + "?thread-id=thread"
	if _, err := bobAPI.ParseThreadURL(legacy); err == nil {
		t.Fatal("ambiguous direct thread link accepted")
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("sender-id", "invalid-key")
	parsed.RawQuery = query.Encode()
	if _, err := bobAPI.ParseThreadURL(parsed.String()); err == nil {
		t.Fatal("invalid participant key accepted")
	}
	if _, err := NewService(nil).ShareThreadURL(bobID, "thread"); err == nil {
		t.Fatal("direct link generated without local thread data")
	}
	selfLink, err := aliceAPI.ShareThreadURL(aliceID, "self-thread")
	if err != nil {
		t.Fatal(err)
	}
	self, err := aliceAPI.ParseThreadURL(selfLink)
	if err != nil || self.ChatID != aliceID {
		t.Fatalf("incorrect self-chat destination: %+v, %v", self, err)
	}
}

func TestLegacyDirectThreadURLAcrossParticipants(t *testing.T) {
	alice, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	bob, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	outsider, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	aliceID := crypto.PubkeyToHex(&alice.PublicKey)
	bobID := crypto.PubkeyToHex(&bob.PublicKey)
	link := baseShareURL + "/p/" + bobID + "?sender-id=" + aliceID + "&thread-id=thread"
	for _, tc := range []struct {
		identity       *ecdsa.PublicKey
		expectedChatID string
	}{{&alice.PublicKey, bobID}, {&bob.PublicKey, aliceID}, {&outsider.PublicKey, ""}} {
		service := NewService(&threadIdentityProvider{identity: tc.identity})
		data, err := service.ParseThreadURL(link)
		if tc.expectedChatID == "" {
			if err == nil {
				t.Fatal("third party accepted legacy DM link")
			}
		} else if err != nil || data.ChatID != tc.expectedChatID {
			t.Fatalf("incorrect legacy destination: %+v, %v", data, err)
		}
	}
}

func TestPrivateThreadURLRequiresLocalData(t *testing.T) {
	service := NewService(nil)
	if _, err := service.ParseThreadURL(baseShareURL + "/p/?thread-id=missing"); err == nil {
		t.Fatal("resolved private link without local data")
	}
	for _, suffix := range []string{"&sender-id=invalid", "&sender-id=", "&thread-id=other"} {
		if _, err := ParseThreadURL(baseShareURL + "/p/?thread-id=thread" + suffix); err == nil {
			t.Fatalf("accepted malformed private link: %s", suffix)
		}
	}
	if _, err := ParseThreadURL(baseShareURL + "/p/?thread-id=" + strings.Repeat("a", 257)); err == nil {
		t.Fatal("accepted oversized thread reference")
	}
}

func TestPrivateThreadURLDoesNotRequireIdentity(t *testing.T) {
	peer, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	chatID := crypto.PubkeyToHex(&peer.PublicKey)
	service := NewService(&threadIdentityProvider{chats: map[string]string{"thread": chatID}})
	link, err := service.ShareThreadURL(chatID, "thread")
	if err != nil {
		t.Fatal(err)
	}
	data, err := service.ParseThreadURL(link)
	if err != nil || data.ChatID != chatID {
		t.Fatalf("local resolution unexpectedly required identity: %+v, %v", data, err)
	}
}
