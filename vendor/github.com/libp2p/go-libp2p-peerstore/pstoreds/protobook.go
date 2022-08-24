package pstoreds

import (
	"errors"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"

	pstore "github.com/libp2p/go-libp2p-core/peerstore"
)

type protoSegment struct {
	sync.RWMutex
}

type protoSegments [256]*protoSegment

func (s *protoSegments) get(p peer.ID) *protoSegment {
	return s[byte(p[len(p)-1])]
}

var errTooManyProtocols = errors.New("too many protocols")

type ProtoBookOption func(*dsProtoBook) error

func WithMaxProtocols(num int) ProtoBookOption {
	return func(pb *dsProtoBook) error {
		pb.maxProtos = num
		return nil
	}
}

type dsProtoBook struct {
	segments  protoSegments
	meta      pstore.PeerMetadata
	maxProtos int
}

var _ pstore.ProtoBook = (*dsProtoBook)(nil)

func NewProtoBook(meta pstore.PeerMetadata, opts ...ProtoBookOption) (*dsProtoBook, error) {
	pb := &dsProtoBook{
		meta: meta,
		segments: func() (ret protoSegments) {
			for i := range ret {
				ret[i] = &protoSegment{}
			}
			return ret
		}(),
		maxProtos: 1024,
	}

	for _, opt := range opts {
		if err := opt(pb); err != nil {
			return nil, err
		}
	}
	return pb, nil
}

func (pb *dsProtoBook) SetProtocols(p peer.ID, protos ...string) error {
	if len(protos) > pb.maxProtos {
		return errTooManyProtocols
	}

	protomap := make(map[string]struct{}, len(protos))
	for _, proto := range protos {
		protomap[proto] = struct{}{}
	}

	s := pb.segments.get(p)
	s.Lock()
	defer s.Unlock()

	return pb.meta.Put(p, "protocols", protomap)
}

func (pb *dsProtoBook) AddProtocols(p peer.ID, protos ...string) error {
	s := pb.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pmap, err := pb.getProtocolMap(p)
	if err != nil {
		return err
	}
	if len(pmap)+len(protos) > pb.maxProtos {
		return errTooManyProtocols
	}

	for _, proto := range protos {
		pmap[proto] = struct{}{}
	}

	return pb.meta.Put(p, "protocols", pmap)
}

func (pb *dsProtoBook) GetProtocols(p peer.ID) ([]string, error) {
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

func (pb *dsProtoBook) RemovePeer(p peer.ID) {
	pb.meta.RemovePeer(p)
}
