package pb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func cf(val string) *ContentFilter {
	return &ContentFilter{
		ContentTopic: val,
	}
}

func TestValidateRequest(t *testing.T) {
	request := HistoryRPC{}
	require.ErrorIs(t, request.ValidateQuery(), errMissingRequestID)
	request.RequestId = "test"
	require.ErrorIs(t, request.ValidateQuery(), errMissingQuery)
	request.Query = &HistoryQuery{
		ContentFilters: []*ContentFilter{
			cf("1"), cf("2"), cf("3"), cf("4"), cf("5"),
			cf("6"), cf("7"), cf("8"), cf("9"), cf("10"),
			cf("11"),
		},
	}
	require.ErrorIs(t, request.ValidateQuery(), errMaxContentFilters)
	request.Query.ContentFilters = []*ContentFilter{cf("a"), cf("")}
	require.ErrorIs(t, request.ValidateQuery(), errEmptyContentTopics)
	request.Query.ContentFilters = []*ContentFilter{cf("a")}
	require.NoError(t, request.ValidateQuery())
}

func TestValidateResponse(t *testing.T) {
	response := HistoryRPC{}
	require.ErrorIs(t, response.ValidateResponse("test"), errMissingRequestID)
	response.RequestId = "test1"
	require.ErrorIs(t, response.ValidateResponse("test"), errRequestIDMismatch)
	response.RequestId = "test"
	response.Response = &HistoryResponse{}
	require.NoError(t, response.ValidateResponse("test"))
}
