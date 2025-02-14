package waku

/*
	#cgo LDFLAGS: -L../third_party/nwaku/build/ -lwaku
	#cgo LDFLAGS: -L../third_party/nwaku -Wl,-rpath,../third_party/nwaku/build/

	#include "../third_party/nwaku/library/libwaku.h"
	#include <stdio.h>
	#include <stdlib.h>

	extern void globalEventCallback(int ret, char* msg, size_t len, void* userData);

	typedef struct {
		int ret;
		char* msg;
		size_t len;
		void* ffiWg;
	} Resp;

	static void* allocResp(void* wg) {
		Resp* r = calloc(1, sizeof(Resp));
		r->ffiWg = wg;
		return r;
	}

	static void freeResp(void* resp) {
		if (resp != NULL) {
			free(resp);
		}
	}

	static char* getMyCharPtr(void* resp) {
		if (resp == NULL) {
			return NULL;
		}
		Resp* m = (Resp*) resp;
		return m->msg;
	}

	static size_t getMyCharLen(void* resp) {
		if (resp == NULL) {
			return 0;
		}
		Resp* m = (Resp*) resp;
		return m->len;
	}

	static int getRet(void* resp) {
		if (resp == NULL) {
			return 0;
		}
		Resp* m = (Resp*) resp;
		return m->ret;
	}

	// resp must be set != NULL in case interest on retrieving data from the callback
	void GoCallback(int ret, char* msg, size_t len, void* resp);

	#define WAKU_CALL(call)                                                        \
	do {                                                                           \
		int ret = call;                                                            \
		if (ret != 0) {                                                            \
			printf("Failed the call to: %s. Returned code: %d\n", #call, ret);     \
			exit(1);                                                               \
		}                                                                          \
	} while (0)

	static void* cGoWakuNew(const char* configJson, void* resp) {
		// We pass NULL because we are not interested in retrieving data from this callback
		void* ret = waku_new(configJson, (WakuCallBack) GoCallback, resp);
		return ret;
	}

	static void cGoWakuStart(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_start(wakuCtx, (WakuCallBack) GoCallback, resp));
	}

	static void cGoWakuStop(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_stop(wakuCtx, (WakuCallBack) GoCallback, resp));
	}

	static void cGoWakuDestroy(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_destroy(wakuCtx, (WakuCallBack) GoCallback, resp));
	}

	static void cGoWakuStartDiscV5(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_start_discv5(wakuCtx, (WakuCallBack) GoCallback, resp));
	}

	static void cGoWakuStopDiscV5(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_stop_discv5(wakuCtx, (WakuCallBack) GoCallback, resp));
	}

	static void cGoWakuVersion(void* wakuCtx, void* resp) {
		WAKU_CALL(waku_version(wakuCtx, (WakuCallBack) GoCallback, resp));
	}

	static void cGoWakuSetEventCallback(void* wakuCtx) {
		// The 'globalEventCallback' Go function is shared amongst all possible Waku instances.

		// Given that the 'globalEventCallback' is shared, we pass again the
		// wakuCtx instance but in this case is needed to pick up the correct method
		// that will handle the event.

		// In other words, for every call the libwaku makes to globalEventCallback,
		// the 'userData' parameter will bring the context of the node that registered
		// that globalEventCallback.

		// This technique is needed because cgo only allows to export Go functions and not methods.

		waku_set_event_callback(wakuCtx, (WakuCallBack) globalEventCallback, wakuCtx);
	}

	static void cGoWakuContentTopic(void* wakuCtx,
							char* appName,
							int appVersion,
							char* contentTopicName,
							char* encoding,
							void* resp) {

		WAKU_CALL( waku_content_topic(wakuCtx,
							appName,
							appVersion,
							contentTopicName,
							encoding,
							(WakuCallBack) GoCallback,
							resp) );
	}

	static void cGoWakuPubsubTopic(void* wakuCtx, char* topicName, void* resp) {
		WAKU_CALL( waku_pubsub_topic(wakuCtx, topicName, (WakuCallBack) GoCallback, resp) );
	}

	static void cGoWakuDefaultPubsubTopic(void* wakuCtx, void* resp) {
		WAKU_CALL (waku_default_pubsub_topic(wakuCtx, (WakuCallBack) GoCallback, resp));
	}

	static void cGoWakuRelayPublish(void* wakuCtx,
                       const char* pubSubTopic,
                       const char* jsonWakuMessage,
                       int timeoutMs,
					   void* resp) {

		WAKU_CALL (waku_relay_publish(wakuCtx,
                       pubSubTopic,
                       jsonWakuMessage,
                       timeoutMs,
                       (WakuCallBack) GoCallback,
                       resp));
	}

	static void cGoWakuRelaySubscribe(void* wakuCtx, char* pubSubTopic, void* resp) {
		WAKU_CALL ( waku_relay_subscribe(wakuCtx,
							pubSubTopic,
							(WakuCallBack) GoCallback,
							resp) );
	}

	static void cGoWakuRelayAddProtectedShard(void* wakuCtx, int clusterId, int shardId, char* publicKey, void* resp) {
		WAKU_CALL ( waku_relay_add_protected_shard(wakuCtx,
							clusterId,
							shardId,
							publicKey,
							(WakuCallBack) GoCallback,
							resp) );
	}

	static void cGoWakuRelayUnsubscribe(void* wakuCtx, char* pubSubTopic, void* resp) {

		WAKU_CALL ( waku_relay_unsubscribe(wakuCtx,
							pubSubTopic,
							(WakuCallBack) GoCallback,
							resp) );
	}

	static void cGoWakuConnect(void* wakuCtx, char* peerMultiAddr, int timeoutMs, void* resp) {
		WAKU_CALL( waku_connect(wakuCtx,
						peerMultiAddr,
						timeoutMs,
						(WakuCallBack) GoCallback,
						resp) );
	}

	static void cGoWakuDialPeer(void* wakuCtx,
									char* peerMultiAddr,
									char* protocol,
									int timeoutMs,
									void* resp) {

		WAKU_CALL( waku_dial_peer(wakuCtx,
						peerMultiAddr,
						protocol,
						timeoutMs,
						(WakuCallBack) GoCallback,
						resp) );
	}

	static void cGoWakuDialPeerById(void* wakuCtx,
									char* peerId,
									char* protocol,
									int timeoutMs,
									void* resp) {

		WAKU_CALL( waku_dial_peer_by_id(wakuCtx,
						peerId,
						protocol,
						timeoutMs,
						(WakuCallBack) GoCallback,
						resp) );
	}

	static void cGoWakuDisconnectPeerById(void* wakuCtx, char* peerId, void* resp) {
		WAKU_CALL( waku_disconnect_peer_by_id(wakuCtx,
						peerId,
						(WakuCallBack) GoCallback,
						resp) );
	}

	static void cGoWakuListenAddresses(void* wakuCtx, void* resp) {
		WAKU_CALL (waku_listen_addresses(wakuCtx, (WakuCallBack) GoCallback, resp) );
	}

	static void cGoWakuGetMyENR(void* ctx, void* resp) {
		WAKU_CALL (waku_get_my_enr(ctx, (WakuCallBack) GoCallback, resp) );
	}

	static void cGoWakuGetMyPeerId(void* ctx, void* resp) {
		WAKU_CALL (waku_get_my_peerid(ctx, (WakuCallBack) GoCallback, resp) );
	}

	static void cGoWakuPingPeer(void* ctx, char* peerAddr, int timeoutMs, void* resp) {
		WAKU_CALL (waku_ping_peer(ctx, peerAddr, timeoutMs, (WakuCallBack) GoCallback, resp) );
	}

	static void cGoWakuGetNumPeersInMesh(void* ctx, char* pubSubTopic, void* resp) {
		WAKU_CALL (waku_relay_get_num_peers_in_mesh(ctx, pubSubTopic, (WakuCallBack) GoCallback, resp) );
	}

	static void cGoWakuGetNumConnectedRelayPeers(void* ctx, char* pubSubTopic, void* resp) {
		WAKU_CALL (waku_relay_get_num_connected_peers(ctx, pubSubTopic, (WakuCallBack) GoCallback, resp) );
	}

	static void cGoWakuGetConnectedPeers(void* wakuCtx, void* resp) {
		WAKU_CALL (waku_get_connected_peers(wakuCtx, (WakuCallBack) GoCallback, resp) );
	}

	static void cGoWakuGetPeerIdsFromPeerStore(void* wakuCtx, void* resp) {
		WAKU_CALL (waku_get_peerids_from_peerstore(wakuCtx, (WakuCallBack) GoCallback, resp) );
	}

	static void cGoWakuLightpushPublish(void* wakuCtx,
					const char* pubSubTopic,
					const char* jsonWakuMessage,
					void* resp) {

		WAKU_CALL (waku_lightpush_publish(wakuCtx,
						pubSubTopic,
						jsonWakuMessage,
						(WakuCallBack) GoCallback,
						resp));
	}

	static void cGoWakuStoreQuery(void* wakuCtx,
					const char* jsonQuery,
					const char* peerAddr,
					int timeoutMs,
					void* resp) {

		WAKU_CALL (waku_store_query(wakuCtx,
									jsonQuery,
									peerAddr,
									timeoutMs,
									(WakuCallBack) GoCallback,
									resp));
	}

	static void cGoWakuPeerExchangeQuery(void* wakuCtx,
								uint64_t numPeers,
								void* resp) {

		WAKU_CALL (waku_peer_exchange_request(wakuCtx,
									numPeers,
									(WakuCallBack) GoCallback,
									resp));
	}

	static void cGoWakuGetPeerIdsByProtocol(void* wakuCtx,
									 const char* protocol,
									 void* resp) {

		WAKU_CALL (waku_get_peerids_by_protocol(wakuCtx,
									protocol,
									(WakuCallBack) GoCallback,
									resp));
	}

	static void cGoWakuDnsDiscovery(void* wakuCtx,
									 const char* entTreeUrl,
									 const char* nameDnsServer,
									 int timeoutMs,
									 void* resp) {

		WAKU_CALL (waku_dns_discovery(wakuCtx,
									entTreeUrl,
									nameDnsServer,
									timeoutMs,
									(WakuCallBack) GoCallback,
									resp));
	}

*/
import "C"
import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pproto "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"github.com/waku-org/waku-go-bindings/waku/common"
	"go.uber.org/zap"
)

