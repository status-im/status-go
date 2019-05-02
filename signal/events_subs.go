package signal

const (
	// EventSubscriptionsData is triggered when there is new data in any of the subscriptions
	EventSubscriptionsData = "subscriptions.data"
	// EventSubscriptionsError is triggered when subscriptions failed to get new data
	EventSubscriptionsError = "subscriptions.error"
)

type SubscriptionDataEvent struct {
	FilterID string        `json:"filter_id"`
	Data     []interface{} `json:"data"`
}

type SubscriptionErrorEvent struct {
	FilterID     string `json:"filter_id"`
	ErrorMessage string `json:"error_message"`
	ErrorCode    int    `json:"error_code,string"`
}

// SendSubscriptionDataEvent
func SendSubscriptionDataEvent(filterID string, data []interface{}) {
	send(EventSubscriptionsData, SubscriptionDataEvent{
		FilterID: filterID,
		Data:     data,
	})
}

// SendSubscriptionErrorEvent
func SendSubscriptionErrorEvent(filterID string, err error, errCode int) {
	send(EventSubscriptionsError, SubscriptionErrorEvent{
		ErrorMessage: err.Error(),
		ErrorCode:    errCode,
	})
}
