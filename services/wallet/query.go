package wallet

import (
	"bytes"
	"database/sql"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const baseTransfersQuery = "SELECT transfers.hash, type, blocks.hash, blocks.number, address, tx, receipt FROM transfers JOIN blocks ON blk_hash = blocks.hash"

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
		q.buf.WriteString(" blocks.number >= ?")
		q.args = append(q.args, (*SQLBigInt)(start))
	}
	return q
}

func (q *transfersQuery) FilterEnd(end *big.Int) *transfersQuery {
	if end != nil {
		q.andOrWhere()
		q.added = true
		q.buf.WriteString(" blocks.number <= ?")
		q.args = append(q.args, (*SQLBigInt)(end))
	}
	return q
}

func (q *transfersQuery) FilterAddress(address common.Address) *transfersQuery {
	q.andOrWhere()
	q.added = true
	q.buf.WriteString(" address = ?")
	q.args = append(q.args, address)
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
		}
		err = rows.Scan(
			&transfer.ID, &transfer.Type, &transfer.BlockHash, (*SQLBigInt)(transfer.BlockNumber), &transfer.Address,
			&JSONBlob{transfer.Transaction}, &JSONBlob{transfer.Receipt})
		if err != nil {
			return nil, err
		}
		rst = append(rst, transfer)
	}
	return rst, nil
}