const requestTimeout = 30 * time.Second
const MsgChanBufferSize = 1024
const TopicHealthChanBufferSize = 1024
const ConnectionChangeChanBufferSize = 1024

//export GoCallback
func GoCallback(ret C.int, msg *C.char, len C.size_t, resp unsafe.Pointer) {
	if resp != nil {
		m := (*C.Resp)(resp)
		m.ret = ret
		m.msg = msg
		m.len = len
		wg := (*sync.WaitGroup)(m.ffiWg)
		wg.Done()
	}
}

// WakuNode represents an instance of an nwaku node
type WakuNode struct {
	wakuCtx              unsafe.Pointer
	config               *common.WakuConfig
	MsgChan              chan common.Envelope
	TopicHealthChan      chan topicHealth
	ConnectionChangeChan chan connectionChange
	nodeName             string
}

func NewWakuNode(config *common.WakuConfig, nodeName string) (*WakuNode, error) {
	Debug("Creating new WakuNode: %v", nodeName)
	n := &WakuNode{
		config:   config,
		nodeName: nodeName,
	}

	wg := sync.WaitGroup{}

	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	var cJsonConfig = C.CString(string(jsonConfig))
	var resp = C.allocResp(unsafe.Pointer(&wg))

	defer C.free(unsafe.Pointer(cJsonConfig))
	defer C.freeResp(resp)

	if C.getRet(resp) != C.RET_OK {
		errMsg := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		Error("error wakuNew for %s: %v", nodeName, errMsg)
		return nil, errors.New(errMsg)
	}

	wg.Add(1)
	n.wakuCtx = C.cGoWakuNew(cJsonConfig, resp)
	wg.Wait()

	n.MsgChan = make(chan common.Envelope, MsgChanBufferSize)
	n.TopicHealthChan = make(chan topicHealth, TopicHealthChanBufferSize)
	n.ConnectionChangeChan = make(chan connectionChange, ConnectionChangeChanBufferSize)

	C.cGoWakuSetEventCallback(n.wakuCtx)
	registerNode(n)

	Debug("Successfully created WakuNode: %s", nodeName)
	return n, nil
}

