package communities

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"go.uber.org/zap"

	maps "golang.org/x/exp/maps"
	slices "golang.org/x/exp/slices"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/protocol/ens"
	"github.com/status-im/status-go/protocol/protobuf"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type PermissionChecker interface {
	CheckPermissionToJoin(*Community, []gethcommon.Address) (*CheckPermissionToJoinResponse, error)
	CheckPermissions(permissions []*CommunityTokenPermission, accountsAndChainIDs []*AccountChainIDsCombination, shortcircuit bool) (*CheckPermissionsResponse, error)
}

type DefaultPermissionChecker struct {
	tokenManager        TokenManager
	collectiblesManager CollectiblesManager
	ensVerifier         *ens.Verifier

	logger *zap.Logger
}

func (p *DefaultPermissionChecker) getOwnedENS(addresses []gethcommon.Address) ([]string, error) {
	ownedENS := make([]string, 0)
	if p.ensVerifier == nil {
		p.logger.Warn("no ensVerifier configured for communities manager")
		return ownedENS, nil
	}
	for _, address := range addresses {
		name, err := p.ensVerifier.ReverseResolve(address)
		if err != nil && err.Error() != "not a resolver" {
			return ownedENS, err
		}
		if name != "" {
			ownedENS = append(ownedENS, name)
		}
	}
	return ownedENS, nil
}
func (p *DefaultPermissionChecker) GetOwnedERC721Tokens(walletAddresses []gethcommon.Address, tokenRequirements map[uint64]map[string]*protobuf.TokenCriteria, chainIDs []uint64) (CollectiblesByChain, error) {
	if p.collectiblesManager == nil {
		return nil, errors.New("no collectibles manager")
	}

	ctx := context.Background()

	ownedERC721Tokens := make(CollectiblesByChain)

	for chainID, erc721Tokens := range tokenRequirements {

		skipChain := true
		for _, cID := range chainIDs {
			if chainID == cID {
				skipChain = false
			}
		}

		if skipChain {
			continue
		}

		contractAddresses := make([]gethcommon.Address, 0)
		for contractAddress := range erc721Tokens {
			contractAddresses = append(contractAddresses, gethcommon.HexToAddress(contractAddress))
		}

		if _, exists := ownedERC721Tokens[chainID]; !exists {
			ownedERC721Tokens[chainID] = make(map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress)
		}

		for _, owner := range walletAddresses {
			balances, err := p.collectiblesManager.FetchBalancesByOwnerAndContractAddress(ctx, walletcommon.ChainID(chainID), owner, contractAddresses)
			if err != nil {
				p.logger.Info("couldn't fetch owner assets", zap.Error(err))
				return nil, err
			}
			ownedERC721Tokens[chainID][owner] = balances
		}
	}
	return ownedERC721Tokens, nil
}

func (p *DefaultPermissionChecker) accountChainsCombinationToMap(combinations []*AccountChainIDsCombination) map[gethcommon.Address][]uint64 {
	result := make(map[gethcommon.Address][]uint64)
	for _, combination := range combinations {
		result[combination.Address] = combination.ChainIDs
	}
	return result
}

// merge valid combinations w/o duplicates
func (p *DefaultPermissionChecker) MergeValidCombinations(left, right []*AccountChainIDsCombination) []*AccountChainIDsCombination {

	leftMap := p.accountChainsCombinationToMap(left)
	rightMap := p.accountChainsCombinationToMap(right)

	// merge maps, result in left map
	for k, v := range rightMap {
		if _, exists := leftMap[k]; !exists {
			leftMap[k] = v
			continue
		} else {
			// append chains which are new
			chains := leftMap[k]
			for _, chainID := range v {
				if !slices.Contains(chains, chainID) {
					chains = append(chains, chainID)
				}
			}
			leftMap[k] = chains
		}
	}

	result := []*AccountChainIDsCombination{}
	for k, v := range leftMap {
		result = append(result, &AccountChainIDsCombination{
			Address:  k,
			ChainIDs: v,
		})
	}

	return result
}

