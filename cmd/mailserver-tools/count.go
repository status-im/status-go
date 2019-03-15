package main

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-ethereum/rlp"
	"github.com/status-im/status-go/mailserver"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var (
	emptyHash common.Hash
)

func countInTopic(db *leveldb.DB, topic whisper.TopicType, start, end time.Time) (int, error) {
	startKey := mailserver.NewDBKey(uint32(start.Unix()), emptyHash)
	endKey := mailserver.NewDBKey(uint32(end.Unix()), emptyHash)

	iter := db.NewIterator(&util.Range{Start: startKey.Bytes(), Limit: endKey.Bytes()}, nil)
	defer iter.Release()

	counter := 0

	for iter.Next() {
		var envelope whisper.Envelope

		if err := rlp.DecodeBytes(iter.Value(), &envelope); err != nil {
			return 0, err
		}

		if topic == envelope.Topic {
			counter++
		}
	}

	return counter, nil
}

func countLast24HoursInTopic(db *leveldb.DB, topic whisper.TopicType) (int, error) {
	end := time.Now()
	start := end.Add(-time.Hour * 24)

	return countInTopic(db, topic, start, end)
}
