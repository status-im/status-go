package notification

import "github.com/NaySoftware/go-fcm"

// Response is a copy of fcm.FcmResponseStatus, good to have it as default type fore notification response
type Response struct {
	Ok           bool
	StatusCode   int
	MulticastID  int64               `json:"multicast_id"`
	Success      int                 `json:"success"`
	Fail         int                 `json:"failure"`
	CanonicalIDs int                 `json:"canonical_ids"`
	Results      []map[string]string `json:"results,omitempty"`
	MsgID        int64               `json:"message_id,omitempty"`
	Err          string              `json:"error,omitempty"`
	RetryAfter   string
}

// FromFCMResponseStatus turns FCM response status into generic Response object
func FromFCMResponseStatus(f *fcm.FcmResponseStatus) *Response {
	return &Response{
		Ok:           f.Ok,
		StatusCode:   f.StatusCode,
		MulticastID:  f.MulticastId,
		Success:      f.Success,
		Fail:         f.Fail,
		CanonicalIDs: f.Canonical_ids,
		Results:      f.Results,
		MsgID:        f.MsgId,
		Err:          f.Err,
		RetryAfter:   f.RetryAfter,
	}
}
