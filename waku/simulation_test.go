// Copyright 2019 The Waku Library Authors.
//
// The Waku library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Waku library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty off
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Waku library. If not, see <http://www.gnu.org/licenses/>.
//
// This software uses the go-ethereum library, which is licensed
// under the GNU Lesser General Public Library, version 3 or any later.

package waku

import (
	"bytes"
	"crypto/ecdsa"
	mrand "math/rand"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"

	"github.com/status-im/status-go/waku/common"
)

var keys = []string{
	"d49dcf37238dc8a7aac57dc61b9fee68f0a97f062968978b9fafa7d1033d03a9",
	"73fd6143c48e80ed3c56ea159fe7494a0b6b393a392227b422f4c3e8f1b54f98",
	"119dd32adb1daa7a4c7bf77f847fb28730785aa92947edf42fdd997b54de40dc",
	"deeda8709dea935bb772248a3144dea449ffcc13e8e5a1fd4ef20ce4e9c87837",
	"5bd208a079633befa349441bdfdc4d85ba9bd56081525008380a63ac38a407cf",
	"1d27fb4912002d58a2a42a50c97edb05c1b3dffc665dbaa42df1fe8d3d95c9b5",
	"15def52800c9d6b8ca6f3066b7767a76afc7b611786c1276165fbc61636afb68",
	"51be6ab4b2dc89f251ff2ace10f3c1cc65d6855f3e083f91f6ff8efdfd28b48c",
	"ef1ef7441bf3c6419b162f05da6037474664f198b58db7315a6f4de52414b4a0",
	"09bdf6985aabc696dc1fbeb5381aebd7a6421727343872eb2fadfc6d82486fd9",
	"15d811bf2e01f99a224cdc91d0cf76cea08e8c67905c16fee9725c9be71185c4",
	"2f83e45cf1baaea779789f755b7da72d8857aeebff19362dd9af31d3c9d14620",
	"73f04e34ac6532b19c2aae8f8e52f38df1ac8f5cd10369f92325b9b0494b0590",
	"1e2e07b69e5025537fb73770f483dc8d64f84ae3403775ef61cd36e3faf162c1",
	"8963d9bbb3911aac6d30388c786756b1c423c4fbbc95d1f96ddbddf39809e43a",
	"0422da85abc48249270b45d8de38a4cc3c02032ede1fcf0864a51092d58a2f1f",
	"8ae5c15b0e8c7cade201fdc149831aa9b11ff626a7ffd27188886cc108ad0fa8",
	"acd8f5a71d4aecfcb9ad00d32aa4bcf2a602939b6a9dd071bab443154184f805",
	"a285a922125a7481600782ad69debfbcdb0316c1e97c267aff29ef50001ec045",
	"28fd4eee78c6cd4bf78f39f8ab30c32c67c24a6223baa40e6f9c9a0e1de7cef5",
	"c5cca0c9e6f043b288c6f1aef448ab59132dab3e453671af5d0752961f013fc7",
	"46df99b051838cb6f8d1b73f232af516886bd8c4d0ee07af9a0a033c391380fd",
	"c6a06a53cbaadbb432884f36155c8f3244e244881b5ee3e92e974cfa166d793f",
	"783b90c75c63dc72e2f8d11b6f1b4de54d63825330ec76ee8db34f06b38ea211",
	"9450038f10ca2c097a8013e5121b36b422b95b04892232f930a29292d9935611",
	"e215e6246ed1cfdcf7310d4d8cdbe370f0d6a8371e4eb1089e2ae05c0e1bc10f",
	"487110939ed9d64ebbc1f300adeab358bc58875faf4ca64990fbd7fe03b78f2b",
	"824a70ea76ac81366da1d4f4ac39de851c8ac49dca456bb3f0a186ceefa269a5",
	"ba8f34fa40945560d1006a328fe70c42e35cc3d1017e72d26864cd0d1b150f15",
	"30a5dfcfd144997f428901ea88a43c8d176b19c79dde54cc58eea001aa3d246c",
	"de59f7183aca39aa245ce66a05245fecfc7e2c75884184b52b27734a4a58efa2",
	"92629e2ff5f0cb4f5f08fffe0f64492024d36f045b901efb271674b801095c5a",
	"7184c1701569e3a4c4d2ddce691edd983b81e42e09196d332e1ae2f1e062cff4",
}