// The event callback sends back the node's ctx to know to which
// node is the event being emited for. Since we only have a global
// callback in the go side, We register all the nodes that we create
// so we can later obtain which instance of `WakuNode` is should
// be invoked depending on the ctx received

var nodeRegistry map[unsafe.Pointer]*WakuNode

func init() {
	nodeRegistry = make(map[unsafe.Pointer]*WakuNode)
}

func registerNode(node *WakuNode) {
	_, ok := nodeRegistry[node.wakuCtx]
	if !ok {
		nodeRegistry[node.wakuCtx] = node
	}
}

func unregisterNode(node *WakuNode) {
	delete(nodeRegistry, node.wakuCtx)
}

//export globalEventCallback
func globalEventCallback(callerRet C.int, msg *C.char, len C.size_t, userData unsafe.Pointer) {
	if callerRet == C.RET_OK {
		eventStr := C.GoStringN(msg, C.int(len))
		node, ok := nodeRegistry[userData] // userData contains node's ctx
		if ok {
			node.OnEvent(eventStr)
		}
	} else {
		errMsgField := zap.Skip()
		if len != 0 {
			errMsgField = zap.String("error", C.GoStringN(msg, C.int(len)))
		}
		log.Error("globalEventCallback retCode not ok", zap.Int("retCode", int(callerRet)), errMsgField)
	}
}

type jsonEvent struct {
	EventType string `json:"eventType"`
}

type topicHealth struct {
	PubsubTopic string `json:"pubsubTopic"`
	TopicHealth string `json:"topicHealth"`
}

