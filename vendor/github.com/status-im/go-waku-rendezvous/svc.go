package rendezvous

import (
	"sync"
	"time"

	pb "github.com/status-im/go-waku-rendezvous/pb"

	ggio "github.com/gogo/protobuf/io"

	"github.com/libp2p/go-libp2p-core/host"
	inet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	MaxTTL               = 20 // 20sec
	networkDelay         = 500 * time.Millisecond
	cleanerPeriod        = 2 * time.Second
	MaxNamespaceLength   = 256
	MaxPeerAddressLength = 2048
	MaxDiscoverLimit     = int64(1000)
)

type RendezvousService struct {
	h       host.Host
	storage Storage
	cleaner *Cleaner
	wg      sync.WaitGroup
	quit    chan struct{}
}

func NewRendezvousService(host host.Host, storage Storage) *RendezvousService {
	rz := &RendezvousService{
		storage: storage,
		h:       host,
		cleaner: NewCleaner(),
	}

	return rz
}

func (rz *RendezvousService) Start() error {
	rz.h.SetStreamHandler(RendezvousID_v001, rz.handleStream)

	if err := rz.startCleaner(); err != nil {
		return err
	}
	// once server is restarted all cleaner info is lost. so we need to rebuild it
	return rz.storage.IterateAllKeys(func(key RecordsKey, deadline time.Time) error {
		if !rz.cleaner.Exist(key.String()) {
			ns := TopicPart(key)
			log.Debugf("active registration with", "ns", string(ns))
		}
		rz.cleaner.Add(deadline, key.String())
		return nil
	})
}

func (rz *RendezvousService) startCleaner() error {
	rz.quit = make(chan struct{})
	rz.wg.Add(1)
	go func() {
		for {
			select {
			case <-time.After(cleanerPeriod):
				rz.purgeOutdated()
			case <-rz.quit:
				rz.wg.Done()
				return
			}
		}
	}()
	return nil
}

// Stop closes listener and waits till all helper goroutines are stopped.
func (rz *RendezvousService) Stop() {
	if rz.quit == nil {
		return
	}
	select {
	case <-rz.quit:
		return
	default:
	}
	close(rz.quit)
	rz.wg.Wait()
}

func (rz *RendezvousService) purgeOutdated() {
	keys := rz.cleaner.PopSince(time.Now())
	log.Info("removed records from cleaner", "deadlines", len(rz.cleaner.deadlines), "heap", len(rz.cleaner.heap), "lth", len(keys))
	for _, key := range keys {
		topic := TopicPart([]byte(key))
		log.Debug("Removing record with", "topic", string(topic))
		if err := rz.storage.RemoveByKey(key); err != nil {
			log.Error("error removing key from storage", "key", key, "error", err)
		}
	}
}

func (rz *RendezvousService) handleStream(s inet.Stream) {
	defer s.Reset()

	pid := s.Conn().RemotePeer()
	log.Debugf("New stream from %s", pid.Pretty())

	r := ggio.NewDelimitedReader(s, inet.MessageSizeMax)
	w := ggio.NewDelimitedWriter(s)

	for {
		var req pb.Message
		var res pb.Message

		err := r.ReadMsg(&req)
		if err != nil {
			return
		}

		t := req.GetType()
		switch t {
		case pb.Message_REGISTER:
			r := rz.handleRegister(pid, req.GetRegister())
			res.Type = pb.Message_REGISTER_RESPONSE
			res.RegisterResponse = r
			err = w.WriteMsg(&res)
			if err != nil {
				log.Debugf("Error writing response: %s", err.Error())
				return
			}

		case pb.Message_DISCOVER:
			r := rz.handleDiscover(pid, req.GetDiscover())
			res.Type = pb.Message_DISCOVER_RESPONSE
			res.DiscoverResponse = r
			err = w.WriteMsg(&res)
			if err != nil {
				log.Debugf("Error writing response: %s", err.Error())
				return
			}

		default:
			log.Debugf("Unexpected message: %s", t.String())
			return
		}
	}
}

