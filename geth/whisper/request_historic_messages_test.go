package whisper

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/stretchr/testify/require"
)

func TestParseStringParams(t *testing.T) {
	testCases := []struct {
		name    string
		data    string
		request historicMessagesRequest
		err     error
	}{
		{
			"successful parsing of valid data",
			`[{
				"enode": "enode://0f51d75c9469de0852571c4618fe151265d4930ea35f968eb1a12e69c12f7cbabed856a12b31268a825ca2c9bafa47ef665b1b17be1ab71de83338c4b7439b24@127.0.0.1:30303",
				"topic": "0xaabb11ee",
				"symKeyID": "7963bd35a4534f773aee33bd0aec2d175a9d8d104fc020eb0769b05adeb6dda2",
				"from": 1612505820,
				"to": 1612515820
			}]`,
			historicMessagesRequest{
				Peer:     mustGetPeer("enode://0f51d75c9469de0852571c4618fe151265d4930ea35f968eb1a12e69c12f7cbabed856a12b31268a825ca2c9bafa47ef665b1b17be1ab71de83338c4b7439b24@127.0.0.1:30303"),
				Topic:    whisperv5.BytesToTopic([]byte("0xaabb11ee")),
				SymKeyID: "7963bd35a4534f773aee33bd0aec2d175a9d8d104fc020eb0769b05adeb6dda2",
				TimeLow:  1612505820,
				TimeUp:   1612515820,
			},
			nil,
		},
		{
			"invalid enode",
			`[{
				"enode": "invalid-enode"
			}]`,
			historicMessagesRequest{},
			errors.New("enode must be a string and have a valid format: invalid URL scheme, want \"enode\""),
		},
		{
			"topic is required",
			`[{
				"enode": "enode://0f51d75c9469de0852571c4618fe151265d4930ea35f968eb1a12e69c12f7cbabed856a12b31268a825ca2c9bafa47ef665b1b17be1ab71de83338c4b7439b24@127.0.0.1:30303"
			}]`,
			historicMessagesRequest{},
			errors.New("topic value is required"),
		},
		{
			"symKeyID is required",
			`[{
				"enode": "enode://0f51d75c9469de0852571c4618fe151265d4930ea35f968eb1a12e69c12f7cbabed856a12b31268a825ca2c9bafa47ef665b1b17be1ab71de83338c4b7439b24@127.0.0.1:30303",
				"topic": "0xaabb11ee"
			}]`,
			historicMessagesRequest{},
			errors.New("symKeyID value is required"),
		},
		{
			"test default TimeLow and TimeUp",
			`[{
				"enode": "enode://0f51d75c9469de0852571c4618fe151265d4930ea35f968eb1a12e69c12f7cbabed856a12b31268a825ca2c9bafa47ef665b1b17be1ab71de83338c4b7439b24@127.0.0.1:30303",
				"topic": "0xaabb11ee",
				"symKeyID": "7963bd35a4534f773aee33bd0aec2d175a9d8d104fc020eb0769b05adeb6dda2"
			}]`,
			historicMessagesRequest{
				Peer:     mustGetPeer("enode://0f51d75c9469de0852571c4618fe151265d4930ea35f968eb1a12e69c12f7cbabed856a12b31268a825ca2c9bafa47ef665b1b17be1ab71de83338c4b7439b24@127.0.0.1:30303"),
				Topic:    whisperv5.BytesToTopic([]byte("0xaabb11ee")),
				SymKeyID: "7963bd35a4534f773aee33bd0aec2d175a9d8d104fc020eb0769b05adeb6dda2",
				TimeLow:  uint32(time.Now().Add(-24 * time.Hour).Unix()),
				TimeUp:   uint32(time.Now().Unix()),
			},
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var args []interface{}
			require.NoError(t, json.Unmarshal([]byte(tc.data), &args))

			request, err := parseArgs(args)
			if err != nil {
				require.EqualError(t, err, tc.err.Error())
			} else {
				require.Equal(t, tc.request, request)
			}
		})
	}
}

func TestParseMapParams(t *testing.T) {
	params := map[string]interface{}{
		"enode":    "enode://0f51d75c9469de0852571c4618fe151265d4930ea35f968eb1a12e69c12f7cbabed856a12b31268a825ca2c9bafa47ef665b1b17be1ab71de83338c4b7439b24@127.0.0.1:30303",
		"topic":    whisperv5.BytesToTopic([]byte("test-topic")),
		"symKeyID": "7963bd35a4534f773aee33bd0aec2d175a9d8d104fc020eb0769b05adeb6dda2",
		"from":     1612505820,
		"to":       1612515820,
	}
	request, err := parseArgs([]interface{}{params})
	require.NoError(t, err)
	require.Equal(
		t,
		historicMessagesRequest{
			Peer:     mustGetPeer("enode://0f51d75c9469de0852571c4618fe151265d4930ea35f968eb1a12e69c12f7cbabed856a12b31268a825ca2c9bafa47ef665b1b17be1ab71de83338c4b7439b24@127.0.0.1:30303"),
			Topic:    whisperv5.BytesToTopic([]byte("test-topic")),
			SymKeyID: "7963bd35a4534f773aee33bd0aec2d175a9d8d104fc020eb0769b05adeb6dda2",
			TimeLow:  1612505820,
			TimeUp:   1612515820,
		},
		request,
	)
}

func mustGetPeer(enode string) []byte {
	node, err := discover.ParseNode(enode)
	if err != nil {
		panic(err)
	}
	return node.ID[:]
}