type connectionChange struct {
	PeerId    peer.ID `json:"peerId"`
	PeerEvent string  `json:"peerEvent"`
}

func (n *WakuNode) OnEvent(eventStr string) {
	jsonEvent := jsonEvent{}
	err := json.Unmarshal([]byte(eventStr), &jsonEvent)
	if err != nil {
		Error("could not unmarshal nwaku event string: %v", err)

		return
	}

	switch jsonEvent.EventType {
	case "message":
		n.parseMessageEvent(eventStr)
	case "relay_topic_health_change":
		n.parseTopicHealthChangeEvent(eventStr)
	case "connection_change":
		n.parseConnectionChangeEvent(eventStr)
	}
}

func (n *WakuNode) parseMessageEvent(eventStr string) {
	envelope, err := common.NewEnvelope(eventStr)
	if err != nil {
		Error("could not parse message %v", err)
	}
	n.MsgChan <- envelope
}

func (n *WakuNode) parseTopicHealthChangeEvent(eventStr string) {

	topicHealth := topicHealth{}
	err := json.Unmarshal([]byte(eventStr), &topicHealth)
	if err != nil {
		Error("could not parse topic health change %v", err)
	}
	n.TopicHealthChan <- topicHealth
}

func (n *WakuNode) parseConnectionChangeEvent(eventStr string) {

	connectionChange := connectionChange{}
	err := json.Unmarshal([]byte(eventStr), &connectionChange)
	if err != nil {
		Error("could not parse connection change %v", err)
	}
	n.ConnectionChangeChan <- connectionChange
}

func (n *WakuNode) GetNumConnectedRelayPeers(optPubsubTopic ...string) (int, error) {

	Debug("Fetching number of connected relay peers for %s", n.nodeName)
	pubsubTopic := ""
	if len(optPubsubTopic) > 0 {
		pubsubTopic = optPubsubTopic[0]
	}

	wg := sync.WaitGroup{}
	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	var cPubsubTopic = C.CString(pubsubTopic)
	defer C.free(unsafe.Pointer(cPubsubTopic))

	wg.Add(1)
	C.cGoWakuGetNumConnectedRelayPeers(n.wakuCtx, cPubsubTopic, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		numPeersStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		numPeers, err := strconv.Atoi(numPeersStr)
		if err != nil {
			Error("Failed to convert relay peer count for %s: %v", n.nodeName, err)
			return 0, err
		}
		Debug("Successfully fetched number of connected relay peers for %s: %d", n.nodeName, numPeers)
		return numPeers, nil
	}

	errMsg := "error GetNumConnectedRelayPeers: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	Error("Failed to get number of connected relay peers for %s: %s", n.nodeName, errMsg)

	return 0, errors.New(errMsg)
}

func (n *WakuNode) DisconnectPeerByID(peerID peer.ID) error {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	var cPeerId = C.CString(peerID.String())
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPeerId))

	wg.Add(1)
	C.cGoWakuDisconnectPeerById(n.wakuCtx, cPeerId, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error DisconnectPeerById: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (n *WakuNode) GetConnectedPeers() (peer.IDSlice, error) {
	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to get connected peers %v", err)
		return nil, err
	}

	Debug("Fetching connected peers for %v", n.nodeName)

	wg := sync.WaitGroup{}
	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuGetConnectedPeers(n.wakuCtx, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		peersStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		if peersStr == "" {
			Debug("No connected peers found for " + n.nodeName)
			return nil, nil
		}

		peerIDs := strings.Split(peersStr, ",")
		var peers peer.IDSlice
		for _, peerID := range peerIDs {
			id, err := peer.Decode(peerID)
			if err != nil {
				Error("Failed to decode peer ID for "+n.nodeName, zap.Error(err))
				return nil, err
			}
			peers = append(peers, id)
		}

		Debug("Successfully fetched connected peers for "+n.nodeName, zap.Int("count", len(peers)))
		return peers, nil
	}

	errMsg := "error GetConnectedPeers: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	Error("Failed to get connected peers for "+n.nodeName, zap.String("error", errMsg))

	return nil, errors.New(errMsg)
}