func (rz *RendezvousService) handleRegister(p peer.ID, m *pb.Message_Register) *pb.Message_RegisterResponse {
	ns := m.GetNs()
	if ns == "" {
		return newRegisterResponseError(pb.Message_E_INVALID_NAMESPACE, "unspecified namespace")
	}

	if len(ns) > MaxNamespaceLength {
		return newRegisterResponseError(pb.Message_E_INVALID_NAMESPACE, "namespace too long")
	}

	mpi := m.GetPeer()
	if mpi == nil {
		return newRegisterResponseError(pb.Message_E_INVALID_PEER_INFO, "missing peer info")
	}

	peerRecord, err := pbToPeerRecord(mpi)
	if err != nil {
		return newRegisterResponseError(pb.Message_E_INVALID_PEER_INFO, "invalid peer record")
	}

	if peerRecord.ID != p {
		return newRegisterResponseError(pb.Message_E_INVALID_PEER_INFO, "peer id mismatch")
	}

	if len(peerRecord.Addrs) == 0 {
		return newRegisterResponseError(pb.Message_E_INVALID_PEER_INFO, "missing peer addresses")
	}

	mlen := 0
	for _, maddr := range peerRecord.Addrs {
		mlen += len(maddr.Bytes())
	}
	if mlen > MaxPeerAddressLength {
		return newRegisterResponseError(pb.Message_E_INVALID_PEER_INFO, "peer info too long")
	}

	// Note:
	// We don't validate the addresses, because they could include protocols we don't understand
	// Perhaps we should though.

	mttl := m.GetTtl()
	if mttl < 0 || mttl > MaxTTL {
		return newRegisterResponseError(pb.Message_E_INVALID_TTL, "bad ttl")
	}

	ttl := DefaultTTL
	if mttl > 0 {
		ttl = int(mttl)
	}

	deadline := time.Now().Add(time.Duration(ttl)).Add(networkDelay)

	envPayload, err := marshalEnvelope(mpi)
	if err != nil {
		return newRegisterResponseError(pb.Message_E_INTERNAL_ERROR, err.Error())
	}

	key, err := rz.storage.Add(ns, peerRecord.ID, envPayload, ttl, deadline)
	if err != nil {
		return newRegisterResponseError(pb.Message_E_INTERNAL_ERROR, err.Error())
	}

	if !rz.cleaner.Exist(key) {
		log.Debugf("active registration with", "ns", ns)
	}

	log.Debugf("updating record in the cleaner", "deadline", deadline, "ns", ns)
	rz.cleaner.Add(deadline, key)

	log.Infof("registered peer %s %s (%d)", p, ns, ttl)

	return newRegisterResponse(ttl)
}

func (rz *RendezvousService) handleDiscover(p peer.ID, m *pb.Message_Discover) *pb.Message_DiscoverResponse {
	ns := m.GetNs()

	if len(ns) > MaxNamespaceLength {
		return newDiscoverResponseError(pb.Message_E_INVALID_NAMESPACE, "namespace too long")
	}

	limit := MaxDiscoverLimit
	mlimit := m.GetLimit()
	if mlimit > 0 && mlimit < int64(limit) {
		limit = mlimit
	}

	records, err := rz.storage.GetRandom(ns, limit)
	if err != nil {
		log.Errorf("Error in query: %s", err.Error())
		return newDiscoverResponseError(pb.Message_E_INTERNAL_ERROR, "database error")
	}

	log.Infof("discover query: %s %s -> %d", p, ns, len(records))

	response, err := newDiscoverResponse(records)
	if err != nil {
		log.Errorf("Error in response: %s", err.Error())
		return newDiscoverResponseError(pb.Message_E_INTERNAL_ERROR, "error building response")
	}

	return response
}
