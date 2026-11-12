package sharedurls

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/protocol/common"
)

type ThreadURLData struct {
	ChatID      string `json:"chatId"`
	ThreadID    string `json:"threadId"`
	CommunityID string `json:"communityId"`
	SenderID    string `json:"senderId,omitempty"`
}

const maxThreadURLIDLength = 256

// ShareThreadURL adds a thread destination to its parent's existing link format.
// Direct chats omit participants and resolve their parent from local thread data.
func (s *Service) ShareThreadURL(chatID string, threadID string) (string, error) {
	if strings.TrimSpace(chatID) == "" || strings.TrimSpace(threadID) == "" || len(threadID) > maxThreadURLIDLength {
		return "", fmt.Errorf("valid chatID and threadID are required")
	}
	parent := &url.URL{Scheme: "https", Host: "status.app"}
	query := url.Values{}
	if _, err := crypto.HexToPubkey(chatID); err == nil {
		localChatID, err := s.provider.ResolveDirectThreadChatID(threadID)
		if err != nil {
			return "", err
		}
		if localChatID != chatID {
			return "", fmt.Errorf("thread does not belong to the selected direct chat")
		}
		parent.Path = "/p/"
	} else if communityID := deriveCommunityIDFromChatID(chatID); communityID != "" {
		communityKey, err := hexutil.Decode(communityID)
		if err != nil {
			return "", err
		}
		link, err := s.ShareCommunityChannelURLWithChatKey(communityKey, chatID[len(communityID):])
		if err != nil {
			return "", err
		}
		parent, err = url.Parse(link)
		if err != nil {
			return "", err
		}
	} else {
		parent.Path = "/g/"
		query.Set("a2", chatID)
	}
	query.Set("thread-id", threadID)
	parent.RawQuery = query.Encode()
	return parent.String(), nil
}

// ParseThreadURL resolves identity-free direct links locally. Participant-bearing
// links remain readable for compatibility. Neither form creates or grants access.
func (s *Service) ParseThreadURL(rawURL string) (*ThreadURLData, error) {
	data, err := ParseThreadURL(rawURL)
	if err != nil {
		return nil, err
	}
	if data.ChatID == "" {
		data.ChatID, err = s.provider.ResolveDirectThreadChatID(data.ThreadID)
		if err != nil {
			return nil, err
		}
		if _, err := crypto.HexToPubkey(data.ChatID); err != nil {
			return nil, fmt.Errorf("direct thread parent unavailable")
		}
		return data, nil
	}
	peer, peerErr := crypto.HexToPubkey(data.ChatID)
	if data.SenderID == "" {
		if peerErr == nil {
			return nil, fmt.Errorf("direct thread URL requires sender-id")
		}
		return data, nil
	}
	sender, senderErr := crypto.HexToPubkey(data.SenderID)
	if peerErr != nil || senderErr != nil {
		return nil, fmt.Errorf("invalid direct thread participants")
	}
	identity := s.provider.IdentityPublicKey()
	if identity == nil {
		return nil, errNoDataProvider
	}
	viewerID := crypto.PubkeyToHex(identity)
	peerID := crypto.PubkeyToHex(peer)
	senderID := crypto.PubkeyToHex(sender)
	switch viewerID {
	case peerID:
		data.ChatID = senderID
	case senderID:
		data.ChatID = peerID
	default:
		return nil, fmt.Errorf("direct thread URL is not addressed to this identity")
	}
	data.CommunityID = ""
	return data, nil
}

func parseThreadParentURL(rawURL string) (*url.URL, error) {
	return common.ParseStatusThreadParentURL(rawURL)
}

// IsThreadURL recognizes destinations before attempting to resolve them.
func IsThreadURL(rawURL string) bool {
	return common.IsStatusThreadURL(rawURL)
}

func ParseThreadURL(rawURL string) (*ThreadURLData, error) {
	parent, err := parseThreadParentURL(rawURL)
	if err != nil {
		return nil, err
	}
	query, err := url.ParseQuery(parent.RawQuery)
	if err != nil {
		return nil, err
	}
	threadIDs := query["thread-id"]
	if len(threadIDs) != 1 || strings.TrimSpace(threadIDs[0]) == "" || len(threadIDs[0]) > maxThreadURLIDLength {
		return nil, fmt.Errorf("one valid threadID is required")
	}
	senderIDs := query["sender-id"]
	if len(senderIDs) > 1 || (len(senderIDs) == 1 && strings.TrimSpace(senderIDs[0]) == "") {
		return nil, fmt.Errorf("invalid sender-id")
	}
	data := &ThreadURLData{ThreadID: threadIDs[0], SenderID: query.Get("sender-id")}
	switch {
	case strings.HasPrefix(parent.Path, "/p/"):
		data.ChatID = strings.TrimPrefix(parent.Path, "/p/")
		if data.ChatID == "" {
			if len(senderIDs) > 0 {
				return nil, fmt.Errorf("unexpected sender-id for local direct thread")
			}
			break
		}
		if _, err := crypto.HexToPubkey(data.ChatID); err != nil {
			return nil, fmt.Errorf("invalid direct chatID")
		}
	case strings.HasPrefix(parent.Path, "/g/"):
		chatIDs := query["a2"]
		if data.SenderID != "" || len(chatIDs) != 1 || strings.TrimSpace(chatIDs[0]) == "" {
			return nil, fmt.Errorf("one group chatID is required")
		}
		data.ChatID = chatIDs[0]
	case strings.HasPrefix(parent.Path, "/cc/"):
		if data.SenderID != "" {
			return nil, fmt.Errorf("unexpected sender-id for community thread")
		}
		// Preserve the channel payload and community key for the existing parser.
		parent.RawQuery = ""
		parsed, err := ParseSharedURL(parent.String())
		if err != nil {
			return nil, err
		}
		if parsed.Community == nil || parsed.Channel == nil {
			return nil, fmt.Errorf("invalid parent channel")
		}
		data.CommunityID = parsed.Community.CommunityID
		data.ChatID = data.CommunityID + parsed.Channel.ChannelUUID
	}
	return data, nil
}