func (n *WakuNode) RelaySubscribe(pubsubTopic string) error {
	if pubsubTopic == "" {
		return errors.New("pubsub topic is empty")
	}

	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	var cPubsubTopic = C.CString(pubsubTopic)

	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPubsubTopic))

	if n.wakuCtx == nil {
		return errors.New("wakuCtx is nil")
	}

	wg.Add(1)
	C.cGoWakuRelaySubscribe(n.wakuCtx, cPubsubTopic, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		return nil
	}

	errMsg := "error WakuRelaySubscribe: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (n *WakuNode) RelayAddProtectedShard(clusterId uint16, shardId uint16, pubkey *ecdsa.PublicKey) error {
	if pubkey == nil {
		return errors.New("error WakuRelayAddProtectedShard: pubkey can't be nil")
	}

	keyHexStr := hex.EncodeToString(crypto.FromECDSAPub(pubkey))

	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	var cPublicKey = C.CString(keyHexStr)

	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPublicKey))

	if n.wakuCtx == nil {
		return errors.New("wakuCtx is nil")
	}

	wg.Add(1)
	C.cGoWakuRelayAddProtectedShard(n.wakuCtx, C.int(clusterId), C.int(shardId), cPublicKey, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		return nil
	}

	errMsg := "error WakuRelayAddProtectedShard: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (n *WakuNode) RelayUnsubscribe(pubsubTopic string) error {
	if pubsubTopic == "" {
		return errors.New("pubsub topic is empty")
	}

	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	var cPubsubTopic = C.CString(pubsubTopic)

	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPubsubTopic))

	if n.wakuCtx == nil {
		return errors.New("wakuCtx is nil")
	}

	wg.Add(1)
	C.cGoWakuRelayUnsubscribe(n.wakuCtx, cPubsubTopic, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		return nil
	}

	errMsg := "error WakuRelayUnsubscribe: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (n *WakuNode) PeerExchangeRequest(numPeers uint64) (uint64, error) {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuPeerExchangeQuery(n.wakuCtx, C.uint64_t(numPeers), resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		numRecvPeersStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		numRecvPeers, err := strconv.ParseUint(numRecvPeersStr, 10, 64)
		if err != nil {
			return 0, err
		}
		return numRecvPeers, nil
	}

	errMsg := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return 0, errors.New(errMsg)
}

func (n *WakuNode) StartDiscV5() error {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuStartDiscV5(n.wakuCtx, resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error WakuStartDiscV5: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (n *WakuNode) StopDiscV5() error {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuStopDiscV5(n.wakuCtx, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error WakuStopDiscV5: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (n *WakuNode) Version() (string, error) {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuVersion(n.wakuCtx, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		var version = C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		return version, nil
	}

	errMsg := "error WakuVersion: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return "", errors.New(errMsg)
}

func (n *WakuNode) StoreQuery(ctx context.Context, storeRequest *common.StoreQueryRequest, peerInfo peer.AddrInfo) (*common.StoreQueryResponse, error) {
	timeoutMs := getContextTimeoutMilliseconds(ctx)

	b, err := json.Marshal(storeRequest)
	if err != nil {
		return nil, err
	}

	addrs := make([]string, len(peerInfo.Addrs))
	for i, addr := range utils.EncapsulatePeerID(peerInfo.ID, peerInfo.Addrs...) {
		addrs[i] = addr.String()
	}

	var cJsonQuery = C.CString(string(b))
	var cPeerAddr = C.CString(strings.Join(addrs, ","))
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))

	defer C.free(unsafe.Pointer(cJsonQuery))
	defer C.free(unsafe.Pointer(cPeerAddr))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuStoreQuery(n.wakuCtx, cJsonQuery, cPeerAddr, C.int(timeoutMs), resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		jsonResponseStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		storeQueryResponse := common.StoreQueryResponse{}
		err = json.Unmarshal([]byte(jsonResponseStr), &storeQueryResponse)
		if err != nil {
			return nil, err
		}
		return &storeQueryResponse, nil
	}
	errMsg := "error WakuStoreQuery: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return nil, errors.New(errMsg)
}

func (n *WakuNode) RelayPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string) (common.MessageHash, error) {
	timeoutMs := getContextTimeoutMilliseconds(ctx)

	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return common.MessageHash(""), err
	}

	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	var cPubsubTopic = C.CString(pubsubTopic)
	var msg = C.CString(string(jsonMsg))
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPubsubTopic))
	defer C.free(unsafe.Pointer(msg))

	wg.Add(1)
	C.cGoWakuRelayPublish(n.wakuCtx, cPubsubTopic, msg, C.int(timeoutMs), resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		msgHash := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		parsedMsgHash, err := common.ToMessageHash(msgHash)
		if err != nil {
			return common.MessageHash(""), err
		}
		return parsedMsgHash, nil
	}
	errMsg := "WakuRelayPublish: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return common.MessageHash(""), errors.New(errMsg)
}

