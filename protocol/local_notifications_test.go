package protocol

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/contacts"
	"github.com/status-im/status-go/protocol/protobuf"
)

// mockNotificationSettings implements NotificationSettingsProvider for unit tests
type mockNotificationSettings struct {
	oneToOneChats      string
	groupChats         string
	personalMentions   string
	globalMentions     string
	allMessages        string
	messagePreview     int
	hasExemption       bool
	exMuteAllMessages  bool
	exPersonalMentions string
	exGlobalMentions   string
	exOtherMessages    string
}

func (m *mockNotificationSettings) GetOneToOneChats() (string, error) { return m.oneToOneChats, nil }
func (m *mockNotificationSettings) GetGroupChats() (string, error)    { return m.groupChats, nil }
func (m *mockNotificationSettings) GetPersonalMentions() (string, error) {
	return m.personalMentions, nil
}
func (m *mockNotificationSettings) GetGlobalMentions() (string, error)   { return m.globalMentions, nil }
func (m *mockNotificationSettings) GetAllMessages() (string, error)      { return m.allMessages, nil }
func (m *mockNotificationSettings) GetMessagePreview() (int, error)      { return m.messagePreview, nil }
func (m *mockNotificationSettings) HasExemption(id string) (bool, error) { return m.hasExemption, nil }
func (m *mockNotificationSettings) GetExMuteAllMessages(id string) (bool, error) {
	return m.exMuteAllMessages, nil
}
func (m *mockNotificationSettings) GetExPersonalMentions(id string) (string, error) {
	return m.exPersonalMentions, nil
}
func (m *mockNotificationSettings) GetExGlobalMentions(id string) (string, error) {
	return m.exGlobalMentions, nil
}
func (m *mockNotificationSettings) GetExOtherMessages(id string) (string, error) {
	return m.exOtherMessages, nil
}

func TestApplyMessagePreview(t *testing.T) {
	title, message := "Alice", "Hello there"
	t.Run("anonymous", func(t *testing.T) {
		dt, dm := applyMessagePreview(title, message, messagePreviewAnonymous)
		require.Equal(t, messagePreviewAnonymousTitle, dt)
		require.Equal(t, messagePreviewDefaultMessage, dm)
	})
	t.Run("name only", func(t *testing.T) {
		dt, dm := applyMessagePreview(title, message, messagePreviewNameOnly)
		require.Equal(t, title, dt)
		require.Equal(t, messagePreviewDefaultMessage, dm)
	})
	t.Run("name and message", func(t *testing.T) {
		dt, dm := applyMessagePreview(title, message, messagePreviewNameAndMessage)
		require.Equal(t, title, dt)
		require.Equal(t, message, dm)
	})
	t.Run("unknown value defaults to full", func(t *testing.T) {
		dt, dm := applyMessagePreview(title, message, 99)
		require.Equal(t, title, dt)
		require.Equal(t, message, dm)
	})
}

func TestGetMessagePreviewTextNilMessage(t *testing.T) {
	require.Equal(t, "", getMessagePreviewText(nil, nil))
}

func TestGetMessagePreviewTextTrimsMessageText(t *testing.T) {
	msg := &common.Message{
		ChatMessage: &protobuf.ChatMessage{
			Text: "  hello world  ",
		},
	}
	require.Equal(t, "hello world", getMessagePreviewText(msg, nil))
}

func TestTimestampOrZero(t *testing.T) {
	require.Equal(t, uint64(0), timestampOrZero(nil))

	msg := &common.Message{WhisperTimestamp: 42}
	require.Equal(t, uint64(42), timestampOrZero(msg))
}

func TestToContactRequestNotificationFallbackMessageAndZeroTimestamp(t *testing.T) {
	body := NotificationBody{
		Contact: &contacts.Contact{
			ID:          "0xabc",
			DisplayName: "Alice",
		},
		Chat: &Chat{
			ID:       "chat-1",
			ChatType: ChatTypeOneToOne,
		},
	}

	notif, err := body.toContactRequestNotification("0x1", 0, messagePreviewNameAndMessage)
	require.NoError(t, err)
	require.Equal(t, "Alice sent you a contact request", notif.Message)
	require.Equal(t, uint64(0), notif.Timestamp)
}

