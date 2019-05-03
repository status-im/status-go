package subscriptions

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/status-im/status-go/signal"
	"github.com/stretchr/testify/require"
)

const (
	filterID = "123"
	filterNS = "tst"
)

type mockFilter struct {
	filterID       string
	data           []interface{}
	filterError    error
	uninstalled    bool
	uninstallError error
}

func newMockFilter(filterID string) *mockFilter {
	return &mockFilter{
		filterID: filterID,
	}
}

func (mf *mockFilter) getId() string {
	return mf.filterID
}
func (mf *mockFilter) getChanges() ([]interface{}, error) {
	if mf.filterError != nil {
		err := mf.filterError
		mf.filterError = nil
		return nil, err
	} else {
		data := mf.data
		mf.data = nil
		return data, nil
	}
}

func (mf *mockFilter) uninstall() error {
	mf.uninstalled = true
	return mf.uninstallError
}

func (mf *mockFilter) setData(data ...interface{}) {
	mf.data = data
}

func (mf *mockFilter) setError(err error) {
	mf.data = nil
	mf.filterError = err
}

func TestSubscriptionGetData(t *testing.T) {
	filter := newMockFilter(filterID)

	subs := NewSubscriptions(time.Microsecond)

	subID, _ := subs.Create(filterNS, filter)

	require.Equal(t, string(subID), fmt.Sprintf("%s-%s", filterNS, filterID))

	proceed := make(chan struct{})

	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		defer close(proceed)
		validateFilterData(t, jsonEvent, string(subID), "1", "2", "3", "4")
	})

	filter.setData("1", "2", "3", "4")

	select {
	case <-proceed:
		return
	case <-time.After(time.Second):
		require.NoError(t, errors.New("timeout while waiting for filter results"))
	}

	require.NoError(t, subs.RemoveAll())
	signal.ResetDefaultNodeNotificationHandler()
}

func TestSubscriptionGetError(t *testing.T) {
	filter := newMockFilter(filterID)

	subs := NewSubscriptions(time.Microsecond)

	subID, _ := subs.Create(filterNS, filter)

	require.Equal(t, string(subID), fmt.Sprintf("%s-%s", filterNS, filterID))

	proceed := make(chan struct{})

	expectedError := errors.New("test-error")

	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		defer close(proceed)
		validateFilterError(t, jsonEvent, string(subID), expectedError.Error())
	})

	filter.setError(expectedError)

	select {
	case <-proceed:
		return
	case <-time.After(time.Second):
		require.NoError(t, errors.New("timeout while waiting for filter results"))
	}

	require.NoError(t, subs.RemoveAll())
	signal.ResetDefaultNodeNotificationHandler()
}

func TestSubscriptionRemove(t *testing.T) {
	filter := newMockFilter(filterID)

	subs := NewSubscriptions(time.Microsecond)

	subID, _ := subs.Create(filterNS, filter)

	require.NoError(t, subs.Remove(subID))

	require.True(t, filter.uninstalled)
	require.Equal(t, len(subs.subs), 0)
}

func TestSubscriptionRemoveError(t *testing.T) {
	filter := newMockFilter(filterID)
	filter.uninstallError = errors.New("uninstall-error-1")

	subs := NewSubscriptions(time.Microsecond)

	subID, _ := subs.Create(filterNS, filter)

	require.Equal(t, subs.Remove(subID), filter.uninstallError)

	require.True(t, filter.uninstalled)
	require.Equal(t, len(subs.subs), 0)
}

func TestSubscriptionRemoveAll(t *testing.T) {
	filter0 := newMockFilter(filterID)
	filter1 := newMockFilter(filterID + "1")

	subs := NewSubscriptions(time.Microsecond)
	subs.Create(filterNS, filter0)
	subs.Create(filterNS, filter1)

	require.Equal(t, len(subs.subs), 2)

	require.NoError(t, subs.RemoveAll())

	require.True(t, filter0.uninstalled)
	require.True(t, filter1.uninstalled)

	require.Equal(t, len(subs.subs), 0)
}

func TestSubscriptionRemoveAllError(t *testing.T) {
	filter0 := newMockFilter(filterID)
	filter0.uninstallError = errors.New("error-0")
	filter1 := newMockFilter(filterID + "1")
	filter1.uninstallError = errors.New("error-1")
	filter2 := newMockFilter(filterID + "2")

	subs := NewSubscriptions(time.Microsecond)
	subs.Create(filterNS, filter0)
	subs.Create(filterNS, filter1)
	subs.Create(filterNS, filter2)

	require.Equal(t, len(subs.subs), 3)

	err := subs.RemoveAll()

	require.NotNil(t, err)

	require.True(t, strings.Contains(err.Error(), "error-0"))
	require.True(t, strings.Contains(err.Error(), "error-1"))

	require.True(t, filter0.uninstalled)
	require.True(t, filter1.uninstalled)
	require.True(t, filter2.uninstalled)

	require.Equal(t, len(subs.subs), 0)
}

func validateFilterError(t *testing.T, jsonEvent string, expectedSubID string, expectedErrorMessage string) {
	result := struct {
		Event signal.SubscriptionErrorEvent `json:"event"`
		Type  string                        `json:"type"`
	}{}

	require.NoError(t, json.Unmarshal([]byte(jsonEvent), &result))

	require.Equal(t, signal.EventSubscriptionsError, result.Type)
	require.Equal(t, expectedErrorMessage, result.Event.ErrorMessage)
}

func validateFilterData(t *testing.T, jsonEvent string, expectedSubID string, expectedData ...interface{}) {
	result := struct {
		Event signal.SubscriptionDataEvent `json:"event"`
		Type  string                       `json:"type"`
	}{}

	require.NoError(t, json.Unmarshal([]byte(jsonEvent), &result))

	require.Equal(t, signal.EventSubscriptionsData, result.Type)
	require.Equal(t, expectedData, result.Event.Data)
	require.Equal(t, expectedSubID, result.Event.FilterID)

}
