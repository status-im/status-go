package token

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/contracts/community-tokens/assets"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/logutils"
	communitytoken "github.com/status-im/status-go/internal/protocol/communities/token"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/pkg/services/utils"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

func (tm *Manager) discoverTokenCommunityID(ctx context.Context, token *tokentypes.Token, address common.Address) {
	if token == nil || token.CommunityData != nil {
		// Token is invalid or is alrady discovered. Nothing to do here.
		return
	}
	backend, err := tm.ethClientGetter.EthClient(token.ChainID)
	if err != nil {
		return
	}
	caller, err := assets.NewAssetsCaller(address, backend)
	if err != nil {
		return
	}
	uri, err := caller.BaseTokenURI(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return
	}

	if uri == "" {
		// Update token community ID to prevent further checks
		err := tm.setDiscoveredCommunityID(ctx, token.Token, "")
		if err != nil {
			logutils.ZapLogger().Error("Cannot update community id", zap.Error(err))
		}
		return
	}

	uri = strings.TrimSuffix(uri, "/")
	communityIDHex, err := utils.DeserializePublicKey(uri)
	if err != nil {
		return
	}
	communityID := cryptotypes.EncodeHex(communityIDHex)

	err = tm.setDiscoveredCommunityID(ctx, token.Token, communityID)
	if err != nil {
		logutils.ZapLogger().Error("Cannot update community id", zap.Error(err))
		return
	}
	token.CommunityData = &tokentypes.CommunityData{ID: communityID}
}

func (tm *Manager) setDiscoveredCommunityID(ctx context.Context, token *types.Token, communityID string) error {
	persist := func(ctx context.Context) error {
		_, err := tm.walletDB.ExecContext(ctx, "UPDATE tokens SET community_id=? WHERE network_id=? AND address=?", communityID, token.ChainID, token.Address)
		return err
	}
	if catalogue, ok := tm.tokensManager.(customCatalogue); ok && communityID != "" {
		return catalogue.DeleteCustom(ctx, token.Key(), persist)
	}
	return persist(ctx)
}

func (tm *Manager) GetCommunityTokenType(chainID uint64, tokenContractAddress string) (protobuf.CommunityTokenType, error) {
	if tm.communityTokensDB != nil {
		return tm.communityTokensDB.GetTokenType(chainID, tokenContractAddress)
	}
	return protobuf.CommunityTokenType_UNKNOWN_TOKEN_TYPE, nil
}

func (tm *Manager) GetCommunityTokenPrivilegesLevel(chainID uint64, tokenContractAddress string) (communitytoken.PrivilegesLevel, error) {
	if tm.communityTokensDB != nil {
		return tm.communityTokensDB.GetTokenPrivilegesLevel(chainID, tokenContractAddress)
	}
	return communitytoken.CommunityLevel, nil
}

func (tm *Manager) getDiscoveredTokens(onlyCommunityCustoms bool) ([]*tokentypes.Token, error) {
	query := "SELECT address, name, symbol, decimals, network_id, community_id FROM tokens WHERE community_id IS NOT NULL AND community_id != ''"
	if !onlyCommunityCustoms {
		query = "SELECT address, name, symbol, decimals, network_id, community_id FROM tokens WHERE community_id IS NULL OR community_id = ''"
	}

	rows, err := tm.walletDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*tokentypes.Token
	for rows.Next() {
		token := &tokentypes.Token{
			Token: &types.Token{},
		}
		var communityIDDB sql.NullString
		err := rows.Scan(&token.Address, &token.Name, &token.Symbol, &token.Decimals, &token.ChainID, &communityIDDB)
		if err != nil {
			return nil, err
		}

		if communityIDDB.Valid {
			token.CommunityData = &tokentypes.CommunityData{
				ID: communityIDDB.String,
			}

			if lvl, err := tm.GetCommunityTokenPrivilegesLevel(token.ChainID, token.Address.Hex()); err == nil {
				privilegesLevel := int(lvl)
				token.PrivilegesLevel = &privilegesLevel
			}

			if tm.communityTokenImageBuilder != nil {
				token.LogoURI = tm.communityTokenImageBuilder.MakeCommunityTokenImagesURL(token.CommunityData.ID, token.ChainID, token.Symbol)
			}

			_ = tm.fillCommunityData(token)
		}
		result = append(result, token)
	}

	return result, rows.Err()
}

