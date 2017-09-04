package notificationserver

import (
	"encoding/hex"
	"encoding/json"

	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

const (
	topicNewChatSession   = "/user/newchat"
	topicRegisterDevice   = "/chat/register"
	topicSendNotification = "/chat/notification"
	DefaultWorkTime       = 5
)

// ServerDiscovery handles server discovery requests
func (s *NotificationServer) ServerDiscovery(resp *whisper.MessageParams, req *whisper.ReceivedMessage) {
	// response payload
	payload := ServerDiscoveryResponse{
		ServerID: s.serverID,
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		// Error
		return
	}

	// response
	resp.Payload = rawPayload
	resp.Src = s.protocolKey
	resp.Dst = req.Src
	resp.Topic = req.Topic
	resp.TTL = whisper.DefaultTTL
	resp.PoW = whisper.DefaultMinimumPoW
	resp.WorkTime = DefaultWorkTime
}

// ServerAcceptance handles server acceptance requests
func (s *NotificationServer) ServerAcceptance(resp *whisper.MessageParams, req *whisper.ReceivedMessage) {
	// request payload
	var reqPayload ServerAcceptanceRequest
	if err := json.Unmarshal(req.Payload, &reqPayload); err != nil {
		// Error
		return
	}

	// make sure that only requests made to the current node are processed
	if reqPayload.ServerID != s.serverID {
		return
	}

	// generate client session key
	keyID, err := s.GenerateSymKey()
	if err != nil {
		// Error
		return
	}
	rawKey, err := s.GetSymKey(keyID)
	if err != nil {
		// Error
		return
	}

	// create session - handlers & session file
	s.HandleFunc(topicNewChatSession, rawKey, s.NewChatSession) // Error
	session := &Session{
		Type:   SessionClient,
		Key:    rawKey,
		Values: make(map[string]interface{}),
	}
	filename := crypto.Keccak256Hash(rawKey).String()
	if err := s.StoreSession(filename, session, hex.EncodeToString(crypto.FromECDSA(s.protocolKey))); err != nil {
		// Error
		return
	}

	// response payload
	respPayload := ServerAcceptanceResponse{
		Key: "0x" + hex.EncodeToString(rawKey),
	}
	rawRespPayload, err := json.Marshal(respPayload)
	if err != nil {
		// Error
		return
	}

	// response
	resp.Src = s.protocolKey
	resp.Dst = req.Src
	resp.Topic = req.Topic
	resp.TTL = whisper.DefaultTTL
	resp.PoW = whisper.DefaultMinimumPoW
	resp.WorkTime = DefaultWorkTime
	resp.Payload = rawRespPayload
}

// NewChatSession handles chat creation requests
func (s *NotificationServer) NewChatSession(resp *whisper.MessageParams, req *whisper.ReceivedMessage) {
	// generate chat session key
	keyID, err := s.GenerateSymKey()
	if err != nil {
		// Error
		return
	}
	rawKey, err := s.GetSymKey(keyID)
	if err != nil {
		// Error
		return
	}

	rspPayload := NewChatSessionResponse{
		Key: "0x" + hex.EncodeToString(rawKey),
	}
	rawRespPayload, err := json.Marshal(rspPayload)
	if err != nil {
		// Error Handling
		return
	}

	// create chat session - handlers & session file
	if err := s.HandleFunc(topicSendNotification, rawKey, s.SendNotification); err != nil {
		// Error
		return
	}
	if err := s.HandleFunc(topicRegisterDevice, rawKey, s.RegisterDevice); err != nil {
		// Error
		return
	}
	session := &Session{
		Type:   SessionChat,
		Key:    rawKey,
		Values: make(map[string]interface{}),
	}
	// store tokens
	// json uses uses map[string]interface{}
	session.Values["tokens"] = make(map[string]interface{})
	filename := crypto.Keccak256Hash(rawKey).String()
	if err := s.StoreSession(filename, session, hex.EncodeToString(crypto.FromECDSA(s.protocolKey))); err != nil {
		// Error
		return
	}

	// response
	resp.Dst = req.Src
	resp.KeySym = session.Key
	resp.Topic = req.Topic
	resp.TTL = whisper.DefaultTTL
	resp.PoW = whisper.DefaultMinimumPoW
	resp.WorkTime = DefaultWorkTime
	resp.Payload = rawRespPayload
}

// RegisterDevice handles device registration requests
func (s *NotificationServer) RegisterDevice(resp *whisper.MessageParams, req *whisper.ReceivedMessage) {
	// request payload
	var reqPayload RegisterDeviceRequest
	if err := json.Unmarshal(req.Payload, &reqPayload); err != nil {
		// Error
		return
	}

	// load chat session
	session, err := s.GetSession(req.SymKeyHash.String(), hex.EncodeToString(crypto.FromECDSA(s.protocolKey)))
	if err != nil {
		// Error
		return
	}

	// register device
	value := session.Values["tokens"]
	// json uses map[string]interface{}
	tokens := value.(map[string]interface{})
	hash := crypto.Keccak256Hash(crypto.FromECDSAPub(req.Src)).String()
	tokens[hash] = reqPayload.DeviceToken

	// store chat session
	if err := s.StoreSession(req.SymKeyHash.String(), session, hex.EncodeToString(crypto.FromECDSA(s.protocolKey))); err != nil {
		// Error
		return
	}

	// response
	resp.Dst = req.Src
	resp.KeySym = session.Key
	resp.Topic = req.Topic
	resp.TTL = whisper.DefaultTTL
	resp.PoW = whisper.DefaultMinimumPoW
	resp.WorkTime = DefaultWorkTime
	//resp.Payload = rawRespPayload - Error Handling
}

// SendNotification handles notification requests
func (s *NotificationServer) SendNotification(resp *whisper.MessageParams, req *whisper.ReceivedMessage) {
	// request payload
	var reqPayload SendNotificationRequest
	if err := json.Unmarshal(req.Payload, &reqPayload); err != nil {
		// Error
		return
	}

	// load chat session
	session, err := s.GetSession(req.SymKeyHash.String(), hex.EncodeToString(crypto.FromECDSA(s.protocolKey)))
	if err != nil {
		// Error
		return
	}

	// gather list of targets
	// json uses map[string]interface{}
	tokens := session.Values["tokens"].(map[string]interface{})
	var targets []string
	requester := crypto.Keccak256Hash(crypto.FromECDSAPub(req.Src)).String()
	for identity, token := range tokens {
		if identity != requester {
			targets = append(targets, token.(string))
		}
	}

	// notify targets
	if err := s.Notify(targets, reqPayload.Data); err != nil {
		// Error
		return
	}

	// response
	resp.Dst = req.Src
	resp.KeySym = session.Key
	resp.Topic = req.Topic
	resp.TTL = whisper.DefaultTTL
	resp.PoW = whisper.DefaultMinimumPoW
	resp.WorkTime = DefaultWorkTime
	//resp.Payload = rawRespPayload - Error Handling
}

// Valid handles the requests validation - not used for now
func Valid(h whisper.HandlerFunc) whisper.HandlerFunc {
	return whisper.HandlerFunc(func(resp *whisper.MessageParams, req *whisper.ReceivedMessage) {
		if req.Src == nil {
			// Error
			return
		}
		h.ServeWhisper(resp, req)
	})
}