func TestToPrivateGroupInviteNotificationUsesMessagePreviewWhenPresent(t *testing.T) {
	body := NotificationBody{
		Contact: &contacts.Contact{
			ID:          "0xabc",
			DisplayName: "Alice",
		},
		Chat: &Chat{
			ID:       "group-1",
			Name:     "Secret Group",
			ChatType: ChatTypePrivateGroupChat,
		},
		Message: &common.Message{
			ChatMessage: &protobuf.ChatMessage{
				Text: "  please join us  ",
			},
		},
	}

	notif := body.toPrivateGroupInviteNotification("0x2", 0, messagePreviewNameAndMessage)
	require.Equal(t, "please join us", notif.Message)
}

func TestShowMessageNotification_OneToOneChats(t *testing.T) {
	key, _ := crypto.GenerateKey()
	pkHex := types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey))
	chat := &Chat{ID: pkHex, ChatType: ChatTypeOneToOne, Active: true}
	msg := common.NewMessage()
	msg.ID, msg.ChatId, msg.Text, msg.From = "m1", chat.ID, "hi", "0xabc"
	msg.MessageType = protobuf.MessageType_ONE_TO_ONE

	t.Run("SendAlerts allows notification", func(t *testing.T) {
		settings := &mockNotificationSettings{oneToOneChats: notifValueSendAlerts}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})
	t.Run("TurnOff blocks notification", func(t *testing.T) {
		settings := &mockNotificationSettings{oneToOneChats: notifValueTurnOff}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.False(t, got)
	})
}

func TestShowMessageNotification_GroupChats(t *testing.T) {
	chat := &Chat{ID: "grp1", ChatType: ChatTypePrivateGroupChat, Active: true, Name: "Group"}
	msg := common.NewMessage()
	msg.ID, msg.ChatId, msg.Text, msg.From = "m1", chat.ID, "hi", "0xabc"
	msg.MessageType = protobuf.MessageType_PRIVATE_GROUP
	key, _ := crypto.GenerateKey()

	t.Run("SendAlerts allows notification", func(t *testing.T) {
		settings := &mockNotificationSettings{groupChats: notifValueSendAlerts}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})
	t.Run("TurnOff blocks notification", func(t *testing.T) {
		settings := &mockNotificationSettings{groupChats: notifValueTurnOff}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.False(t, got)
	})
}

func TestShowMessageNotification_PublicChat_Mentions(t *testing.T) {
	chat := &Chat{ID: "pub1", ChatType: ChatTypePublic, Active: true, Name: "status"}
	msg := common.NewMessage()
	msg.ID, msg.ChatId, msg.Text, msg.From, msg.Mentioned = "m1", chat.ID, "hi", "0xabc", true
	msg.MessageType = protobuf.MessageType_PUBLIC_GROUP
	key, _ := crypto.GenerateKey()

	t.Run("PersonalMentions SendAlerts allows", func(t *testing.T) {
		settings := &mockNotificationSettings{personalMentions: notifValueSendAlerts, globalMentions: notifValueTurnOff}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})
	t.Run("GlobalMentions SendAlerts allows", func(t *testing.T) {
		settings := &mockNotificationSettings{personalMentions: notifValueTurnOff, globalMentions: notifValueSendAlerts}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})
	t.Run("Both TurnOff blocks", func(t *testing.T) {
		settings := &mockNotificationSettings{personalMentions: notifValueTurnOff, globalMentions: notifValueTurnOff}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.False(t, got)
	})
}

func TestShowMessageNotification_PublicChat_AllMessages(t *testing.T) {
	chat := &Chat{ID: "pub1", ChatType: ChatTypePublic, Active: true, Name: "status"}
	msg := common.NewMessage()
	msg.ID, msg.ChatId, msg.Text, msg.From = "m1", chat.ID, "hi", "0xabc"
	msg.MessageType = protobuf.MessageType_PUBLIC_GROUP
	key, _ := crypto.GenerateKey()

	t.Run("AllMessages SendAlerts allows", func(t *testing.T) {
		settings := &mockNotificationSettings{allMessages: notifValueSendAlerts}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})
	t.Run("AllMessages TurnOff blocks", func(t *testing.T) {
		settings := &mockNotificationSettings{allMessages: notifValueTurnOff}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.False(t, got)
	})
}

