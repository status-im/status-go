package requests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteThreadValidate(t *testing.T) {
	for _, request := range []*DeleteThread{nil, {}, {ChatID: "chat"}, {ThreadID: "thread"}, {ChatID: " \n", ThreadID: "thread"}, {ChatID: "chat", ThreadID: "\t"}} {
		require.ErrorIs(t, request.Validate(), ErrDeleteThreadInvalidRequest)
	}
	require.NoError(t, (&DeleteThread{ChatID: "chat", ThreadID: "thread"}).Validate())
}