func (p *DefaultPermissionChecker) CheckPermissionToJoin(community *Community, addresses []gethcommon.Address) (*CheckPermissionToJoinResponse, error) {
	becomeAdminPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_ADMIN)
	becomeMemberPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)
	becomeTokenMasterPermissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)

	adminOrTokenMasterPermissionsToJoin := append(becomeAdminPermissions, becomeTokenMasterPermissions...)

	allChainIDs, err := p.tokenManager.GetAllChainIDs()
	if err != nil {
		return nil, err
	}

	accountsAndChainIDs := combineAddressesAndChainIDs(addresses, allChainIDs)

	// Check becomeMember and (admin & token master) permissions separately.
	becomeMemberPermissionsResponse, err := p.checkPermissionsOrDefault(becomeMemberPermissions, accountsAndChainIDs)
	if err != nil {
		return nil, err
	}

	if len(adminOrTokenMasterPermissionsToJoin) <= 0 {
		return becomeMemberPermissionsResponse, nil
	}
	// If there are any admin or token master permissions, combine result.

	adminOrTokenPermissionsResponse, err := p.CheckPermissions(adminOrTokenMasterPermissionsToJoin, accountsAndChainIDs, false)
	if err != nil {
		return nil, err
	}

	mergedPermissions := make(map[string]*PermissionTokenCriteriaResult)
	maps.Copy(mergedPermissions, becomeMemberPermissionsResponse.Permissions)
	maps.Copy(mergedPermissions, adminOrTokenPermissionsResponse.Permissions)

	mergedCombinations := p.MergeValidCombinations(becomeMemberPermissionsResponse.ValidCombinations, adminOrTokenPermissionsResponse.ValidCombinations)

	combinedResponse := &CheckPermissionsResponse{
		Satisfied:         becomeMemberPermissionsResponse.Satisfied || adminOrTokenPermissionsResponse.Satisfied,
		Permissions:       mergedPermissions,
		ValidCombinations: mergedCombinations,
	}

	return combinedResponse, nil
}

func (p *DefaultPermissionChecker) checkPermissionsOrDefault(permissions []*CommunityTokenPermission, accountsAndChainIDs []*AccountChainIDsCombination) (*CheckPermissionsResponse, error) {
	if len(permissions) == 0 {
		// There are no permissions to join on this community at the moment,
		// so we reveal all accounts + all chain IDs
		response := &CheckPermissionsResponse{
			Satisfied:         true,
			Permissions:       make(map[string]*PermissionTokenCriteriaResult),
			ValidCombinations: accountsAndChainIDs,
		}
		return response, nil
	}
	return p.CheckPermissions(permissions, accountsAndChainIDs, false)
}