func TestShowMessageNotification_PublicChat_ReplyToOwn(t *testing.T) {
	key, _ := crypto.GenerateKey()
	pkHex := crypto.PubkeyToHex(&key.PublicKey)
	chat := &Chat{ID: "pub1", ChatType: ChatTypePublic, Active: true, Name: "status"}
	parent := common.NewMessage()
	parent.ID, parent.ChatId, parent.From = "p1", chat.ID, pkHex
	msg := common.NewMessage()
	msg.ID, msg.ChatId, msg.Text, msg.From, msg.ResponseTo = "m1", chat.ID, "reply", "0xabc", "p1"
	msg.MessageType = protobuf.MessageType_PUBLIC_GROUP

	t.Run("AllMessages SendAlerts allows reply to own", func(t *testing.T) {
		settings := &mockNotificationSettings{allMessages: notifValueSendAlerts}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, parent)
		require.True(t, got)
	})
	t.Run("AllMessages TurnOff blocks reply to own", func(t *testing.T) {
		settings := &mockNotificationSettings{allMessages: notifValueTurnOff}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, parent)
		require.False(t, got)
	})
}

func TestShowMessageNotification_Exemptions(t *testing.T) {
	chat := &Chat{ID: "comm1", ChatType: ChatTypeCommunityChat, Active: true, Name: "Community", CommunityID: "comm1"}
	key, _ := crypto.GenerateKey()

	t.Run("exMuteAllMessages blocks even with mentions", func(t *testing.T) {
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From, msg.Mentioned = "m1", chat.ID, "hi", "0xabc", true
		msg.MessageType = protobuf.MessageType_COMMUNITY_CHAT
		settings := &mockNotificationSettings{
			personalMentions:  notifValueSendAlerts,
			globalMentions:    notifValueSendAlerts,
			hasExemption:      true,
			exMuteAllMessages: true,
		}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.False(t, got)
	})

	t.Run("exemption overrides AllMessages for other messages", func(t *testing.T) {
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From = "m1", chat.ID, "hi", "0xabc"
		msg.MessageType = protobuf.MessageType_COMMUNITY_CHAT
		settings := &mockNotificationSettings{
			allMessages:     notifValueSendAlerts,
			hasExemption:    true,
			exOtherMessages: notifValueTurnOff,
		}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.False(t, got)
	})

	t.Run("exemption SendAlerts allows when global TurnOff", func(t *testing.T) {
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From = "m1", chat.ID, "hi", "0xabc"
		msg.MessageType = protobuf.MessageType_COMMUNITY_CHAT
		settings := &mockNotificationSettings{
			allMessages:     notifValueTurnOff,
			hasExemption:    true,
			exOtherMessages: notifValueSendAlerts,
		}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})

	t.Run("exPersonalMentions exemption TurnOff blocks when global SendAlerts", func(t *testing.T) {
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From, msg.Mentioned = "m1", chat.ID, "hi", "0xabc", true
		msg.MessageType = protobuf.MessageType_COMMUNITY_CHAT
		settings := &mockNotificationSettings{
			personalMentions:   notifValueSendAlerts,
			globalMentions:     notifValueSendAlerts,
			hasExemption:       true,
			exPersonalMentions: notifValueTurnOff,
			exGlobalMentions:   notifValueTurnOff,
		}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.False(t, got)
	})

	t.Run("exPersonalMentions exemption SendAlerts allows when global TurnOff", func(t *testing.T) {
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From, msg.Mentioned = "m1", chat.ID, "hi", "0xabc", true
		msg.MessageType = protobuf.MessageType_COMMUNITY_CHAT
		settings := &mockNotificationSettings{
			personalMentions:   notifValueTurnOff,
			globalMentions:     notifValueTurnOff,
			hasExemption:       true,
			exPersonalMentions: notifValueSendAlerts,
		}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})

	t.Run("exGlobalMentions exemption SendAlerts allows when global TurnOff", func(t *testing.T) {
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From, msg.Mentioned = "m1", chat.ID, "hi", "0xabc", true
		msg.MessageType = protobuf.MessageType_COMMUNITY_CHAT
		settings := &mockNotificationSettings{
			personalMentions: notifValueTurnOff,
			globalMentions:   notifValueTurnOff,
			hasExemption:     true,
			exGlobalMentions: notifValueSendAlerts,
		}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})
}

