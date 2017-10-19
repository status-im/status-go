package services

import (
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// LesService interface for LightEthereum service.
type LesService interface {
	APIs() []rpc.API
	ResetWithGenesisBlock(*types.Block)
	BlockChain() *light.LightChain
	TxPool() *light.TxPool
	Engine() consensus.Engine
	LesVersion() int
	Downloader() *downloader.Downloader
	EventMux() *event.TypeMux
	Protocols() []p2p.Protocol
	Start(*p2p.Server) error
	Stop() error
	WriteTrustedCht(light.TrustedCht)
}
