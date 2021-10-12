package pstoreds

import (
	"context"
	"fmt"
	"io"
	"time"

	base32 "github.com/multiformats/go-base32"

	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"

	peer "github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-core/peerstore"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

// Configuration object for the peerstore.
type Options struct {
	// The size of the in-memory cache. A value of 0 or lower disables the cache.
	CacheSize uint

	// Sweep interval to purge expired addresses from the datastore. If this is a zero value, GC will not run
	// automatically, but it'll be available on demand via explicit calls.
	GCPurgeInterval time.Duration

	// Interval to renew the GC lookahead window. If this is a zero value, lookahead will be disabled and we'll
	// traverse the entire datastore for every purge cycle.
	GCLookaheadInterval time.Duration

	// Initial delay before GC processes start. Intended to give the system breathing room to fully boot
	// before starting GC.
	GCInitialDelay time.Duration
}

// DefaultOpts returns the default options for a persistent peerstore, with the full-purge GC algorithm:
//
// * Cache size: 1024.
// * GC purge interval: 2 hours.
// * GC lookahead interval: disabled.
// * GC initial delay: 60 seconds.
func DefaultOpts() Options {
	return Options{
		CacheSize:           1024,
		GCPurgeInterval:     2 * time.Hour,
		GCLookaheadInterval: 0,
		GCInitialDelay:      60 * time.Second,
	}
}

type pstoreds struct {
	peerstore.Metrics

	dsKeyBook
	dsAddrBook
	dsProtoBook
	dsPeerMetadata
}

// NewPeerstore creates a peerstore backed by the provided persistent datastore.
func NewPeerstore(ctx context.Context, store ds.Batching, opts Options) (*pstoreds, error) {
	addrBook, err := NewAddrBook(ctx, store, opts)
	if err != nil {
		return nil, err
	}

	keyBook, err := NewKeyBook(ctx, store, opts)
	if err != nil {
		return nil, err
	}

	peerMetadata, err := NewPeerMetadata(ctx, store, opts)
	if err != nil {
		return nil, err
	}

	protoBook := NewProtoBook(peerMetadata)

	ps := &pstoreds{
		Metrics:        pstore.NewMetrics(),
		dsKeyBook:      *keyBook,
		dsAddrBook:     *addrBook,
		dsPeerMetadata: *peerMetadata,
		dsProtoBook:    *protoBook,
	}
	return ps, nil
}

// uniquePeerIds extracts and returns unique peer IDs from database keys.
func uniquePeerIds(ds ds.Datastore, prefix ds.Key, extractor func(result query.Result) string) (peer.IDSlice, error) {
	var (
		q       = query.Query{Prefix: prefix.String(), KeysOnly: true}
		results query.Results
		err     error
	)

	if results, err = ds.Query(q); err != nil {
		log.Error(err)
		return nil, err
	}

	defer results.Close()

	idset := make(map[string]struct{})
	for result := range results.Next() {
		k := extractor(result)
		idset[k] = struct{}{}
	}

	if len(idset) == 0 {
		return peer.IDSlice{}, nil
	}

	ids := make(peer.IDSlice, 0, len(idset))
	for id := range idset {
		pid, _ := base32.RawStdEncoding.DecodeString(id)
		id, _ := peer.IDFromBytes(pid)
		ids = append(ids, id)
	}
	return ids, nil
}

func (ps *pstoreds) Close() (err error) {
	var errs []error
	weakClose := func(name string, c interface{}) {
		if cl, ok := c.(io.Closer); ok {
			if err = cl.Close(); err != nil {
				errs = append(errs, fmt.Errorf("%s error: %s", name, err))
			}
		}
	}

	weakClose("keybook", ps.dsKeyBook)
	weakClose("addressbook", ps.dsAddrBook)
	weakClose("protobook", ps.dsProtoBook)
	weakClose("peermetadata", ps.dsPeerMetadata)

	if len(errs) > 0 {
		return fmt.Errorf("failed while closing peerstore; err(s): %q", errs)
	}
	return nil
}

func (ps *pstoreds) Peers() peer.IDSlice {
	set := map[peer.ID]struct{}{}
	for _, p := range ps.PeersWithKeys() {
		set[p] = struct{}{}
	}
	for _, p := range ps.PeersWithAddrs() {
		set[p] = struct{}{}
	}

	pps := make(peer.IDSlice, 0, len(set))
	for p := range set {
		pps = append(pps, p)
	}
	return pps
}

func (ps *pstoreds) PeerInfo(p peer.ID) peer.AddrInfo {
	return peer.AddrInfo{
		ID:    p,
		Addrs: ps.dsAddrBook.Addrs(p),
	}
}