func TestShowMessageNotification_InactiveChat(t *testing.T) {
	key, _ := crypto.GenerateKey()
	pkHex := types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey))

	t.Run("inactive 1:1 still notifies", func(t *testing.T) {
		chat := &Chat{ID: pkHex, ChatType: ChatTypeOneToOne, Active: false}
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From = "m1", chat.ID, "hi", "0xabc"
		msg.MessageType = protobuf.MessageType_ONE_TO_ONE
		settings := &mockNotificationSettings{oneToOneChats: notifValueSendAlerts}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})

	t.Run("inactive public chat blocks", func(t *testing.T) {
		chat := &Chat{ID: "pub1", ChatType: ChatTypePublic, Active: false, Name: "status"}
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From = "m1", chat.ID, "hi", "0xabc"
		msg.MessageType = protobuf.MessageType_PUBLIC_GROUP
		settings := &mockNotificationSettings{allMessages: notifValueSendAlerts}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.False(t, got)
	})
}

func TestShowMessageNotification_NilSettings(t *testing.T) {
	key, _ := crypto.GenerateKey()
	pkHex := types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey))

	t.Run("nil settings 1:1 allows", func(t *testing.T) {
		chat := &Chat{ID: pkHex, ChatType: ChatTypeOneToOne, Active: true}
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From = "m1", chat.ID, "hi", "0xabc"
		msg.MessageType = protobuf.MessageType_ONE_TO_ONE
		got := showMessageNotification(nil, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})

	t.Run("nil settings mention allows", func(t *testing.T) {
		chat := &Chat{ID: "pub1", ChatType: ChatTypePublic, Active: true}
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From, msg.Mentioned = "m1", chat.ID, "hi", "0xabc", true
		msg.MessageType = protobuf.MessageType_PUBLIC_GROUP
		got := showMessageNotification(nil, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})
}

func TestShowMessageNotification_CommunityChat(t *testing.T) {
	chat := &Chat{ID: "chan1", ChatType: ChatTypeCommunityChat, Active: true, Name: "General", CommunityID: "comm1"}
	key, _ := crypto.GenerateKey()

	t.Run("AllMessages applies to community chat", func(t *testing.T) {
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From = "m1", chat.ID, "hi", "0xabc"
		msg.MessageType = protobuf.MessageType_COMMUNITY_CHAT
		settings := &mockNotificationSettings{allMessages: notifValueSendAlerts}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})

	t.Run("mention in community uses Personal/GlobalMentions", func(t *testing.T) {
		msg := common.NewMessage()
		msg.ID, msg.ChatId, msg.Text, msg.From, msg.Mentioned = "m1", chat.ID, "hi @me", "0xabc", true
		msg.MessageType = protobuf.MessageType_COMMUNITY_CHAT
		settings := &mockNotificationSettings{personalMentions: notifValueSendAlerts, globalMentions: notifValueTurnOff}
		got := showMessageNotification(settings, key.PublicKey, msg, chat, nil)
		require.True(t, got)
	})
}

func TestCommunityIconDataURI(t *testing.T) {
	t.Run("nil returns empty", func(t *testing.T) {
		got := communityIconDataURI(nil)
		require.Empty(t, got)
	})
}

func TestChatIconDataURI(t *testing.T) {
	t.Run("nil returns empty", func(t *testing.T) {
		got := chatIconDataURI(nil)
		require.Empty(t, got)
	})
	t.Run("Base64Image as data URI returned as-is", func(t *testing.T) {
		chat := &Chat{Base64Image: "data:image/png;base64,iVBORw0KGgo="}
		got := chatIconDataURI(chat)
		require.Equal(t, "data:image/png;base64,iVBORw0KGgo=", got)
	})
	t.Run("Base64Image as raw base64 gets prefix", func(t *testing.T) {
		chat := &Chat{Base64Image: "abc123"}
		got := chatIconDataURI(chat)
		require.Equal(t, "data:image/png;base64,abc123", got)
	})
	t.Run("Identicon as data URI returned when no Base64Image", func(t *testing.T) {
		chat := &Chat{Identicon: "data:image/png;base64,identicon123"}
		got := chatIconDataURI(chat)
		require.Equal(t, "data:image/png;base64,identicon123", got)
	})
	t.Run("Identicon as raw base64 gets prefix when no Base64Image", func(t *testing.T) {
		chat := &Chat{Identicon: "rawidenticon"}
		got := chatIconDataURI(chat)
		require.Equal(t, "data:image/png;base64,rawidenticon", got)
	})
	t.Run("Base64Image preferred over Identicon", func(t *testing.T) {
		chat := &Chat{Base64Image: "data:image/png;base64,img", Identicon: "data:image/png;base64,id"}
		got := chatIconDataURI(chat)
		require.Equal(t, "data:image/png;base64,img", got)
	})
}

func TestNewMessageNotification_Icons(t *testing.T) {
	key, _ := crypto.GenerateKey()
	pkHex := types.EncodeHex(crypto.FromECDSAPub(&key.PublicKey))
	contact, _ := contacts.BuildContactFromPublicKey(&key.PublicKey)
	contact.DisplayName = "Alice"
	resolvePrimaryName := func(id string) (string, error) { return id, nil }
	profilePicturesVisibility := int(settings.ProfilePicturesVisibilityEveryone)
	messagePreview := messagePreviewNameAndMessage

	msg := common.NewMessage()
	msg.ID, msg.ChatId, msg.Text, msg.From = "m1", "chat1", "hi", pkHex
	msg.MessageType = protobuf.MessageType_ONE_TO_ONE
	msg.WhisperTimestamp = 1000

	t.Run("one-to-one has no CommunityIcon or ChatIcon", func(t *testing.T) {
		chat := &Chat{ID: pkHex, ChatType: ChatTypeOneToOne, Active: true}
		notif, err := NewMessageNotification("m1", msg, chat, contact, nil, resolvePrimaryName, profilePicturesVisibility, messagePreview)
		require.NoError(t, err)
		require.Empty(t, notif.CommunityIcon)
		require.Empty(t, notif.ChatIcon)
		require.NotEmpty(t, notif.Author.Icon)
	})

	t.Run("group chat has ChatIcon when chat has Base64Image", func(t *testing.T) {
		chat := &Chat{
			ID:          "grp1",
			ChatType:    ChatTypePrivateGroupChat,
			Active:      true,
			Name:        "Group",
			Base64Image: "data:image/png;base64,groupimg",
		}
		notif, err := NewMessageNotification("m1", msg, chat, contact, nil, resolvePrimaryName, profilePicturesVisibility, messagePreview)
		require.NoError(t, err)
		require.Empty(t, notif.CommunityIcon)
		require.Equal(t, "data:image/png;base64,groupimg", notif.ChatIcon)
	})

	t.Run("community chat with community that has image sets CommunityIcon", func(t *testing.T) {
		chat := &Chat{ID: "chan1", ChatType: ChatTypeCommunityChat, Active: true, Name: "General", CommunityID: "comm1"}
		community := createTestCommunityWithImage(t)
		notif, err := NewMessageNotification("m1", msg, chat, contact, community, resolvePrimaryName, profilePicturesVisibility, messagePreview)
		require.NoError(t, err)
		require.NotEmpty(t, notif.CommunityIcon, "CommunityIcon should be set when community has thumbnail image")
		require.Contains(t, notif.CommunityIcon, "data:image/")
		require.Contains(t, notif.CommunityIcon, "base64,")
		require.Empty(t, notif.ChatIcon)
	})

	t.Run("community chat without community has no CommunityIcon", func(t *testing.T) {
		chat := &Chat{ID: "chan1", ChatType: ChatTypeCommunityChat, Active: true, Name: "General", CommunityID: "comm1"}
		notif, err := NewMessageNotification("m1", msg, chat, contact, nil, resolvePrimaryName, profilePicturesVisibility, messagePreview)
		require.NoError(t, err)
		require.Empty(t, notif.CommunityIcon)
	})
}

func TestNewPrivateGroupInviteNotification_ChatIcon(t *testing.T) {
	key, _ := crypto.GenerateKey()
	contact, _ := contacts.BuildContactFromPublicKey(&key.PublicKey)
	contact.DisplayName = "Alice"
	profilePicturesVisibility := int(settings.ProfilePicturesVisibilityEveryone)
	messagePreview := messagePreviewNameAndMessage

	t.Run("sets ChatIcon from chat Base64Image", func(t *testing.T) {
		chat := &Chat{
			ID:          "grp1",
			ChatType:    ChatTypePrivateGroupChat,
			Name:        "Group",
			Base64Image: "data:image/png;base64,inviteimg",
		}
		notif := NewPrivateGroupInviteNotification("inv1", chat, contact, profilePicturesVisibility, messagePreview)
		require.Equal(t, "data:image/png;base64,inviteimg", notif.ChatIcon)
		require.NotEmpty(t, notif.Author.Icon)
	})
}

func TestNewCommunityRequestToJoinNotification_AuthorAndCommunityIcon(t *testing.T) {
	key, _ := crypto.GenerateKey()
	contact, _ := contacts.BuildContactFromPublicKey(&key.PublicKey)
	contact.DisplayName = "Bob"
	profilePicturesVisibility := int(settings.ProfilePicturesVisibilityEveryone)
	messagePreview := messagePreviewNameAndMessage

	t.Run("sets Author and CommunityIcon when community has image", func(t *testing.T) {
		community := createTestCommunityWithImage(t)
		notif := NewCommunityRequestToJoinNotification("req1", community, contact, profilePicturesVisibility, messagePreview)
		require.Equal(t, "Bob", notif.Author.Name)
		require.NotEmpty(t, notif.Author.ID)
		require.NotEmpty(t, notif.Author.Icon)
		require.NotEmpty(t, notif.CommunityIcon)
		require.Contains(t, notif.CommunityIcon, "data:image/")
	})
}

func TestNotification_MarshalJSON_IncludesIconFields(t *testing.T) {
	key, _ := crypto.GenerateKey()
	contact, _ := contacts.BuildContactFromPublicKey(&key.PublicKey)
	contact.DisplayName = "Alice"
	chat := &Chat{
		ID:          "grp1",
		ChatType:    ChatTypePrivateGroupChat,
		Name:        "Group",
		Base64Image: "data:image/png;base64,test",
	}
	notif := NewPrivateGroupInviteNotification("inv1", chat, contact, int(settings.ProfilePicturesVisibilityEveryone), messagePreviewNameAndMessage)
	data, err := json.Marshal(notif)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, "data:image/png;base64,test", parsed["chatIcon"])
}

// createTestCommunityWithImage returns a minimal community with a thumbnail image for testing.
func createTestCommunityWithImage(t *testing.T) *communities.Community {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	// Minimal 1x1 PNG (valid PNG header + IDAT + IEND)
	pngPayload := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xff, 0xff, 0x3f,
		0x00, 0x05, 0xfe, 0x02, 0xfe, 0xdc, 0xcc, 0x59,
		0xe7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
	config := communities.Config{
		PrivateKey:  key,
		ControlNode: &key.PublicKey,
		ID:          &key.PublicKey,
		CommunityDescription: &protobuf.CommunityDescription{
			Identity: &protobuf.ChatIdentity{
				DisplayName: "Test",
				Images: map[string]*protobuf.IdentityImage{
					"thumbnail": {Payload: pngPayload},
				},
			},
		},
		MemberIdentity: key,
	}
	community, err := communities.New(config, &timeSourceStub{}, &communities.NoopDescriptionEncryptor{}, nil)
	require.NoError(t, err)
	return community
}

type timeSourceStub struct{}

func (t *timeSourceStub) GetCurrentTime() uint64 {
	return 0
}