func (n *WakuNode) DnsDiscovery(ctx context.Context, enrTreeUrl string, nameDnsServer string) ([]multiaddr.Multiaddr, error) {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	var cEnrTree = C.CString(enrTreeUrl)
	var cDnsServer = C.CString(nameDnsServer)
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cEnrTree))
	defer C.free(unsafe.Pointer(cDnsServer))

	timeoutMs := getContextTimeoutMilliseconds(ctx)
	wg.Add(1)
	C.cGoWakuDnsDiscovery(n.wakuCtx, cEnrTree, cDnsServer, C.int(timeoutMs), resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		var addrsRet []multiaddr.Multiaddr
		nodeAddresses := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		addrss := strings.Split(nodeAddresses, ",")
		for _, addr := range addrss {
			addr, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				return nil, err
			}
			addrsRet = append(addrsRet, addr)
		}
		return addrsRet, nil
	}
	errMsg := "error WakuDnsDiscovery: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return nil, errors.New(errMsg)
}

func (n *WakuNode) PingPeer(ctx context.Context, peerInfo peer.AddrInfo) (time.Duration, error) {
	addrs := make([]string, len(peerInfo.Addrs))
	for i, addr := range utils.EncapsulatePeerID(peerInfo.ID, peerInfo.Addrs...) {
		addrs[i] = addr.String()
	}

	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	var cPeerId = C.CString(strings.Join(addrs, ","))
	defer C.free(unsafe.Pointer(cPeerId))

	timeoutMs := getContextTimeoutMilliseconds(ctx)
	wg.Add(1)
	C.cGoWakuPingPeer(n.wakuCtx, cPeerId, C.int(timeoutMs), resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		rttStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		rttInt, err := strconv.ParseInt(rttStr, 10, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(rttInt), nil
	}

	errMsg := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return 0, fmt.Errorf("PingPeer: %s", errMsg)
}

func (n *WakuNode) Start() error {
	Debug("Starting %s", n.nodeName)

	wg := sync.WaitGroup{}
	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuStart(n.wakuCtx, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		Debug("Successfully started %s", n.nodeName)
		return nil
	}

	errMsg := "error WakuStart: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	Error("Failed to start %s: %s", n.nodeName, errMsg)

	return errors.New(errMsg)
}

func (n *WakuNode) Stop() error {

	Debug("Stopping %s", n.nodeName)
	wg := sync.WaitGroup{}
	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuStop(n.wakuCtx, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		Debug("Successfully stopped %s", n.nodeName)
		return nil
	}

	errMsg := "error WakuStop: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	Error("Failed to stop %s: %s", n.nodeName, errMsg)

	return errors.New(errMsg)
}

func (n *WakuNode) Destroy() error {
	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to destroy %v", err)
		return err
	}

	Debug("Destroying %v", n.nodeName)

	wg := sync.WaitGroup{}
	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuDestroy(n.wakuCtx, resp)
	wg.Wait()

	if C.getRet(resp) == C.RET_OK {
		Debug("Successfully destroyed " + n.nodeName)
		return nil
	}

	errMsg := "error WakuDestroy: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	Error("Failed to destroy "+n.nodeName, zap.String("error", errMsg))

	return errors.New(errMsg)
}

func (n *WakuNode) PeerID() (peer.ID, error) {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuGetMyPeerId(n.wakuCtx, resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		peerIdStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		id, err := peer.Decode(peerIdStr)
		if err != nil {
			errMsg := "WakuGetMyPeerId - decoding peerId: %w"
			return "", fmt.Errorf(errMsg, err)
		}
		return id, nil
	}
	errMsg := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return "", errors.New(errMsg)
}

func (n *WakuNode) Connect(ctx context.Context, addr multiaddr.Multiaddr) error {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	var cPeerMultiAddr = C.CString(addr.String())
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPeerMultiAddr))

	timeoutMs := getContextTimeoutMilliseconds(ctx)
	wg.Add(1)
	C.cGoWakuConnect(n.wakuCtx, cPeerMultiAddr, C.int(timeoutMs), resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error WakuConnect: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (n *WakuNode) DialPeerByID(ctx context.Context, peerID peer.ID, protocol libp2pproto.ID) error {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	var cPeerId = C.CString(peerID.String())
	var cProtocol = C.CString(string(protocol))
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPeerId))
	defer C.free(unsafe.Pointer(cProtocol))

	timeoutMs := getContextTimeoutMilliseconds(ctx)
	wg.Add(1)
	C.cGoWakuDialPeerById(n.wakuCtx, cPeerId, cProtocol, C.int(timeoutMs), resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error DialPeerById: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (n *WakuNode) ListenAddresses() ([]multiaddr.Multiaddr, error) {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuListenAddresses(n.wakuCtx, resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		var addrsRet []multiaddr.Multiaddr
		listenAddresses := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		addrss := strings.Split(listenAddresses, ",")
		for _, addr := range addrss {
			addr, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				return nil, err
			}
			addrsRet = append(addrsRet, addr)
		}
		return addrsRet, nil
	}
	errMsg := "error WakuListenAddresses: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return nil, errors.New(errMsg)
}

