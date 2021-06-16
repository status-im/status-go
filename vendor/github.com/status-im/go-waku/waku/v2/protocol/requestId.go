package protocol

import (
	"crypto/rand"
	"sync"

	"github.com/cruxic/go-hmac-drbg/hmacdrbg"
	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("request-gen")

var brHmacDrbgPool = sync.Pool{New: func() interface{} {
	seed := make([]byte, 48)
	_, err := rand.Read(seed)
	if err != nil {
		log.Fatal(err)
	}
	return hmacdrbg.NewHmacDrbg(256, seed, nil)
}}

func GenerateRequestId() []byte {
	rng := brHmacDrbgPool.Get().(*hmacdrbg.HmacDrbg)
	defer brHmacDrbgPool.Put(rng)

	randData := make([]byte, 32)
	if !rng.Generate(randData) {
		//Reseed is required every 10,000 calls
		seed := make([]byte, 48)
		_, err := rand.Read(seed)
		if err != nil {
			log.Fatal(err)
		}
		err = rng.Reseed(seed)
		if err != nil {
			//only happens if seed < security-level
			log.Fatal(err)
		}

		if !rng.Generate(randData) {
			log.Error("could not generate random request id")
		}
	}
	return randData
}