// CheckPermissions will retrieve balances and check whether the user has
// permission to join the community, if shortcircuit is true, it will stop as soon
// as we know the answer
func (p *DefaultPermissionChecker) CheckPermissions(permissions []*CommunityTokenPermission, accountsAndChainIDs []*AccountChainIDsCombination, shortcircuit bool) (*CheckPermissionsResponse, error) {

	response := &CheckPermissionsResponse{
		Satisfied:         false,
		Permissions:       make(map[string]*PermissionTokenCriteriaResult),
		ValidCombinations: make([]*AccountChainIDsCombination, 0),
	}

	erc20TokenRequirements, erc721TokenRequirements, _ := ExtractTokenCriteria(permissions)

	erc20ChainIDsMap := make(map[uint64]bool)
	erc721ChainIDsMap := make(map[uint64]bool)

	erc20TokenAddresses := make([]gethcommon.Address, 0)
	accounts := make([]gethcommon.Address, 0)

	for _, accountAndChainIDs := range accountsAndChainIDs {
		accounts = append(accounts, accountAndChainIDs.Address)
	}

	// figure out chain IDs we're interested in
	for chainID, tokens := range erc20TokenRequirements {
		erc20ChainIDsMap[chainID] = true
		for contractAddress := range tokens {
			erc20TokenAddresses = append(erc20TokenAddresses, gethcommon.HexToAddress(contractAddress))
		}
	}

	for chainID := range erc721TokenRequirements {
		erc721ChainIDsMap[chainID] = true
	}

	chainIDsForERC20 := calculateChainIDsSet(accountsAndChainIDs, erc20ChainIDsMap)
	chainIDsForERC721 := calculateChainIDsSet(accountsAndChainIDs, erc721ChainIDsMap)

	// if there are no chain IDs that match token criteria chain IDs
	// we aren't able to check balances on selected networks
	if len(erc20ChainIDsMap) > 0 && len(chainIDsForERC20) == 0 {
		p.logger.Info("network not supported")
		response.NetworksNotSupported = true
		return response, nil
	}

	ownedERC20TokenBalances := make(map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big, 0)
	if len(chainIDsForERC20) > 0 {
		p.logger.Info("getting erc20 balances")
		// this only returns balances for the networks we're actually interested in
		balances, err := p.tokenManager.GetBalancesByChain(context.Background(), accounts, erc20TokenAddresses, chainIDsForERC20)
		if err != nil {
			return nil, err
		}
		p.logger.Info("got balances", zap.Any("balnaces", balances))
		ownedERC20TokenBalances = balances
	}

	ownedERC721Tokens := make(CollectiblesByChain)
	if len(chainIDsForERC721) > 0 {
		p.logger.Info("getting erc21")
		collectibles, err := p.GetOwnedERC721Tokens(accounts, erc721TokenRequirements, chainIDsForERC721)
		if err != nil {
			return nil, err
		}
		p.logger.Info("got colectibles", zap.Any("balnaces", collectibles))
		ownedERC721Tokens = collectibles
	}

	accountsChainIDsCombinations := make(map[gethcommon.Address]map[uint64]bool)

	for _, tokenPermission := range permissions {

		p.logger.Info("checking permission", zap.Any("p", tokenPermission))
		permissionRequirementsMet := true
		response.Permissions[tokenPermission.Id] = &PermissionTokenCriteriaResult{Role: tokenPermission.Type}

		// There can be multiple token requirements per permission.
		// If only one is not met, the entire permission is marked
		// as not fulfilled
		for _, tokenRequirement := range tokenPermission.TokenCriteria {

			p.logger.Info("checking token requirement", zap.Any("p", tokenRequirement))
			tokenRequirementMet := false
			tokenRequirementResponse := TokenRequirementResponse{TokenCriteria: tokenRequirement}

			if tokenRequirement.Type == protobuf.CommunityTokenType_ERC721 {
				if len(ownedERC721Tokens) == 0 {

					p.logger.Info("checking token requirement 2")
					response.Permissions[tokenPermission.Id].TokenRequirements = append(response.Permissions[tokenPermission.Id].TokenRequirements, tokenRequirementResponse)
					response.Permissions[tokenPermission.Id].Criteria = append(response.Permissions[tokenPermission.Id].Criteria, false)
					continue
				}

			chainIDLoopERC721:
				for chainID, addressStr := range tokenRequirement.ContractAddresses {
					p.logger.Info("checking token requirement 3")
					contractAddress := gethcommon.HexToAddress(addressStr)
					if _, exists := ownedERC721Tokens[chainID]; !exists || len(ownedERC721Tokens[chainID]) == 0 {
						p.logger.Info("checking token requirement 4")
						continue chainIDLoopERC721
					}

					p.logger.Info("checking token requirement 5")
					for account := range ownedERC721Tokens[chainID] {
						p.logger.Info("checking token requirement 6", zap.Any("account", account))
						if _, exists := ownedERC721Tokens[chainID][account]; !exists {
							continue
						}

						p.logger.Info("checking token requirement 7")
						tokenBalances := ownedERC721Tokens[chainID][account][contractAddress]
						if len(tokenBalances) > 0 {
							p.logger.Info("checking token requirement 8")
							// 'account' owns some TokenID owned from contract 'address'
							if _, exists := accountsChainIDsCombinations[account]; !exists {
								accountsChainIDsCombinations[account] = make(map[uint64]bool)
							}

							if len(tokenRequirement.TokenIds) == 0 {
								// no specific tokenId of this collection is needed
								tokenRequirementMet = true
								accountsChainIDsCombinations[account][chainID] = true
								break chainIDLoopERC721
							}

						tokenIDsLoop:
							for _, tokenID := range tokenRequirement.TokenIds {
								tokenIDBigInt := new(big.Int).SetUint64(tokenID)

								for _, asset := range tokenBalances {
									if asset.TokenID.Cmp(tokenIDBigInt) == 0 && asset.Balance.Sign() > 0 {
										tokenRequirementMet = true
										accountsChainIDsCombinations[account][chainID] = true
										break tokenIDsLoop
									}
								}
							}
						}
					}
				}
			} else if tokenRequirement.Type == protobuf.CommunityTokenType_ERC20 {
				p.logger.Info("checking token erc20 1")
				if len(ownedERC20TokenBalances) == 0 {
					p.logger.Info("checking token erc20 2")
					response.Permissions[tokenPermission.Id].TokenRequirements = append(response.Permissions[tokenPermission.Id].TokenRequirements, tokenRequirementResponse)
					response.Permissions[tokenPermission.Id].Criteria = append(response.Permissions[tokenPermission.Id].Criteria, false)
					continue
				}

				accumulatedBalance := new(big.Int)

			chainIDLoopERC20:
				for chainID, address := range tokenRequirement.ContractAddresses {
					p.logger.Info("checking token erc20 2", zap.Any("chain-id", chainID), zap.Any("address", address))
					if _, exists := ownedERC20TokenBalances[chainID]; !exists || len(ownedERC20TokenBalances[chainID]) == 0 {
						p.logger.Info("checking token erc20 3", zap.Any("chain-id", chainID), zap.Any("address", address))
						continue chainIDLoopERC20
					}
					contractAddress := gethcommon.HexToAddress(address)
					for account := range ownedERC20TokenBalances[chainID] {
						p.logger.Info("checking token erc20 3", zap.Any("chain-id", chainID), zap.Any("address", address), zap.Any("account", account))
						if _, exists := ownedERC20TokenBalances[chainID][account][contractAddress]; !exists {
							continue
						}

						value := ownedERC20TokenBalances[chainID][account][contractAddress]

						if _, exists := accountsChainIDsCombinations[account]; !exists {
							accountsChainIDsCombinations[account] = make(map[uint64]bool)
						}

						if value.ToInt().Cmp(big.NewInt(0)) > 0 {
							// account has balance > 0 on this chain for this token, so let's add it the chain IDs
							accountsChainIDsCombinations[account][chainID] = true
						}

						// check if adding current chain account balance to accumulated balance
						// satisfies required amount
						prevBalance := accumulatedBalance
						accumulatedBalance.Add(prevBalance, value.ToInt())

						requiredAmount, success := new(big.Int).SetString(tokenRequirement.AmountInWei, 10)
						if !success {
							p.logger.Info("checking token erc20 3", zap.Any("chain-id", chainID), zap.Any("address", address), zap.Any("account", account))
							return nil, fmt.Errorf("amountInWeis value is incorrect")
						}

						if accumulatedBalance.Cmp(requiredAmount) != -1 {
							p.logger.Info("checking token erc20 4", zap.Any("chain-id", chainID), zap.Any("address", address), zap.Any("account", account))
							tokenRequirementMet = true
							if shortcircuit {
								break chainIDLoopERC20
							}
						}
					}
				}

			} else if tokenRequirement.Type == protobuf.CommunityTokenType_ENS {

				for _, account := range accounts {
					ownedENSNames, err := p.getOwnedENS([]gethcommon.Address{account})
					if err != nil {
						return nil, err
					}

					if _, exists := accountsChainIDsCombinations[account]; !exists {
						accountsChainIDsCombinations[account] = make(map[uint64]bool)
					}

					if !strings.HasPrefix(tokenRequirement.EnsPattern, "*.") {
						for _, ownedENS := range ownedENSNames {
							if ownedENS == tokenRequirement.EnsPattern {
								tokenRequirementMet = true
								accountsChainIDsCombinations[account][walletcommon.EthereumMainnet] = true
							}
						}
					} else {
						parentName := tokenRequirement.EnsPattern[2:]
						for _, ownedENS := range ownedENSNames {
							if strings.HasSuffix(ownedENS, parentName) {
								tokenRequirementMet = true
								accountsChainIDsCombinations[account][walletcommon.EthereumMainnet] = true
							}
						}
					}
				}
			}
			if !tokenRequirementMet {
				permissionRequirementsMet = false
			}

			tokenRequirementResponse.Satisfied = tokenRequirementMet
			response.Permissions[tokenPermission.Id].TokenRequirements = append(response.Permissions[tokenPermission.Id].TokenRequirements, tokenRequirementResponse)
			response.Permissions[tokenPermission.Id].Criteria = append(response.Permissions[tokenPermission.Id].Criteria, tokenRequirementMet)
		}
		// multiple permissions are treated as logical OR, meaning
		// if only one of them is fulfilled, the user gets permission
		// to join and we can stop early
		if shortcircuit && permissionRequirementsMet {
			p.logger.Info("breaking")
			break
		}
	}

	// attach valid account and chainID combinations to response
	for account, chainIDs := range accountsChainIDsCombinations {
		combination := &AccountChainIDsCombination{
			Address: account,
		}
		for chainID := range chainIDs {
			combination.ChainIDs = append(combination.ChainIDs, chainID)
		}
		response.ValidCombinations = append(response.ValidCombinations, combination)
	}

	p.logger.Info("response", zap.Any("response", response))
	response.calculateSatisfied()
	p.logger.Info("response 2", zap.Any("response", response))

	return response, nil
}