func (n *WakuNode) ENR() (*enode.Node, error) {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuGetMyENR(n.wakuCtx, resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		enrStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		n, err := enode.Parse(enode.ValidSchemes, enrStr)
		if err != nil {
			return nil, err
		}
		return n, nil
	}
	errMsg := "error WakuGetMyENR: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return nil, errors.New(errMsg)
}

func (n *WakuNode) GetNumPeersInMesh(pubsubTopic string) (int, error) {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	var cPubsubTopic = C.CString(pubsubTopic)
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPubsubTopic))

	wg.Add(1)
	C.cGoWakuGetNumPeersInMesh(n.wakuCtx, cPubsubTopic, resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		numPeersStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		numPeers, err := strconv.Atoi(numPeersStr)
		if err != nil {
			errMsg := "GetNumPeersInMesh - error converting string to int: " + err.Error()
			return 0, errors.New(errMsg)
		}
		return numPeers, nil
	}
	errMsg := "error GetNumPeersInMesh: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return 0, errors.New(errMsg)
}

func (n *WakuNode) GetPeerIDsFromPeerStore() (peer.IDSlice, error) {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	defer C.freeResp(resp)

	wg.Add(1)
	C.cGoWakuGetPeerIdsFromPeerStore(n.wakuCtx, resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		peersStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		if peersStr == "" {
			return nil, nil
		}
		// peersStr contains a comma-separated list of peer ids
		itemsPeerIds := strings.Split(peersStr, ",")

		var peers peer.IDSlice
		for _, peerId := range itemsPeerIds {
			id, err := peer.Decode(peerId)
			if err != nil {
				return nil, fmt.Errorf("GetPeerIdsFromPeerStore - decoding peerId: %w", err)
			}
			peers = append(peers, id)
		}
		return peers, nil
	}
	errMsg := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return nil, fmt.Errorf("GetPeerIdsFromPeerStore: %s", errMsg)
}

func (n *WakuNode) GetPeerIDsByProtocol(protocol libp2pproto.ID) (peer.IDSlice, error) {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	var cProtocol = C.CString(string(protocol))
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cProtocol))

	wg.Add(1)
	C.cGoWakuGetPeerIdsByProtocol(n.wakuCtx, cProtocol, resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		peersStr := C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
		if peersStr == "" {
			return nil, nil
		}
		// peersStr contains a comma-separated list of peer ids
		itemsPeerIds := strings.Split(peersStr, ",")

		var peers peer.IDSlice
		for _, p := range itemsPeerIds {
			id, err := peer.Decode(p)
			if err != nil {
				return nil, fmt.Errorf("GetPeerIdsByProtocol - decoding peerId: %w", err)
			}
			peers = append(peers, id)
		}
		return peers, nil
	}
	errMsg := "error GetPeerIdsByProtocol: " +
		C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return nil, fmt.Errorf("GetPeerIdsByProtocol: %s", errMsg)
}

func (n *WakuNode) DialPeer(ctx context.Context, peerAddr multiaddr.Multiaddr, protocol libp2pproto.ID) error {
	wg := sync.WaitGroup{}

	var resp = C.allocResp(unsafe.Pointer(&wg))
	var cPeerMultiAddr = C.CString(peerAddr.String())
	var cProtocol = C.CString(string(protocol))
	defer C.freeResp(resp)
	defer C.free(unsafe.Pointer(cPeerMultiAddr))
	defer C.free(unsafe.Pointer(cProtocol))

	timeoutMs := getContextTimeoutMilliseconds(ctx)
	wg.Add(1)
	C.cGoWakuDialPeer(n.wakuCtx, cPeerMultiAddr, cProtocol, C.int(timeoutMs), resp)
	wg.Wait()
	if C.getRet(resp) == C.RET_OK {
		return nil
	}
	errMsg := "error DialPeer: " + C.GoStringN(C.getMyCharPtr(resp), C.int(C.getMyCharLen(resp)))
	return errors.New(errMsg)
}

func (n *WakuNode) GetNumConnectedPeers() (int, error) {
	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to get number of connected peers %v", err)
		return 0, err
	}

	Debug("Fetching number of connected peers for %v", n.nodeName)

	peers, err := n.GetConnectedPeers()
	if err != nil {
		Error("Failed to fetch connected peers for %v %v ", n.nodeName, err)
		return 0, err
	}

	numPeers := len(peers)
	Debug("Successfully fetched number of connected peers for "+n.nodeName, zap.Int("count", numPeers))

	return numPeers, nil
}

