package appmetrics

import (
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/appmetrics"
)

func NewService(db *appmetrics.Database) *Service {
	return &Service{db: db, metricsBufferedChan: make(chan appmetrics.AppMetric, 8)}
}

type Service struct {
	db                  *appmetrics.Database
	metricsBufferedChan chan appmetrics.AppMetric
}

func (s *Service) Start(*p2p.Server) error {
	return nil
}

func (s *Service) Stop() error {
	// flush pending metrics before stopping the service
	var pendingAppMetrics []appmetrics.AppMetric
	for len(s.metricsBufferedChan) > 0 {
		pendingAppMetrics = append(pendingAppMetrics, <-s.metricsBufferedChan)
	}
	return s.db.SaveAppMetrics(pendingAppMetrics)
}

func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "appmetrics",
			Version:   "0.1.0",
			Service:   NewAPI(s.db, s.metricsBufferedChan),
			Public:    true,
		},
	}
}

func (s *Service) Protocols() []p2p.Protocol {
	return nil
}
