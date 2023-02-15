package transfer

import (
	"bytes"
	"database/sql"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/services/wallet/bigint"
)

const baseTransfersQuery = "SELECT hash, type, blk_hash, blk_number, timestamp, address, tx, sender, receipt, log, network_id, base_gas_fee, COALESCE(multi_transaction_id, 0) FROM transfers"

func newTransfersQuery() *transfersQuery {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(baseTransfersQuery)
	return &transfersQuery{buf: buf}
}

type transfersQuery struct {
	buf   *bytes.Buffer
	args  []interface{}
	added bool
}

func (q *transfersQuery) andOrWhere() {
	if q.added {
		q.buf.WriteString(" AND")
	} else {
		q.buf.WriteString(" WHERE")
	}
}

func (q *transfersQuery) FilterStart(start *big.Int) *transfersQuery {
	if start != nil {
		q.andOrWhere()
		q.added = true
		q.buf.WriteString(" blk_number >= ?")
		q.args = append(q.args, (*bigint.SQLBigInt)(start))
	}
	return q
}

func (q *transfersQuery) FilterEnd(end *big.Int) *transfersQuery {
	if end != nil {
		q.andOrWhere()
		q.added = true
		q.buf.WriteString(" blk_number <= ?")
		q.args = append(q.args, (*bigint.SQLBigInt)(end))
	}
	return q
}

func (q *transfersQuery) FilterLoaded(loaded int) *transfersQuery {
	q.andOrWhere()
	q.added = true
	q.buf.WriteString(" loaded = ? ")
	q.args = append(q.args, loaded)

	return q
}

func (q *transfersQuery) FilterNetwork(network uint64) *transfersQuery {
	q.andOrWhere()
	q.added = true
	q.buf.WriteString(" network_id = ?")
	q.args = append(q.args, network)
	return q
}

func (q *transfersQuery) FilterAddress(address common.Address) *transfersQuery {
	q.andOrWhere()
	q.added = true
	q.buf.WriteString(" address = ?")
	q.args = append(q.args, address)
	return q
}

func (q *transfersQuery) FilterBlockHash(blockHash common.Hash) *transfersQuery {
	q.andOrWhere()
	q.added = true
	q.buf.WriteString(" blk_hash = ?")
	q.args = append(q.args, blockHash)
	return q
}

func (q *transfersQuery) FilterBlockNumber(blockNumber *big.Int) *transfersQuery {
	q.andOrWhere()
	q.added = true
	q.buf.WriteString(" blk_number = ?")
	q.args = append(q.args, (*bigint.SQLBigInt)(blockNumber))
	return q
}

func (q *transfersQuery) Limit(pageSize int64) *transfersQuery {
	q.buf.WriteString(" ORDER BY blk_number DESC, hash ASC ")
	q.buf.WriteString(" LIMIT ?")
	q.args = append(q.args, pageSize)
	return q
}

func (q *transfersQuery) String() string {
	return q.buf.String()
}

func (q *transfersQuery) Args() []interface{} {
	return q.args
}

func (q *transfersQuery) Scan(rows *sql.Rows) (rst []Transfer, err error) {
	for rows.Next() {
		transfer := Transfer{
			BlockNumber: &big.Int{},
			Transaction: &types.Transaction{},
			Receipt:     &types.Receipt{},
			Log:         &types.Log{},
		}
		err = rows.Scan(
			&transfer.ID, &transfer.Type, &transfer.BlockHash,
			(*bigint.SQLBigInt)(transfer.BlockNumber), &transfer.Timestamp, &transfer.Address,
			&JSONBlob{transfer.Transaction}, &transfer.From, &JSONBlob{transfer.Receipt}, &JSONBlob{transfer.Log}, &transfer.NetworkID, &transfer.BaseGasFees, &transfer.MultiTransactionID)
		if err != nil {
			return nil, err
		}
		rst = append(rst, transfer)
	}
	return rst, nil
}