type TestData struct {
	started int64
	counter [NumNodes]int
	mutex   sync.RWMutex
}

type TestNode struct {
	waku    *Waku
	id      *ecdsa.PrivateKey
	server  *p2p.Server
	filerID string
}

const NumNodes = 8 // must not exceed the number of keys (32)

var result TestData
var nodes [NumNodes]*TestNode
var sharedKey = hexutil.MustDecode("0x03ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd31")
var wrongKey = hexutil.MustDecode("0xf91156714d7ec88d3edc1c652c2181dbb3044e8771c683f3b30d33c12b986b11")
var sharedTopic = common.TopicType{0xF, 0x1, 0x2, 0}
var wrongTopic = common.TopicType{0, 0, 0, 0}
var expectedMessage = []byte("per aspera ad astra")
var unexpectedMessage = []byte("per rectum ad astra")
var masterBloomFilter []byte
var masterPow = 0.00000001
var round = 1

func TestSimulationBloomFilter(t *testing.T) {
	// create a chain of waku nodes,
	// installs the filters with shared (predefined) parameters
	initializeBloomFilterMode(t)

	// each node sends one random (not decryptable) message
	for i := 0; i < NumNodes; i++ {
		sendMsg(t, false, i)
	}

	// node #0 sends one expected (decryptable) message
	sendMsg(t, true, 0)

	// check if each node have received and decrypted exactly one message
	checkPropagation(t, true)

	// check if Status message was correctly decoded
	checkBloomFilterExchange(t)
	checkPowExchange(t)

	// send new pow and bloom exchange messages
	resetParams()

	// node #1 sends one expected (decryptable) message
	sendMsg(t, true, 1)

	// check if each node (except node #0) have received and decrypted exactly one message
	checkPropagation(t, false)

	// check if corresponding protocol-level messages were correctly decoded
	checkPowExchangeForNodeZero(t)
	checkBloomFilterExchange(t)

	stopServers()
}

func resetParams() {
	// change pow only for node zero
	masterPow = 7777777.0
	_ = nodes[0].waku.SetMinimumPoW(masterPow, true)

	// change bloom for all nodes
	masterBloomFilter = sharedTopic.ToBloom()
	for i := 0; i < NumNodes; i++ {
		_ = nodes[i].waku.SetBloomFilter(masterBloomFilter)
	}

	round++
}

func initBloom(t *testing.T) {
	masterBloomFilter = make([]byte, common.BloomFilterSize)
	_, err := mrand.Read(masterBloomFilter) // nolint: gosec
	if err != nil {
		t.Fatalf("rand failed: %s.", err)
	}

	msgBloom := sharedTopic.ToBloom()
	masterBloomFilter = addBloom(masterBloomFilter, msgBloom)
	for i := 0; i < 32; i++ {
		masterBloomFilter[i] = 0xFF
	}

	if !common.BloomFilterMatch(masterBloomFilter, msgBloom) {
		t.Fatalf("bloom mismatch on initBloom.")
	}
}

