package relay

type publishParameters struct {
	pubsubTopic string
}

// PublishOption is the type of options accepted when publishing WakuMessages
type PublishOption func(*publishParameters)

// WithPubSubTopic is used to specify the pubsub topic on which a WakuMessage will be broadcasted
func WithPubSubTopic(pubsubTopic string) PublishOption {
	return func(params *publishParameters) {
		params.pubsubTopic = pubsubTopic
	}
}

// WithDefaultPubsubTopic is used to indicate that the message should be broadcasted in the default pubsub topic
func WithDefaultPubsubTopic() PublishOption {
	return func(params *publishParameters) {
		params.pubsubTopic = DefaultWakuTopic
	}
}
