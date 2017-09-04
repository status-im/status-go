package notificationserver

type (

	// TODO(rgerades)
	// Error Handling - responses were added in order to add error fields

	// ServerDiscoveryResponse payload of the server discovery response
	// ServerID will be used to identify the node among the notification
	// nodes that share the protocol identity
	ServerDiscoveryResponse struct {
		ServerID string `json:"server"`
	}

	// ServerAcceptanceRequest payload of the server acceptance request
	// Only the server with the specific ServerID will process the request
	// The client will choose the node which will serve him without knowing
	// any details of the node other than the ServerID
	ServerAcceptanceRequest struct {
		ServerID string `json:"server"`
	}

	// ServerAcceptanceResponse payload of the server acceptance response
	// Key corresponds to the client session key. It will be used to trigger
	// options of the client session. (Ex: create new chat)
	ServerAcceptanceResponse struct {
		Key string `json:"key"`
	}

	// NewChatSessionResponse payload of chat creation response
	// Key corresponds to the chat session key. It will be used to trigger
	// options of the chat session. (Ex: send notification)
	// All elements of the chat will have access to this key.
	NewChatSessionResponse struct {
		Key string `json:"key"`
	}

	// RegisterDeviceRequest payload of the device registration request
	// The DeviceToken corresponds to the unique firebase token that each
	// device pocesses. Once the devices have access to a chat they must
	// register their firebase token.
	RegisterDeviceRequest struct {
		DeviceToken string `json:"device"`
	}

	// RegisterDeviceResponse payload of the device registration response
	RegisterDeviceResponse struct {
	}

	// SendNotificationRequest payload of the notification request
	// Data represents the data attached to the notification
	SendNotificationRequest struct {
		Data string `json:"data"`
	}

	// SendNotificationResponse payload of the notification request response
	SendNotificationResponse struct {
	}
)
