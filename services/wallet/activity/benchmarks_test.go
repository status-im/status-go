package activity

import (
	"context"
	"fmt"
	"testing"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/testutils"
	"github.com/status-im/status-go/services/wallet/transfer"
)

func setupBenchmark(b *testing.B, accountsCount int, inMemory bool) (deps FilterDependencies, close func(), accounts []eth.Address) {
	deps, close = setupTestActivityDBStorageChoice(b, inMemory)

	const transactionCount = 100000
	const mtSendRatio = 0.2   // 20%
	const mtSwapRatio = 0.1   // 10%
	const mtBridgeRatio = 0.1 // 10%
	const pendingCount = 10
	const mtSendCount = int(float64(transactionCount) * mtSendRatio)
	const mtSwapCount = int(float64(transactionCount) * mtSwapRatio)
	// Bridge requires two transactions
	const mtBridgeCount = int(float64(transactionCount) * (mtBridgeRatio / 2))

	trs, _, _ := transfer.GenerateTestTransfers(b, deps.db, 0, transactionCount)

	accounts = []eth.Address{}
	for i := 0; i < accountsCount; i++ {
		if i%2 == 0 {
			accounts = append(accounts, trs[i].From)
		} else {
			accounts = append(accounts, trs[i].To)
		}
	}

	i := 0
	multiTxs := make([]transfer.TestMultiTransaction, mtSendCount+mtSwapCount+mtBridgeCount)
	for ; i < mtSendCount; i++ {
		multiTxs[i] = transfer.GenerateTestSendMultiTransaction(trs[i])
		trs[i].From = accounts[i%len(accounts)]
		multiTxs[i].FromAddress = trs[i].From
		// Currently the network ID is not filled in for send transactions
		multiTxs[i].FromNetworkID = nil
		multiTxs[i].ToNetworkID = nil

		multiTxs[i].MultiTransactionID = transfer.InsertTestMultiTransaction(b, deps.db, &multiTxs[i])
		trs[i].MultiTransactionID = multiTxs[i].MultiTransactionID
	}

	for j := 0; j < mtSwapCount; i, j = i+1, j+1 {
		multiTxs[i] = transfer.GenerateTestSwapMultiTransaction(trs[i], testutils.SntSymbol, int64(i))
		trs[i].From = accounts[i%len(accounts)]
		multiTxs[i].FromAddress = trs[i].From

		multiTxs[i].MultiTransactionID = transfer.InsertTestMultiTransaction(b, deps.db, &multiTxs[i])
		trs[i].MultiTransactionID = multiTxs[i].MultiTransactionID
	}

	for mtIdx := 0; mtIdx < mtBridgeCount; i, mtIdx = i+2, mtIdx+1 {
		firstTrIdx := i
		secondTrIdx := i + 1
		multiTxs[mtIdx] = transfer.GenerateTestBridgeMultiTransaction(trs[firstTrIdx], trs[secondTrIdx])
		trs[firstTrIdx].From = accounts[i%len(accounts)]
		trs[secondTrIdx].To = accounts[(i+3)%len(accounts)]
		multiTxs[mtIdx].FromAddress = trs[firstTrIdx].From
		multiTxs[mtIdx].ToAddress = trs[secondTrIdx].To
		multiTxs[mtIdx].FromAddress = trs[i].From

		multiTxs[mtIdx].MultiTransactionID = transfer.InsertTestMultiTransaction(b, deps.db, &multiTxs[mtIdx])
		trs[firstTrIdx].MultiTransactionID = multiTxs[mtIdx].MultiTransactionID
		trs[secondTrIdx].MultiTransactionID = multiTxs[mtIdx].MultiTransactionID
	}

	for i = 0; i < transactionCount-pendingCount; i++ {
		trs[i].From = accounts[i%len(accounts)]
		transfer.InsertTestTransfer(b, deps.db, trs[i].From, &trs[i])
	}

	for ; i < transactionCount; i++ {
		trs[i].From = accounts[i%len(accounts)]
		transfer.InsertTestPendingTransaction(b, deps.db, &trs[i])
	}

	return
}

var allNetEnabled = []common.ChainID(nil)

