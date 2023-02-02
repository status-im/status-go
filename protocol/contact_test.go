package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

type contactTest struct {
	actualLocalState    ContactRequestState
	expectedLocalState  ContactRequestState
	actualRemoteState   ContactRequestState
	expectedRemoteState ContactRequestState
	expectedAdded       bool
	expectedHasAddedUs  bool
	expectedMutual      bool
}

func (ct contactTest) Contact() Contact {
	return Contact{
		ContactRequestLocalState:  ct.actualLocalState,
		ContactRequestRemoteState: ct.actualRemoteState,
	}
}

func validateContactTest(t *testing.T, contact Contact, tc contactTest, testNum int) {
	failedMessage := fmt.Sprintf("test failed: %d", testNum)
	require.Equal(t, tc.expectedLocalState, contact.ContactRequestLocalState, failedMessage+", contact request local state not matching")
	require.Equal(t, tc.expectedRemoteState, contact.ContactRequestRemoteState, failedMessage+", contact request remote state not matching")

	require.Equal(t, tc.expectedAdded, contact.added(), failedMessage+", added() not matching")
	require.Equal(t, tc.expectedHasAddedUs, contact.hasAddedUs(), failedMessage+", hasAddedUs() not matching")
	require.Equal(t, tc.expectedMutual, contact.mutual(), failedMessage+", mutual() not matching")
}

/*
none/none
sent/none
dismissed/none
none/received
sent/received
dismissed/received
*/

func TestContactContactRequestSent(t *testing.T) {

	clock := uint64(1)

	/* Cases to consider are:
	   Local = none Remote = none
	   Local = none Remote = received
	   Local = sent Remote = none
	   Local = sent Remote = received
	   Local = dismissed Remote = none
	   Local = dismissed Remote = received
	*/

	tests := []contactTest{
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       true,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       true,
			expectedHasAddedUs:  true,
			expectedMutual:      true,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       true,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       true,
			expectedHasAddedUs:  true,
			expectedMutual:      true,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       true,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       true,
			expectedHasAddedUs:  true,
			expectedMutual:      true,
		},
	}

	for testNum, tc := range tests {
		contact := tc.Contact()

		contact.ContactRequestSent(clock)
		validateContactTest(t, contact, tc, testNum+1)

	}
}

func TestContactAcceptContactRequest(t *testing.T) {

	clock := uint64(1)

	tests := []contactTest{
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       true,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       true,
			expectedHasAddedUs:  true,
			expectedMutual:      true,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       true,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       true,
			expectedHasAddedUs:  true,
			expectedMutual:      true,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       true,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       true,
			expectedHasAddedUs:  true,
			expectedMutual:      true,
		},
	}

	for testNum, tc := range tests {
		contact := tc.Contact()

		contact.AcceptContactRequest(clock)
		validateContactTest(t, contact, tc, testNum+1)

	}
}

func TestContactRetractContactRequest(t *testing.T) {

	clock := uint64(1)

	tests := []contactTest{
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
	}

	for testNum, tc := range tests {
		contact := tc.Contact()

		contact.RetractContactRequest(clock)
		validateContactTest(t, contact, tc, testNum+1)

	}
}

func TestContactDismissContactRequest(t *testing.T) {

	clock := uint64(1)

	tests := []contactTest{
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       false,
			expectedHasAddedUs:  true,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       false,
			expectedHasAddedUs:  true,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       false,
			expectedHasAddedUs:  true,
			expectedMutual:      false,
		},
	}

	for testNum, tc := range tests {
		contact := tc.Contact()

		contact.DismissContactRequest(clock)
		validateContactTest(t, contact, tc, testNum+1)

	}
}

func TestContactContactRequestRetracted(t *testing.T) {

	clock := uint64(1)

	tests := []contactTest{
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateNone,
			expectedAdded:       false,
			expectedHasAddedUs:  false,
			expectedMutual:      false,
		},
	}

	for testNum, tc := range tests {
		contact := tc.Contact()

		contact.ContactRequestRetracted(clock)
		validateContactTest(t, contact, tc, testNum+1)

	}
}

func TestContactContactRequestReceived(t *testing.T) {

	clock := uint64(1)

	tests := []contactTest{
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       false,
			expectedHasAddedUs:  true,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       false,
			expectedHasAddedUs:  true,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       true,
			expectedHasAddedUs:  true,
			expectedMutual:      true,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       true,
			expectedHasAddedUs:  true,
			expectedMutual:      true,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       false,
			expectedHasAddedUs:  true,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       false,
			expectedHasAddedUs:  true,
			expectedMutual:      false,
		},
	}

	for testNum, tc := range tests {
		contact := tc.Contact()

		contact.ContactRequestReceived(clock)
		validateContactTest(t, contact, tc, testNum+1)

	}
}

func TestContactContactRequestAccepted(t *testing.T) {

	clock := uint64(1)

	tests := []contactTest{
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       false,
			expectedHasAddedUs:  true,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateNone,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateNone,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       false,
			expectedHasAddedUs:  true,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       true,
			expectedHasAddedUs:  true,
			expectedMutual:      true,
		},
		{
			actualLocalState:    ContactRequestStateSent,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateSent,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       true,
			expectedHasAddedUs:  true,
			expectedMutual:      true,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateNone,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       false,
			expectedHasAddedUs:  true,
			expectedMutual:      false,
		},
		{
			actualLocalState:    ContactRequestStateDismissed,
			actualRemoteState:   ContactRequestStateReceived,
			expectedLocalState:  ContactRequestStateDismissed,
			expectedRemoteState: ContactRequestStateReceived,
			expectedAdded:       false,
			expectedHasAddedUs:  true,
			expectedMutual:      false,
		},
	}

	for testNum, tc := range tests {
		contact := tc.Contact()

		contact.ContactRequestAccepted(clock)
		validateContactTest(t, contact, tc, testNum+1)

	}
}

