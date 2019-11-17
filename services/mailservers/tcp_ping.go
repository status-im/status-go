package mailservers

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"

	"github.com/status-im/status-go/rtt"
)

type PingQuery struct {
	Addresses []string `json:"addresses"`
	TimeoutMs int      `json:"timeoutMs"`
}

type PingResult struct {
	ENode string  `json:"address"`
	RTTMs int     `json:"rtt_ms"`
	Err   *string `json:"error"`
}

func (pr *PingResult) Update(rttMs int, err error) {
	if err != nil {
		errStr := err.Error()
		pr.Err = &errStr
	}
	pr.RTTMs = rttMs
}

func enodeToAddr(enodeAddr string) (string, error) {
	node, err := enode.ParseV4(enodeAddr)
	if err != nil {
		return "", err
	}
	var ip4 enr.IPv4
	err = node.Load(&ip4)
	if err != nil {
		return "", err
	}
	var tcp enr.TCP
	err = node.Load(&tcp)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", net.IP(ip4).String(), tcp), nil
}

func parseEnodes(enodes []string) (map[string]*PingResult, []string) {
	// parse enode addreses into normal host + port addresses
	results := make(map[string]*PingResult, len(enodes))
	var toPing []string

	for i := range enodes {
		addr, err := enodeToAddr(enodes[i])
		if err != nil {
			// using enode since it's irrelevant but needs to be unique
			errStr := err.Error()
			results[enodes[i]] = &PingResult{ENode: enodes[i], Err: &errStr}
			continue
		}
		results[addr] = &PingResult{ENode: enodes[i]}
		toPing = append(toPing, addr)
	}
	return results, toPing
}

func mapValues(m map[string]*PingResult) []*PingResult {
	rval := make([]*PingResult, 0, len(m))
	for _, value := range m {
		rval = append(rval, value)
	}
	return rval
}

func (a *API) Ping(ctx context.Context, pq PingQuery) ([]*PingResult, error) {
	timeout := time.Duration(pq.TimeoutMs) * time.Millisecond

	// parse enodes into pingable addresses
	resultsMap, toPing := parseEnodes(pq.Addresses)

	// run the checks concurrently
	results, err := rtt.CheckHosts(toPing, timeout)
	if err != nil {
		return nil, err
	}

	// set ping results
	for i := range results {
		r := results[i]
		pr := resultsMap[r.Addr]
		if pr == nil {
			continue
		}
		pr.Update(r.RTTMs, r.Err)
	}

	return mapValues(resultsMap), nil
}
