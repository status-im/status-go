package waku

import (
	"fmt"
	"time"

	"github.com/waku-org/waku-go-bindings/waku/common"
	"google.golang.org/protobuf/proto"
)

var DefaultWakuConfig common.WakuConfig
var DefaultStoreQueryRequest common.StoreQueryRequest
var DEFAULT_CLUSTER_ID = uint16(16)

func init() {

	DefaultWakuConfig = common.WakuConfig{
		Relay:           false,
		LogLevel:        "DEBUG",
		Discv5Discovery: true,
		ClusterID:       16,
		Shards:          []uint16{64},
		PeerExchange:    false,
		Store:           false,
		Filter:          false,
		Lightpush:       false,
		Discv5UdpPort:   0,
		TcpPort:         0,
	}

	DefaultStoreQueryRequest = common.StoreQueryRequest{
		IncludeData:       true,
		ContentTopics:     &[]string{"test-content-topic"},
		PaginationLimit:   proto.Uint64(uint64(50)),
		PaginationForward: true,
		TimeStart:         proto.Int64(time.Now().Add(-5 * time.Minute).UnixNano()), // 5 mins before now
	}
}

const ConnectPeerTimeout = 10 * time.Second //default timeout for node to connect to another node
const DefaultTimeOut = 3 * time.Second

var DefaultPubsubTopic = "/waku/2/rs/16/64"
var DefaultContentTopic = "/test/1/default/proto"
var (
	MinPort = 1024  // Minimum allowable port (exported)
	MaxPort = 65535 // Maximum allowable port (exported)
)

var SAMPLE_INPUTS = []struct {
	Description string
	Value       string
}{
	{"A simple string", "Hello World!"},
	{"An integer", "1234567890"},
	{"A dictionary", `{"key": "value"}`},
	{"Chinese characters", "这是一些中文"},
	{"Emojis", "🚀🌟✨"},
	{"Lorem ipsum text", "Lorem ipsum dolor sit amet"},
	{"HTML content", "<html><body>Hello</body></html>"},
	{"Cyrillic characters", "\u041f\u0440\u0438\u0432\u0435\u0442"},
	{"Base64 encoded string", "Base64==dGVzdA=="},
	{"Binary data", "d29ya2luZyB3aXRoIGJpbmFyeSBkYXRh: \x50\x51"},
	{"Special characters with whitespace", "\t\nSpecial\tCharacters\n"},
	{"Boolean false as a string", "False"},
	{"A float number", "3.1415926535"},
	{"A list", "[1, 2, 3, 4, 5]"},
	{"Hexadecimal number as a string", "0xDEADBEEF"},
	{"Email format", "user@example.com"},
	{"URL format", "http://example.com"},
	{"Date and time in ISO format", "2023-11-01T12:00:00Z"},
	{"String with escaped quotes", `"Escaped" \"quotes\"`},
	{"A regular expression", "Regular expression: ^[a-z0-9_-]{3,16}$"},
	{"A very long string", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
	{"A JSON string", `{"name": "John", "age": 30, "city": "New York"}`},
	{"A Unix path", "/usr/local/bin"},
	{"A Windows path", "C:\\Windows\\System32"},
	{"An SQL query", "SELECT * FROM users WHERE id = 1;"},
	{"JavaScript code snippet", "function test() { console.log('Hello World'); }"},
	{"A CSS snippet", "body { background-color: #fff; }"},
	{"A Python one-liner", "print('Hello World')"},
	{"An IP address", "192.168.1.1"},
	{"A domain name", "www.example.com"},
	{"A user agent string", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"},
	{"A credit card number", "1234-5678-9012-3456"},
	{"A phone number", "+1234567890"},
	{"A UUID", "123e4567-e89b-12d3-a456-426614174000"},
	{"A hashtag", "#helloWorld"},
	{"A Twitter handle", "@username"},
	{"A password", "P@ssw0rd!"},
	{"A date in common format", "01/11/2023"},
	{"A time string", "12:00:00"},
	{"A mathematical equation", "E = mc^2"},
}

var PUBSUB_TOPICS_STORE = []string{

	fmt.Sprintf("/waku/2/rs/%d/0", DEFAULT_CLUSTER_ID),
	fmt.Sprintf("/waku/2/rs/%d/1", DEFAULT_CLUSTER_ID),
	fmt.Sprintf("/waku/2/rs/%d/2", DEFAULT_CLUSTER_ID),
	fmt.Sprintf("/waku/2/rs/%d/3", DEFAULT_CLUSTER_ID),
	fmt.Sprintf("/waku/2/rs/%d/4", DEFAULT_CLUSTER_ID),
	fmt.Sprintf("/waku/2/rs/%d/5", DEFAULT_CLUSTER_ID),
	fmt.Sprintf("/waku/2/rs/%d/6", DEFAULT_CLUSTER_ID),
	fmt.Sprintf("/waku/2/rs/%d/7", DEFAULT_CLUSTER_ID),
	fmt.Sprintf("/waku/2/rs/%d/8", DEFAULT_CLUSTER_ID),
}

var CONTENT_TOPICS_DIFFERENT_SHARDS = []string{
	"/myapp/1/latest/proto",
	"/waku/2/content/test.js",
	"/app/22/sometopic/someencoding",
	"/toychat/2/huilong/proto",
	"/statusim/1/community/cbor",
	"/app/27/sometopic/someencoding",
	"/app/29/sometopic/someencoding",
	"/app/20/sometopic/someencoding",
}
