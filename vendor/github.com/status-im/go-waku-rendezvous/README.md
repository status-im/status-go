# Rendezvous Protocol

### Overview

Similar to [status-im/rendezvous](https://github.com/status-im/rendezvous) 
in using a smaller liveness TTL for records (20s), and not using unregistering 
records, due to assuming that the TTL is very low (making it incompatible 
with libp2p original rendezvous spec). This module is intended to be used 
in go-waku as a lightweight mechanism for generalized peer discovery.

A difference compared to status-im/rendezvous is the usage of [routing records](https://github.com/libp2p/specs/blob/master/RFC/0003-routing-records.md) and [signed envelopes](https://github.com/libp2p/specs/blob/master/RFC/0002-signed-envelopes.md) instead of ENR records

**Protocol identifier**: `/vac/waku/rendezvous/0.0.1`

### Usage

**Adding discovery to gossipsub**
```go
import (
  "github.com/libp2p/go-libp2p"
  "github.com/libp2p/go-libp2p-core/host"
  "github.com/libp2p/go-libp2p-core/peer"
  pubsub "github.com/status-im/go-libp2p-pubsub"
  rendezvous "github.com/status-im/go-waku-rendezvous"
)

// create a new libp2p Host that listens on a random TCP port
h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
if err != nil {
  panic(err)
}

// Create a rendezvous instance
rendezvous := rendezvous.NewRendezvousDiscovery(h)

// create a new PubSub service using the GossipSub router
ps, err := pubsub.NewGossipSub(ctx, h, pubsub.WithDiscovery(rendezvous))
if err != nil {
  panic(err)
}
```

**Creating a rendezvous server**
```go
import (
  "database/sql"
  "github.com/syndtr/goleveldb/leveldb"
  "github.com/syndtr/goleveldb/leveldb/opt"
  "github.com/syndtr/goleveldb/leveldb/util"
  "github.com/libp2p/go-libp2p"
  "github.com/libp2p/go-libp2p-core/host"
  "github.com/libp2p/go-libp2p-core/peer"
  pubsub "github.com/status-im/go-libp2p-pubsub"
  rendezvous "github.com/status-im/go-waku-rendezvous"
)

type RendezVousLevelDB struct {
	db *leveldb.DB
}

func NewRendezVousLevelDB(dBPath string) (*RendezVousLevelDB, error) {
	db, err := leveldb.OpenFile(dBPath, &opt.Options{OpenFilesCacheCapacity: 3})

	if err != nil {
		return nil, err
	}

	return &RendezVousLevelDB{db}, nil
}

func (r *RendezVousLevelDB) Delete(key []byte) error {
	return r.db.Delete(key, nil)
}

func (r *RendezVousLevelDB) Put(key []byte, value []byte) error {
	return r.db.Put(key, value, nil)
}

func (r *RendezVousLevelDB) NewIterator(prefix []byte) rendezvous.Iterator {
	return r.db.NewIterator(util.BytesPrefix(prefix), nil)
}


// create a new libp2p Host that listens on a random TCP port
h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
if err != nil {
  panic(err)
}

// LevelDB storage for peer records
db, err := NewRendezVousLevelDB("/tmp/rendezvous")
if err != nil {
  panic(err)
}
storage := rendezvous.NewStorage(db)

rendezvousService = rendezvous.NewRendezvousService(h, storage)
if err := rendezvousService.Start(); err != nil {
  panic(err)
}
```

### Protobuf

- [record.pb.Envelope](https://github.com/libp2p/specs/blob/master/RFC/0002-signed-envelopes.md#wire-format)
- [PeerRecord protobuffer](https://github.com/libp2p/specs/blob/master/RFC/0003-routing-records.md#address-record-format)

```protobuf
message Message {
  enum MessageType {
    REGISTER = 0;
    REGISTER_RESPONSE = 1;
    DISCOVER = 2;
    DISCOVER_RESPONSE = 3;
  }

  enum ResponseStatus {
    OK                  = 0;
    E_INVALID_NAMESPACE = 100;
    E_INVALID_PEER_INFO = 101;
    E_INVALID_TTL       = 102;
    E_NOT_AUTHORIZED    = 200;
    E_INTERNAL_ERROR    = 300;
    E_UNAVAILABLE       = 400;
  }

  message Register {
    string ns = 1;
    bytes signedPeerRecord = 2;
    int64 ttl = 3; // in seconds
  }

  message RegisterResponse {
    ResponseStatus status = 1;
    string statusText = 2;
    int64 ttl = 3;
  }

  message Discover {
    string ns = 1;
    int64 limit = 2;
  }

  message DiscoverResponse {
    repeated Register registrations = 1;
    ResponseStatus status = 3;
    string statusText = 4;
  }

  MessageType type = 1;
  Register register = 2;
  RegisterResponse registerResponse = 3;
  Discover discover = 4;
  DiscoverResponse discoverResponse = 5;
}

```