func (tm *Manager) GetCustoms(onlyCommunityCustoms bool) ([]*tokentypes.Token, error) {
	var (
		result         []*tokentypes.Token
		processedToken = make(map[string]struct{})
	)

	// It's important to process community tokens first, cause we rely to community tokens stored to community_tokens table, then discovered tokens.
	if onlyCommunityCustoms && tm.communityTokensDB != nil {
		communityTokens, err := tm.communityTokensDB.GetTokens()
		if err != nil {
			return nil, err
		}

		for _, communityToken := range communityTokens {
			privilegesLevel := int(communityToken.PrivilegesLevel)
			token := &tokentypes.Token{
				Token: &types.Token{
					Address:  common.HexToAddress(communityToken.Address),
					Name:     communityToken.Name,
					Symbol:   communityToken.Symbol,
					Decimals: uint(communityToken.Decimals),
					ChainID:  uint64(communityToken.ChainID),
					LogoURI:  communityToken.Base64Image,
				},
				CommunityData: &tokentypes.CommunityData{
					ID: communityToken.CommunityID,
				},
				Soulbound:       !communityToken.Transferable,
				PrivilegesLevel: &privilegesLevel,
			}

			if tm.communityTokenImageBuilder != nil {
				token.LogoURI = tm.communityTokenImageBuilder.MakeCommunityTokenImagesURL(token.CommunityData.ID, token.ChainID, token.Symbol)
			}

			processedToken[token.Key()] = struct{}{}
			result = append(result, token)
		}
	}

	discoveredTokens, err := tm.getDiscoveredTokens(onlyCommunityCustoms)
	if err != nil {
		return nil, err
	}

	for _, discoveredToken := range discoveredTokens {
		if _, ok := processedToken[discoveredToken.Key()]; ok {
			continue
		}
		processedToken[discoveredToken.Key()] = struct{}{}
		result = append(result, discoveredToken)
	}

	return result, nil
}

func (tm *Manager) getCustomByChainAddress(onlyCommunityCustoms bool, chainID uint64, address common.Address) (*tokentypes.Token, error) {
	communityTokens, err := tm.GetCustoms(onlyCommunityCustoms)
	if err != nil {
		return nil, err
	}
	for _, ct := range communityTokens {
		if ct.Address == address && ct.ChainID == chainID {
			return ct, nil
		}
	}
	return nil, fmt.Errorf("custom token not found: chainID %d, address %s", chainID, address)
}

func (tm *Manager) UpsertCustom(token tokentypes.Token) error {
	if token.Token == nil {
		return fmt.Errorf("custom token is nil")
	}
	if catalogue, ok := tm.tokensManager.(customCatalogue); ok {
		if token.CommunityData != nil && token.CommunityData.ID != "" {
			return catalogue.DeleteCustom(context.Background(), token.Key(), func(ctx context.Context) error {
				_, err := tm.walletDB.ExecContext(ctx, "INSERT OR REPLACE INTO tokens (network_id,address,name,symbol,decimals,community_id) VALUES (?,?,?,?,?,?)", token.ChainID, token.Address, token.Name, token.Symbol, token.Decimals, token.CommunityData.ID)
				return err
			})
		}
		return catalogue.UpsertCustom(context.Background(), token.Token, func(ctx context.Context, row *types.Token) error {
			_, err := tm.walletDB.ExecContext(ctx, "INSERT OR REPLACE INTO tokens (network_id,address,name,symbol,decimals) VALUES (?,?,?,?,?)", row.ChainID, row.Address, row.Name, row.Symbol, row.Decimals)
			return err
		})
	}
	insert, err := tm.walletDB.Prepare("INSERT OR REPLACE INTO TOKENS (network_id, address, name, symbol, decimals) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer insert.Close()
	_, err = insert.Exec(token.ChainID, token.Address, token.Name, token.Symbol, token.Decimals)
	return err
}

func (tm *Manager) DeleteCustom(chainID uint64, address common.Address) error {
	if catalogue, ok := tm.tokensManager.(customCatalogue); ok {
		return catalogue.DeleteCustom(context.Background(), types.TokenKey(chainID, address), func(ctx context.Context) error {
			_, err := tm.walletDB.ExecContext(ctx, `DELETE FROM tokens WHERE address=? AND network_id=?`, address, chainID)
			return err
		})
	}
	_, err := tm.walletDB.Exec(`DELETE FROM TOKENS WHERE address = ? and network_id = ?`, address, chainID)
	return err
}

// customCatalogue keeps optional C bindings out of ordinary builds.
type customCatalogue interface {
	UpsertCustom(context.Context, *types.Token, func(context.Context, *types.Token) error) error
	DeleteCustom(context.Context, string, func(context.Context) error) error
}

func (tm *Manager) fillCommunityData(token *tokentypes.Token) error {
	if token == nil || token.CommunityData == nil || tm.communityManager == nil {
		return nil
	}

	communityInfo, _, err := tm.communityManager.GetCommunityInfo(token.CommunityData.ID)
	if err != nil {
		return err
	}
	if err == nil && communityInfo != nil {
		// Fetched data from cache. Cache is refreshed during every wallet token list call.
		token.CommunityData.Name = communityInfo.CommunityName
		token.CommunityData.Color = communityInfo.CommunityColor
		token.CommunityData.Image = communityInfo.CommunityImage
	}
	return nil
}
