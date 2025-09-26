package eventfilter

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/go-wallet-sdk/pkg/eventlog"
)

type FilterClient interface {
	FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error)
}

func FilterTransfers(ctx context.Context, client FilterClient, config TransferQueryConfig) ([]eventlog.Event, error) {
	queries := config.ToFilterQueries()

	queryEventsCh := make(chan []eventlog.Event, len(queries))
	queryErrorsCh := make(chan error, len(queries))

	wg := sync.WaitGroup{}
	for _, query := range queries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logs, err := client.FilterLogs(ctx, query)
			if err != nil {
				queryErrorsCh <- err
				return
			}
			queryEvents := make([]eventlog.Event, 0)
			for _, log := range logs {
				event := eventlog.ParseLog(log)
				queryEvents = append(queryEvents, event...)
			}
			queryEventsCh <- queryEvents
		}()
	}
	wg.Wait()

	close(queryEventsCh)
	close(queryErrorsCh)

	queryErrors := make([]error, 0, len(queries))
	for queryError := range queryErrorsCh {
		if queryError != nil {
			queryErrors = append(queryErrors, queryError)
		}
	}
	if len(queryErrors) > 0 {
		return nil, errors.Join(queryErrors...)
	}

	events := make([]eventlog.Event, 0)
	for queryEvents := range queryEventsCh {
		events = append(events, queryEvents...)
	}
	return events, nil
}