func BenchmarkGetActivityEntries(bArg *testing.B) {
	deps, closeFn, accounts := setupBenchmark(bArg, 6, true)
	defer closeFn()

	type params struct {
		inMemory bool
		// resultCount must be nil to expect as many requested
		resultCount            *int
		generateTestParameters func() (addresses []eth.Address, allAddresses bool, networks []common.ChainID, filter *Filter, startIndex int)
	}
	testCases := []struct {
		name   string
		params params
	}{
		{
			"RAM_NoFilter",
			params{
				true,
				nil,
				func() ([]eth.Address, bool, []common.ChainID, *Filter, int) {
					return accounts, true, allNetEnabled, &Filter{}, 0
				},
			},
		},
		{
			"SSD_NoFilter",
			params{
				false,
				nil,
				func() ([]eth.Address, bool, []common.ChainID, *Filter, int) {
					return accounts, true, allNetEnabled, &Filter{}, 0
				},
			},
		},
		{
			"SSD_MovingWindow",
			params{
				false,
				nil,
				func() ([]eth.Address, bool, []common.ChainID, *Filter, int) {
					return accounts, true, allNetEnabled, &Filter{}, 200
				},
			},
		},
		{
			"SSD_AllAddr_AllTos",
			params{
				false,
				nil,
				func() ([]eth.Address, bool, []common.ChainID, *Filter, int) {
					return accounts, true, allNetEnabled, &Filter{CounterpartyAddresses: accounts[3:]}, 0
				},
			},
		},
		{
			"SSD_OneAddress",
			params{
				false,
				nil,
				func() ([]eth.Address, bool, []common.ChainID, *Filter, int) {
					return accounts[0:1], false, allNetEnabled, &Filter{}, 0
				},
			},
		},
		// All memory from here
		{
			"FilterSend_AllAddr",
			params{
				true,
				nil,
				func() ([]eth.Address, bool, []common.ChainID, *Filter, int) {
					return accounts, true, allNetEnabled, &Filter{
						Types: []Type{SendAT},
					}, 0
				},
			},
		},
		{
			"FilterSend_6Addr",
			params{
				true,
				nil,
				func() ([]eth.Address, bool, []common.ChainID, *Filter, int) {
					return accounts[len(accounts)-6:], false, allNetEnabled, &Filter{
						Types: []Type{SendAT},
					}, 0
				},
			},
		},
		{
			"FilterThreeNetworks",
			params{
				true,
				nil,
				func() ([]eth.Address, bool, []common.ChainID, *Filter, int) {
					return accounts, true, []common.ChainID{}, &Filter{}, 0
				},
			},
		},
	}

	const resultCount = 100
	for _, tc := range testCases {
		addresses, allAddresses, nets, filter, startIndex := tc.params.generateTestParameters()
		networks := allNetworksFilter()
		if len(nets) > 0 {
			networks = nets
		}

		bArg.Run(tc.name, func(b *testing.B) {
			// Reset timer after setup
			b.ResetTimer()

			// Run benchmark
			for i := 0; i < b.N; i++ {
				res, err := getActivityEntries(context.Background(), deps, addresses, allAddresses, networks, *filter, startIndex, resultCount)
				if err != nil {
					b.Error(err)
				} else if tc.params.resultCount != nil {
					if len(res) != *tc.params.resultCount {
						b.Error("Got less then expected")
					}
				} else if len(res) != resultCount {
					b.Error("Got less than requested")
				}
			}
		})
	}
}

func BenchmarkSQLQuery(b *testing.B) {
	type params struct {
		query            string
		args             []interface{}
		expectedResCount int
	}

	deps, closeFn, accounts := setupBenchmark(b, 10000, true)
	defer closeFn()

	var addrValuesStr, insertAddrValuesStr, addrPlaceholdersStr string
	var refAccounts []interface{}
	for _, acc := range accounts {
		addrValuesStr += fmt.Sprintf("X'%s',", acc.Hex()[2:])
		insertAddrValuesStr += fmt.Sprintf("(X'%s'),", acc.Hex()[2:])
		addrPlaceholdersStr += "?,"
		refAccounts = append(refAccounts, acc)
	}
	addrValuesStr = addrValuesStr[:len(addrValuesStr)-1]
	insertAddrValuesStr = insertAddrValuesStr[:len(insertAddrValuesStr)-1]
	addrPlaceholdersStr = addrPlaceholdersStr[:len(addrPlaceholdersStr)-1]

	if _, err := deps.db.Exec(fmt.Sprintf("CREATE TEMP TABLE filter_addresses_table (address VARCHAR PRIMARY KEY); INSERT INTO filter_addresses_table (address) VALUES %s;", insertAddrValuesStr)); err != nil {
		b.Fatal("failed to create temporary table", err)
	}

	testCases := []struct {
		name string
		args params
		res  testing.BenchmarkResult
	}{
		{
			name: "JoinQuery",
			args: params{
				query:            "SELECT COUNT(*) FROM transfers JOIN filter_addresses_table ON transfers.tx_from_address = filter_addresses_table.address",
				expectedResCount: 99990,
			},
		},
		{
			name: "LiteralQuery",
			args: params{
				query:            fmt.Sprintf("SELECT COUNT(*) FROM transfers WHERE tx_from_address IN (%s)", addrValuesStr),
				expectedResCount: 99990,
			},
		},
		{
			name: "ParamQuery",
			args: params{
				query:            fmt.Sprintf("SELECT COUNT(*) FROM transfers WHERE tx_from_address IN (%s)", addrPlaceholdersStr),
				args:             refAccounts,
				expectedResCount: 99990,
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					res, err := deps.db.Query(tc.args.query, tc.args.args...)
					if err != nil {
						b.Fatal("failed to query db", err)
					}
					res.Next()

					var count int
					if err := res.Scan(&count); err != nil {
						b.Fatal("failed to scan db result", err)
					}
					if count != tc.args.expectedResCount {
						b.Fatalf("unexpected result count: %d, expected: %d", count, tc.args.expectedResCount)
					}

					res.Close()
				}
			}
		})
	}
}
