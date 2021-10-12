package pstoreds

import (
	"fmt"
	"sync"

	peer "github.com/libp2p/go-libp2p-core/peer"

	pstore "github.com/libp2p/go-libp2p-core/peerstore"
)

type protoSegment struct {
	sync.RWMutex
}

type protoSegments [256]*protoSegment

func (s *protoSegments) get(p peer.ID) *protoSegment {
	return s[byte(p[len(p)-1])]
}

type dsProtoBook struct {
	segments protoSegments
	meta     pstore.PeerMetadata
}

var _ pstore.ProtoBook = (*dsProtoBook)(nil)

func NewProtoBook(meta pstore.PeerMetadata) *dsProtoBook {
	return &dsProtoBook{
		meta: meta,
		segments: func() (ret protoSegments) {
			for i := range ret {
				ret[i] = &protoSegment{}
			}
			return ret
		}(),
	}
}

func (pb *dsProtoBook) SetProtocols(p peer.ID, protos ...string) error {
	if err := p.Validate(); err != nil {
		return err
	}

	s := pb.segments.get(p)
	s.Lock()
	defer s.Unlock()

	protomap := make(map[string]struct{}, len(protos))
	for _, proto := range protos {
		protomap[proto] = struct{}{}
	}

	return pb.meta.Put(p, "protocols", protomap)
}

func (pb *dsProtoBook) AddProtocols(p peer.ID, protos ...string) error {
	if err := p.Validate(); err != nil {
		return err
	}

	s := pb.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pmap, err := pb.getProtocolMap(p)
	if err != nil {
		return err
	}

	for _, proto := range protos {
		pmap[proto] = struct{}{}
	}

	return pb.meta.Put(p, "protocols", pmap)
}

func (pb *dsProtoBook) GetProtocols(p peer.ID) ([]string, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	s := pb.segments.get(p)
	s.RLock()
	defer s.RUnlock()

	pmap, err := pb.getProtocolMap(p)
	if err != nil {
		return nil, err
	}

	res := make([]string, 0, len(pmap))
	for proto := range pmap {
		res = append(res, proto)
	}

	return res, nil
}

func (pb *dsProtoBook) SupportsProtocols(p peer.ID, protos ...string) ([]string, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	s := pb.segments.get(p)
	s.RLock()
	defer s.RUnlock()

	pmap, err := pb.getProtocolMap(p)
	if err != nil {
		return nil, err
	}

	res := make([]string, 0, len(protos))
	for _, proto := range protos {
		if _, ok := pmap[proto]; ok {
			res = append(res, proto)
		}
	}

	return res, nil
}

func (pb *dsProtoBook) FirstSupportedProtocol(p peer.ID, protos ...string) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}

	s := pb.segments.get(p)
	s.RLock()
	defer s.RUnlock()

	pmap, err := pb.getProtocolMap(p)
	if err != nil {
		return "", err
	}
	for _, proto := range protos {
		if _, ok := pmap[proto]; ok {
			return proto, nil
		}
	}

	return "", nil
}

func (pb *dsProtoBook) RemoveProtocols(p peer.ID, protos ...string) error {
	if err := p.Validate(); err != nil {
		return err
	}

	s := pb.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pmap, err := pb.getProtocolMap(p)
	if err != nil {
		return err
	}

	if len(pmap) == 0 {
		// nothing to do.
		return nil
	}

	for _, proto := range protos {
		delete(pmap, proto)
	}

	return pb.meta.Put(p, "protocols", pmap)
}

func (pb *dsProtoBook) getProtocolMap(p peer.ID) (map[string]struct{}, error) {
	iprotomap, err := pb.meta.Get(p, "protocols")
	switch err {
	default:
		return nil, err
	case pstore.ErrNotFound:
		return make(map[string]struct{}), nil
	case nil:
		cast, ok := iprotomap.(map[string]struct{})
		if !ok {
			return nil, fmt.Errorf("stored protocol set was not a map")
		}

		return cast, nil
	}
}