func getContextTimeoutMilliseconds(ctx context.Context) int {
	deadline, ok := ctx.Deadline()
	if ok {
		return int(time.Until(deadline).Milliseconds())
	}
	return 0
}

func FormatWakuRelayTopic(clusterId uint16, shard uint16) string {
	return fmt.Sprintf("/waku/2/rs/%d/%d", clusterId, shard)
}

func GetFreePortIfNeeded(tcpPort int, discV5UDPPort int) (int, int, error) {
	if tcpPort == 0 {
		for i := 0; i < 10; i++ {
			tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("localhost", "0"))
			if err != nil {
				Warn("unable to resolve tcp addr: %v", zap.Error(err))
				continue
			}
			tcpListener, err := net.ListenTCP("tcp", tcpAddr)
			if err != nil {
				Warn("unable to listen on addr: addr=%v, error=%v", tcpAddr, err)

				continue
			}
			tcpPort = tcpListener.Addr().(*net.TCPAddr).Port
			tcpListener.Close()
			break
		}
		if tcpPort == 0 {
			return -1, -1, errors.New("could not obtain a free TCP port")
		}
	}

	if discV5UDPPort == 0 {
		for i := 0; i < 10; i++ {
			udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("localhost", "0"))
			if err != nil {
				Warn("unable to resolve udp addr: %v", err)
				continue
			}

			udpListener, err := net.ListenUDP("udp", udpAddr)
			if err != nil {
				Warn("unable to listen on addr: addr=%v, error=%v", udpAddr, err)

				continue
			}

			discV5UDPPort = udpListener.LocalAddr().(*net.UDPAddr).Port
			udpListener.Close()
			break
		}
		if discV5UDPPort == 0 {
			return -1, -1, errors.New("could not obtain a free UDP port")
		}
	}

	return tcpPort, discV5UDPPort, nil
}

// Create & start node
func StartWakuNode(nodeName string, customCfg *common.WakuConfig) (*WakuNode, error) {

	Debug("Initializing %s", nodeName)

	var nodeCfg common.WakuConfig
	if customCfg == nil {
		nodeCfg = DefaultWakuConfig
	} else {
		nodeCfg = *customCfg
	}

	Debug("Creating %s", nodeName)
	node, err := NewWakuNode(&nodeCfg, nodeName)
	if err != nil {
		Error("Failed to create %s: %v", nodeName, err)
		return nil, err
	}

	Debug("Starting %s", nodeName)
	if err := node.Start(); err != nil {
		Error("Failed to start %s: %v", nodeName, err)
		return nil, err
	}

	Debug("Successfully started %s", nodeName)
	return node, nil
}

func (n *WakuNode) StopAndDestroy() error {
	if n == nil {
		err := errors.New("waku node is nil")
		Error("Failed to stop and destroy: %v", err)
		return err
	}

	Debug("Stopping %s", n.nodeName)

	err := n.Stop()
	if err != nil {
		Error("Failed to stop %s: %v", n.nodeName, err)
		return err
	}

	Debug("Destroying %s", n.nodeName)

	err = n.Destroy()
	if err != nil {
		Error("Failed to destroy %s: %v", n.nodeName, err)
		return err
	}

	Debug("Successfully stopped and destroyed %s", n.nodeName)
	return nil
}

func (n *WakuNode) ConnectPeer(targetNode *WakuNode) error {

	Debug("Connecting %s to %s", n.nodeName, targetNode.nodeName)

	targetPeerID, err := targetNode.PeerID()
	if err != nil {
		Error("Failed to get PeerID of target node %s: %v", targetNode.nodeName, err)
		return err
	}

	targetAddr, err := targetNode.ListenAddresses()
	if err != nil || len(targetAddr) == 0 {
		Error("Failed to get listen addresses for target node %s: %v", targetNode.nodeName, err)
		return errors.New("target node has no listen addresses")
	}

	Debug("Attempting connection to peer %s", targetPeerID.String())

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	err = n.Connect(ctx, targetAddr[0])
	if err != nil {
		Error("Failed to connect to peer %s: %v", targetPeerID.String(), err)
		return err
	}

	Debug("Successfully connected %s to %s", n.nodeName, targetNode.nodeName)
	return nil
}

func (n *WakuNode) DisconnectPeer(target *WakuNode) error {
	Debug("Disconnecting %s from %s", n.nodeName, target.nodeName)

	targetPeerID, err := target.PeerID()
	if err != nil {
		Error("Failed to get PeerID of target node %s: %v", target.nodeName, err)
		return err
	}

	err = n.DisconnectPeerByID(targetPeerID)
	if err != nil {
		Error("Failed to disconnect peer %s: %v", targetPeerID.String(), err)
		return err
	}

	Debug("Successfully disconnected %s from %s", n.nodeName, target.nodeName)
	return nil
}