func initializeBloomFilterMode(t *testing.T) {
	initBloom(t)

	var err error

	for i := 0; i < NumNodes; i++ {
		var node TestNode
		b := make([]byte, common.BloomFilterSize)
		copy(b, masterBloomFilter)
		config := DefaultConfig
		config.BloomFilterMode = true
		node.waku = New(&config, nil)
		_ = node.waku.SetMinimumPoW(masterPow, false)
		_ = node.waku.SetBloomFilter(b)
		if !bytes.Equal(node.waku.BloomFilter(), masterBloomFilter) {
			t.Fatalf("bloom mismatch on init.")
		}
		_ = node.waku.Start()
		topics := make([]common.TopicType, 0)
		topics = append(topics, sharedTopic)
		f := common.Filter{KeySym: sharedKey, Messages: common.NewMemoryMessageStore()}
		f.Topics = [][]byte{topics[0][:]}
		node.filerID, err = node.waku.Subscribe(&f)
		if err != nil {
			t.Fatalf("failed to install the filter: %s.", err)
		}
		node.id, err = crypto.HexToECDSA(keys[i])
		if err != nil {
			t.Fatalf("failed convert the key: %s.", keys[i])
		}
		name := gethcommon.MakeName("waku-go", "2.0")

		node.server = &p2p.Server{
			Config: p2p.Config{
				PrivateKey: node.id,
				MaxPeers:   NumNodes/2 + 1,
				Name:       name,
				Protocols:  node.waku.Protocols(),
				ListenAddr: "127.0.0.1:0",
				NAT:        nat.Any(),
			},
		}

		go startServer(t, node.server) // nolint: staticcheck, govet

		nodes[i] = &node
	}

	waitForServersToStart(t)

	for i := 0; i < NumNodes; i++ {
		for j := 0; j < i; j++ {
			peerNodeID := nodes[j].id
			address, _ := net.ResolveTCPAddr("tcp", nodes[j].server.ListenAddr)
			peer := enode.NewV4(&peerNodeID.PublicKey, address.IP, address.Port, address.Port)
			nodes[i].server.AddPeer(peer)
		}
	}
}

func startServer(t *testing.T, s *p2p.Server) {
	err := s.Start()
	if err != nil {
		t.Fatalf("failed to start the first server. err: %v", err) // nolint: staticcheck
	}

	atomic.AddInt64(&result.started, 1)
}

func stopServers() {
	for i := 0; i < NumNodes; i++ {
		n := nodes[i]
		if n != nil {
			_ = n.waku.Unsubscribe(n.filerID)
			_ = n.waku.Stop()
			n.server.Stop()
		}
	}
}

func checkPropagation(t *testing.T, includingNodeZero bool) {
	if t.Failed() {
		return
	}

	// (cycle * iterations) should not exceed 50 seconds, since TTL=50
	const cycle = 200 // time in milliseconds
	const iterations = 250

	first := 0
	if !includingNodeZero {
		first = 1
	}

	for j := 0; j < iterations; j++ {
		for i := first; i < NumNodes; i++ {
			f := nodes[i].waku.GetFilter(nodes[i].filerID)
			if f == nil {
				t.Fatalf("failed to get filterId %s from node %d, round %d.", nodes[i].filerID, i, round)
			}

			mail := f.Retrieve()
			validateMail(t, i, mail)

			if isTestComplete() {
				checkTestStatus()
				return
			}
		}

		checkTestStatus()
		time.Sleep(cycle * time.Millisecond)
	}

	if !includingNodeZero {
		f := nodes[0].waku.GetFilter(nodes[0].filerID)
		if f != nil {
			t.Fatalf("node zero received a message with low PoW.")
		}
	}

	t.Fatalf("Test was not complete (%d round): timeout %d seconds. nodes=%v", round, iterations*cycle/1000, nodes)
}

func validateMail(t *testing.T, index int, mail []*common.ReceivedMessage) {
	var cnt int
	for _, m := range mail {
		if bytes.Equal(m.Payload, expectedMessage) {
			cnt++
		}
	}

	if cnt == 0 {
		// no messages received yet: nothing is wrong
		return
	}
	if cnt > 1 {
		t.Fatalf("node %d received %d.", index, cnt)
	}

	if cnt == 1 {
		result.mutex.Lock()
		defer result.mutex.Unlock()
		result.counter[index] += cnt
		if result.counter[index] > 1 {
			t.Fatalf("node %d accumulated %d.", index, result.counter[index])
		}
	}
}

