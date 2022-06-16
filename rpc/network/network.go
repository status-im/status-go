package network

import (
	"bytes"
	"database/sql"
	"fmt"

	"github.com/status-im/status-go/params"
)

const baseQuery = "SELECT chain_id, chain_name, rpc_url, block_explorer_url, icon_url, native_currency_name, native_currency_symbol, native_currency_decimals, is_test, layer, enabled, chain_color, short_name FROM networks"

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
			&network.NativeCurrencyName, &network.NativeCurrencySymbol,
			&network.NativeCurrencyDecimals, &network.IsTest, &network.Layer, &network.Enabled, &network.ChainColor, &network.ShortName,
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

func find(chainID uint64, networks []params.Network) int {
	for i := range networks {
		if networks[i].ChainID == chainID {
			return i
		}
	}
	return -1
}

func (nm *Manager) Init(networks []params.Network) error {
	if networks == nil {
		return nil
	}

	var errors string
	currentNetworks, _ := nm.Get(false)

	// Delete networks which are not supported any more
	for i := range currentNetworks {
		if find(currentNetworks[i].ChainID, networks) == -1 {
			err := nm.Delete(currentNetworks[i].ChainID)
			if err != nil {
				errors += fmt.Sprintf("error deleting network with ChainID: %d, %s", currentNetworks[i].ChainID, err.Error())
			}
		}
	}

	// Add new networks and update rpc url for the old ones
	for i := range networks {
		found := false
		for j := range currentNetworks {
			if currentNetworks[j].ChainID == networks[i].ChainID {
				found = true
				if currentNetworks[j].RPCURL != networks[i].RPCURL {
					// Update rpc_url if it's different
					err := nm.UpdateRPCURL(currentNetworks[j].ChainID, networks[i].RPCURL)
					if err != nil {
						errors += fmt.Sprintf("error updating network rpc_url for ChainID: %d, %s", currentNetworks[j].ChainID, err.Error())
					}
				}
				break
			}
		}

		if !found {
			// Add network if doesn't exist
			err := nm.Upsert(&networks[i])
			if err != nil {
				errors += fmt.Sprintf("error inserting network with ChainID: %d, %s", networks[i].ChainID, err.Error())
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(errors)
	}

	return nil
}

func (nm *Manager) Upsert(network *params.Network) error {
	_, err := nm.db.Exec(
		"INSERT OR REPLACE INTO networks (chain_id, chain_name, rpc_url, block_explorer_url, icon_url, native_currency_name, native_currency_symbol, native_currency_decimals, is_test, layer, enabled, chain_color, short_name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		network.ChainID, network.ChainName, network.RPCURL, network.BlockExplorerURL, network.IconURL,
		network.NativeCurrencyName, network.NativeCurrencySymbol, network.NativeCurrencyDecimals,
		network.IsTest, network.Layer, network.Enabled, network.ChainColor, network.ShortName,
	)
	return err
}

func (nm *Manager) Delete(chainID uint64) error {
	_, err := nm.db.Exec("DELETE FROM networks WHERE chain_id = ?", chainID)
	return err
}

func (nm *Manager) UpdateRPCURL(chainID uint64, rpcURL string) error {
	_, err := nm.db.Exec(`UPDATE networks SET rpc_url = ? WHERE chain_id = ?`, rpcURL, chainID)
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
