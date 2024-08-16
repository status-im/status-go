package types

import "encoding/json"

type TelemetryType string

const (
	ProtocolStatsMetric        TelemetryType = "ProtocolStats"
	ReceivedEnvelopeMetric     TelemetryType = "ReceivedEnvelope"
	SentEnvelopeMetric         TelemetryType = "SentEnvelope"
	UpdateEnvelopeMetric       TelemetryType = "UpdateEnvelope"
	ReceivedMessagesMetric     TelemetryType = "ReceivedMessages"
	ErrorSendingEnvelopeMetric TelemetryType = "ErrorSendingEnvelope"
	PeerCountMetric            TelemetryType = "PeerCount"
	PeerConnFailureMetric      TelemetryType = "PeerConnFailure"
)

type TelemetryRequest struct {
	Id            int              `json:"id"`
	TelemetryType TelemetryType    `json:"telemetry_type"`
	TelemetryData *json.RawMessage `json:"telemetry_data"`
}

type PeerCount struct {
	ID            int    `json:"id"`
	CreatedAt     int64  `json:"createdAt"`
	Timestamp     int64  `json:"timestamp"`
	NodeName      string `json:"nodeName"`
	NodeKeyUid    string `json:"nodeKeyUid"`
	PeerID        string `json:"peerId"`
	PeerCount     int    `json:"peerCount"`
	StatusVersion string `json:"statusVersion"`
}

type PeerConnFailure struct {
	ID            int    `json:"id"`
	CreatedAt     int64  `json:"createdAt"`
	Timestamp     int64  `json:"timestamp"`
	NodeName      string `json:"nodeName"`
	NodeKeyUid    string `json:"nodeKeyUid"`
	PeerId        string `json:"peerId"`
	StatusVersion string `json:"statusVersion"`
	FailedPeerId  string `json:"failedPeerId"`
	FailureCount  int    `json:"failureCount"`
}

type SentEnvelope struct {
	ID              int    `json:"id"`
	MessageHash     string `json:"messageHash"`
	SentAt          int64  `json:"sentAt"`
	CreatedAt       int64  `json:"createdAt"`
	PubsubTopic     string `json:"pubsubTopic"`
	Topic           string `json:"topic"`
	SenderKeyUID    string `json:"senderKeyUID"`
	PeerID          string `json:"peerId"`
	NodeName        string `json:"nodeName"`
	ProcessingError string `json:"processingError"`
	PublishMethod   string `json:"publishMethod"`
	StatusVersion   string `json:"statusVersion"`
}

type ErrorSendingEnvelope struct {
	CreatedAt    int64        `json:"createdAt"`
	Error        string       `json:"error"`
	SentEnvelope SentEnvelope `json:"sentEnvelope"`
}

type ReceivedEnvelope struct {
	ID              int    `json:"id"`
	MessageHash     string `json:"messageHash"`
	SentAt          int64  `json:"sentAt"`
	CreatedAt       int64  `json:"createdAt"`
	PubsubTopic     string `json:"pubsubTopic"`
	Topic           string `json:"topic"`
	ReceiverKeyUID  string `json:"receiverKeyUID"`
	PeerID          string `json:"peerId"`
	NodeName        string `json:"nodeName"`
	ProcessingError string `json:"processingError"`
	StatusVersion   string `json:"statusVersion"`
}

type Metric struct {
	TotalIn  int64   `json:"totalIn"`
	TotalOut int64   `json:"totalOut"`
	RateIn   float64 `json:"rateIn"`
	RateOut  float64 `json:"rateOut"`
}

type ProtocolStats struct {
	PeerID          string `json:"hostID"`
	Relay           Metric `json:"relay"`
	Store           Metric `json:"store"`
	FilterPush      Metric `json:"filter-push"`
	FilterSubscribe Metric `json:"filter-subscribe"`
	Lightpush       Metric `json:"lightpush"`
}

type ReceivedMessage struct {
	ID             int    `json:"id"`
	ChatID         string `json:"chatId"`
	MessageHash    string `json:"messageHash"`
	MessageID      string `json:"messageId"`
	MessageType    string `json:"messageType"`
	MessageSize    int    `json:"messageSize"`
	ReceiverKeyUID string `json:"receiverKeyUID"`
	PeerID         string `json:"peerId"`
	NodeName       string `json:"nodeName"`
	SentAt         int64  `json:"sentAt"`
	Topic          string `json:"topic"`
	PubsubTopic    string `json:"pubsubTopic"`
	CreatedAt      int64  `json:"createdAt"`
	StatusVersion  string `json:"statusVersion"`
}
