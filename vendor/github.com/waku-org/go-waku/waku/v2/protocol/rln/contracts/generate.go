package contracts

//go:generate solcjs --abi --bin ./rln-contract/contracts/Rln.sol -o ./
//go:generate abigen --abi ./rln-contract_contracts_Rln_sol_RLN.abi --pkg contracts --type RLN --out ./RLN.go --bin ./rln-contract_contracts_Rln_sol_RLN.bin
//go:generate abigen --abi ./rln-contract_contracts_PoseidonHasher_sol_PoseidonHasher.abi --pkg contracts --type PoseidonHasher --out ./PoseidonHasher.go --bin ./rln-contract_contracts_PoseidonHasher_sol_PoseidonHasher.bin
