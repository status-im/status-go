package network

import (
	"bytes"
	"database/sql"

	"github.com/status-im/status-go/params"
)

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

func (nq *networksQuery) exec(db *sql.DB) ([]*params.Network, error) {
	rows, err := db.Query(nq.buf.String(), nq.args...)
	if err != nil {
		return nil, err
	}
	var res []*params.Network
	defer rows.Close()
	for rows.Next() {
		network := params.Network{}
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
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{
		db: db,
	}
}

func (nm *Manager) Init(networks []params.Network) error {
	if networks == nil {
		return nil
	}

	currentNetworks, _ := nm.Get(false)
	if len(currentNetworks) > 0 {
		return nil
	}

	for i := range networks {
		err := nm.Upsert(&networks[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (nm *Manager) Upsert(network *params.Network) error {
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

func (nm *Manager) Find(chainID uint64) *params.Network {
	networks, err := newNetworksQuery().filterChainID(chainID).exec(nm.db)
	if len(networks) != 1 || err != nil {
		return nil
	}
	return networks[0]
}

func (nm *Manager) Get(onlyEnabled bool) ([]*params.Network, error) {
	query := newNetworksQuery()
	if onlyEnabled {
		query.filterEnabled(true)
	}

	return query.exec(nm.db)
}
