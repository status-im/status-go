package requests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFollowThreadValidate(t *testing.T) {
	for _, request := range []*FollowThread{nil, {}, {ChatID: "chat"}, {ThreadID: "thread"}, {ChatID: " \n", ThreadID: "thread"}, {ChatID: "chat", ThreadID: "\t"}} {
		require.ErrorIs(t, request.Validate(), ErrFollowThreadInvalidRequest)
	}
	require.NoError(t, (&FollowThread{ChatID: "chat", ThreadID: "thread"}).Validate())
}

func TestUnfollowThreadValidate(t *testing.T) {
	for _, request := range []*UnfollowThread{nil, {}, {ChatID: "chat"}, {ThreadID: "thread"}, {ChatID: " \n", ThreadID: "thread"}, {ChatID: "chat", ThreadID: "\t"}} {
		require.ErrorIs(t, request.Validate(), ErrFollowThreadInvalidRequest)
	}
	require.NoError(t, (&UnfollowThread{ChatID: "chat", ThreadID: "thread"}).Validate())
}