func TestMarshalContactJSON(t *testing.T) {
	contact := &Contact{}
	id, err := crypto.GenerateKey()
	require.NoError(t, err)
	contact.ID = common.PubkeyToHex(&id.PublicKey)

	encodedContact, err := json.Marshal(contact)

	require.NoError(t, err)
	require.True(t, strings.Contains(string(encodedContact), "compressedKey\":\"zQ"))

}

func TestContactContactRequestPropagatedStateReceivedOutOfDateLocalStateOnTheirSide(t *testing.T) {
	// We receive a message with expected contact request state != our state
	// and clock < our clock, we ping back the user to reach consistency

	c := &Contact{}
	c.ContactRequestLocalState = ContactRequestStateSent
	c.ContactRequestLocalClock = 1

	result := c.ContactRequestPropagatedStateReceived(
		&protobuf.ContactRequestPropagatedState{
			RemoteState: uint64(ContactRequestStateNone),
			RemoteClock: 0,
			LocalState:  uint64(ContactRequestStateNone),
			LocalClock:  1,
		},
	)

	require.True(t, result.sendBackState)

	// if the state is the same, it should not send back a message

	c = &Contact{}
	c.ContactRequestLocalState = ContactRequestStateNone
	c.ContactRequestLocalClock = 1

	result = c.ContactRequestPropagatedStateReceived(
		&protobuf.ContactRequestPropagatedState{
			RemoteState: uint64(ContactRequestStateNone),
			RemoteClock: 0,
			LocalState:  uint64(ContactRequestStateNone),
			LocalClock:  1,
		},
	)

	require.False(t, result.sendBackState)

	// If the clock is the same, it should not send back a message
	c = &Contact{}
	c.ContactRequestLocalState = ContactRequestStateSent
	c.ContactRequestLocalClock = 1

	result = c.ContactRequestPropagatedStateReceived(
		&protobuf.ContactRequestPropagatedState{
			RemoteState: uint64(ContactRequestStateNone),
			RemoteClock: 1,
			LocalState:  uint64(ContactRequestStateNone),
			LocalClock:  1,
		},
	)

	require.False(t, result.sendBackState)

}

func TestContactContactRequestPropagatedStateReceivedOutOfDateLocalStateOnOurSide(t *testing.T) {
	// We receive a message with expected contact request state == none
	// and clock > our clock. We consider this a retraction, unless we are in the dismissed state, since that should be only changed by a trusted device

	c := &Contact{}
	c.ContactRequestLocalState = ContactRequestStateSent
	c.ContactRequestLocalClock = 1

	c.ContactRequestPropagatedStateReceived(
		&protobuf.ContactRequestPropagatedState{
			RemoteState: uint64(ContactRequestStateNone),
			RemoteClock: 2,
			LocalState:  uint64(ContactRequestStateNone),
			LocalClock:  1,
		},
	)

	require.False(t, c.added())

	// But if it's dismissed, we don't change it
	c = &Contact{}
	c.ContactRequestLocalState = ContactRequestStateDismissed
	c.ContactRequestLocalClock = 1

	c.ContactRequestPropagatedStateReceived(
		&protobuf.ContactRequestPropagatedState{
			RemoteState: uint64(ContactRequestStateNone),
			RemoteClock: 1,
			LocalState:  uint64(ContactRequestStateNone),
			LocalClock:  2,
		},
	)

	require.False(t, c.added())
	require.True(t, c.dismissed())

	// or if it's lower clock

	c = &Contact{}
	c.ContactRequestLocalState = ContactRequestStateSent
	c.ContactRequestLocalClock = 1

	c.ContactRequestPropagatedStateReceived(
		&protobuf.ContactRequestPropagatedState{
			RemoteState: uint64(ContactRequestStateNone),
			RemoteClock: 1,
			LocalState:  uint64(ContactRequestStateNone),
			LocalClock:  0,
		},
	)

	require.True(t, c.added())
}

func TestContactContactRequestPropagatedStateReceivedOutOfDateRemoteState(t *testing.T) {
	// We receive a message with newer remote state, we process it as we would for a normal contact request

	c := &Contact{}

	c.ContactRequestLocalState = ContactRequestStateSent
	c.ContactRequestLocalClock = 1

	c.ContactRequestPropagatedStateReceived(
		&protobuf.ContactRequestPropagatedState{
			RemoteState: uint64(ContactRequestStateSent),
			RemoteClock: 1,
			LocalState:  uint64(ContactRequestStateSent),
			LocalClock:  1,
		},
	)

	require.True(t, c.added())
	require.True(t, c.mutual())

	// and retraction
	c = &Contact{}
	c.ContactRequestLocalState = ContactRequestStateSent
	c.ContactRequestLocalClock = 1
	c.ContactRequestRemoteState = ContactRequestStateReceived
	c.ContactRequestRemoteClock = 1

	c.ContactRequestPropagatedStateReceived(
		&protobuf.ContactRequestPropagatedState{
			RemoteState: uint64(ContactRequestStateSent),
			RemoteClock: 1,
			LocalState:  uint64(ContactRequestStateNone),
			LocalClock:  2,
		},
	)

	require.False(t, c.added())
	require.False(t, c.hasAddedUs())
	require.False(t, c.mutual())
}