func checkTestStatus() {
	var cnt int
	var arr [NumNodes]int

	for i := 0; i < NumNodes; i++ {
		arr[i] = nodes[i].server.PeerCount()
		envelopes := nodes[i].waku.Envelopes()
		if len(envelopes) >= NumNodes {
			cnt++
		}
	}
}

func isTestComplete() bool {
	result.mutex.RLock()
	defer result.mutex.RUnlock()

	for i := 0; i < NumNodes; i++ {
		if result.counter[i] < 1 {
			return false
		}
	}

	for i := 0; i < NumNodes; i++ {
		envelopes := nodes[i].waku.Envelopes()
		if len(envelopes) < NumNodes+1 {
			return false
		}
	}

	return true
}

func sendMsg(t *testing.T, expected bool, id int) {
	if t.Failed() {
		return
	}

	opt := common.MessageParams{KeySym: sharedKey, Topic: sharedTopic, Payload: expectedMessage, PoW: 0.00000001, WorkTime: 1}
	if !expected {
		opt.KeySym = wrongKey
		opt.Topic = wrongTopic
		opt.Payload = unexpectedMessage
		opt.Payload[0] = byte(id)
	}

	msg, err := common.NewSentMessage(&opt)
	if err != nil {
		t.Fatalf("failed to create new message: %s.", err)
	}
	envelope, err := msg.Wrap(&opt, time.Now())
	if err != nil {
		t.Fatalf("failed to seal message: %s", err)
	}

	err = nodes[id].waku.Send(envelope)
	if err != nil {
		t.Fatalf("failed to send message: %s", err)
	}
}

func checkPowExchangeForNodeZero(t *testing.T) {
	const iterations = 200
	for j := 0; j < iterations; j++ {
		lastCycle := j == iterations-1
		ok := checkPowExchangeForNodeZeroOnce(t, lastCycle)
		if ok {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func checkPowExchangeForNodeZeroOnce(t *testing.T, mustPass bool) bool {
	cnt := 0
	for i, node := range nodes {
		for protocol := range node.waku.peers {
			if protocol.EnodeID() == nodes[0].server.Self().ID() {
				cnt++
				if protocol.PoWRequirement() != masterPow {
					if mustPass {
						t.Fatalf("node %d: failed to set the new pow requirement for node zero.", i)
					} else {
						return false
					}
				}
			}
		}
	}
	if cnt == 0 {
		t.Fatalf("looking for node zero: no matching peers found.")
	}
	return true
}

func checkPowExchange(t *testing.T) {
	for i, node := range nodes {
		for protocol := range node.waku.peers {
			if protocol.EnodeID() != nodes[0].server.Self().ID() {
				if protocol.PoWRequirement() != masterPow {
					t.Fatalf("node %d: failed to exchange pow requirement in round %d; expected %f, got %f",
						i, round, masterPow, protocol.PoWRequirement())
				}
			}
		}
	}
}

func checkBloomFilterExchangeOnce(t *testing.T, mustPass bool) bool {
	for i, node := range nodes {
		for protocol := range node.waku.peers {
			equals := bytes.Equal(protocol.BloomFilter(), masterBloomFilter)
			if !equals {
				if mustPass {
					t.Fatalf("node %d: failed to exchange bloom filter requirement in round %d. \n%x expected \n%x got",
						i, round, masterBloomFilter, protocol.BloomFilter())
				} else {
					return false
				}
			}
		}
	}

	return true
}

func checkBloomFilterExchange(t *testing.T) {
	const iterations = 200
	for j := 0; j < iterations; j++ {
		lastCycle := j == iterations-1
		ok := checkBloomFilterExchangeOnce(t, lastCycle)
		if ok {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func waitForServersToStart(t *testing.T) {
	const iterations = 200
	var started int64
	for j := 0; j < iterations; j++ {
		time.Sleep(50 * time.Millisecond)
		started = atomic.LoadInt64(&result.started)
		if started == NumNodes {
			return
		}
	}
	t.Fatalf("Failed to start all the servers, running: %d", started)
}
