package network

import (
	"bytes"
	"database/sql"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type Network struct {
	ChainID                uint64 `json:"chain_id"`
	ChainName              string `json:"chain_name"`
	RPCURL                 string `json:"rpc_url"`
	BlockExplorerURL       string `json:"block_explorer_url,omitempty"`
	IconURL                string `json:"icon_url,omitempty"`
	NativeCurrencyName     string `json:"native_currency_name,omitempty"`
	NativeCurrencySymbol   string `json:"native_currency_symbol,omitempty"`
	NativeCurrencyDecimals uint64 `json:"native_currency_decimals"`
	IsTest                 bool   `json:"is_test"`
	Layer                  uint64 `json:"layer"`
	Enabled                bool   `json:"enabled"`
}

const baseQuery = "SELECT chain_id, chain_name, rpc_url, block_explorer_url, icon_url, native_currency_name, native_currency_symbol, native_currency_decimals, is_test, layer, enabled FROM networks"

func newNetworksQuery() *networksQuery {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(baseQuery)
	return &networksQuery{buf: buf}
}

type networksQuery struct {
	buf   *bytes.Buffer
	args  []interface{}
	added bool
}

func (nq *networksQuery) andOrWhere() {
	if nq.added {
		nq.buf.WriteString(" AND")
	} else {
		nq.buf.WriteString(" WHERE")
	}
}

func (nq *networksQuery) filterEnabled(enabled bool) *networksQuery {
	nq.andOrWhere()
	nq.added = true
	nq.buf.WriteString(" enabled = ?")
	nq.args = append(nq.args, enabled)
	return nq
}

func (nq *networksQuery) filterChainID(chainID uint64) *networksQuery {
	nq.andOrWhere()
	nq.added = true
	nq.buf.WriteString(" chain_id = ?")
	nq.args = append(nq.args, chainID)
	return nq
}

func (nq *networksQuery) exec(db *sql.DB) ([]*Network, error) {
	rows, err := db.Query(nq.buf.String(), nq.args...)
	if err != nil {
		return nil, err
	}
	var res []*Network
	defer rows.Close()
	for rows.Next() {
		network := Network{}
		err := rows.Scan(
			&network.ChainID, &network.ChainName, &network.RPCURL, &network.BlockExplorerURL, &network.IconURL,
			&network.NativeCurrencyName, &network.NativeCurrencySymbol, &network.NativeCurrencyDecimals,
			&network.IsTest, &network.Layer, &network.Enabled,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, &network)
	}

	return res, err
}

type Manager struct {
	db            *sql.DB
	legacyChainID uint64
	legacyClient  *ethclient.Client
	chainClients  map[uint64]*ChainClient
}

func NewManager(db *sql.DB, legacyChainID uint64, legacyClient *ethclient.Client) *Manager {
	return &Manager{
		db:            db,
		legacyChainID: legacyChainID,
		legacyClient:  legacyClient,
		chainClients:  make(map[uint64]*ChainClient),
	}
}

func (nm *Manager) Init(networks []Network) error {
	for i := range networks {
		err := nm.Upsert(&networks[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (nm *Manager) GetChainClient(chainID uint64) (*ChainClient, error) {
	if chainID == nm.legacyChainID {
		return &ChainClient{eth: nm.legacyClient, ChainID: chainID}, nil
	}

	if chainClient, ok := nm.chainClients[chainID]; ok {
		return chainClient, nil
	}

	network := nm.Find(chainID)
	if network == nil {
		return nil, fmt.Errorf("could not find network: %d", chainID)
	}

	rpcClient, err := rpc.Dial(network.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("dial upstream server: %s", err)
	}

	chainClient := &ChainClient{eth: ethclient.NewClient(rpcClient), ChainID: chainID}
	nm.chainClients[chainID] = chainClient
	return chainClient, nil
}

func (nm *Manager) GetChainClients(chainIDs []uint64) (res []*ChainClient, err error) {
	for _, chainID := range chainIDs {
		client, err := nm.GetChainClient(chainID)
		if err != nil {
			return nil, err
		}
		res = append(res, client)
	}

	return res, nil
}

func (nm *Manager) Upsert(network *Network) error {
	_, err := nm.db.Exec(
		"INSERT OR REPLACE INTO networks (chain_id, chain_name, rpc_url, block_explorer_url, icon_url, native_currency_name, native_currency_symbol, native_currency_decimals, is_test, layer, enabled) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		network.ChainID, network.ChainName, network.RPCURL, network.BlockExplorerURL, network.IconURL,
		network.NativeCurrencyName, network.NativeCurrencySymbol, network.NativeCurrencyDecimals,
		network.IsTest, network.Layer, network.Enabled,
	)
	return err
}

func (nm *Manager) Delete(chainID uint64) error {
	_, err := nm.db.Exec("DELETE FROM networks WHERE chain_id = ?", chainID)
	return err
}

func (nm *Manager) Find(chainID uint64) *Network {
	networks, err := newNetworksQuery().filterChainID(chainID).exec(nm.db)
	if len(networks) != 1 || err != nil {
		return nil
	}
	return networks[0]
}

func (nm *Manager) Get(onlyEnabled bool) ([]*Network, error) {
	query := newNetworksQuery()
	if onlyEnabled {
		query.filterEnabled(true)
	}

	return query.exec(nm.db)
}
