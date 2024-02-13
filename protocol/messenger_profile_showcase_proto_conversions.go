package protocol

import "github.com/status-im/status-go/protocol/protobuf"

func FromProfileShowcaseCommunityPreferenceProto(p *protobuf.ProfileShowcaseCommunityPreference) *ProfileShowcaseCommunityPreference {
	return &ProfileShowcaseCommunityPreference{
		CommunityID:        p.GetCommunityId(),
		ShowcaseVisibility: ProfileShowcaseVisibility(p.ShowcaseVisibility),
		Order:              int(p.Order),
	}
}

func FromProfileShowcaseCommunitiesPreferencesProto(preferences []*protobuf.ProfileShowcaseCommunityPreference) []*ProfileShowcaseCommunityPreference {
	out := make([]*ProfileShowcaseCommunityPreference, 0, len(preferences))
	for _, p := range preferences {
		out = append(out, FromProfileShowcaseCommunityPreferenceProto(p))
	}
	return out
}

func ToProfileShowcaseCommunityPreferenceProto(p *ProfileShowcaseCommunityPreference) *protobuf.ProfileShowcaseCommunityPreference {
	return &protobuf.ProfileShowcaseCommunityPreference{
		CommunityId:        p.CommunityID,
		ShowcaseVisibility: protobuf.ProfileShowcaseVisibility(p.ShowcaseVisibility),
		Order:              int32(p.Order),
	}
}

func ToProfileShowcaseCommunitiesPreferencesProto(preferences []*ProfileShowcaseCommunityPreference) []*protobuf.ProfileShowcaseCommunityPreference {
	out := make([]*protobuf.ProfileShowcaseCommunityPreference, 0, len(preferences))
	for _, p := range preferences {
		out = append(out, ToProfileShowcaseCommunityPreferenceProto(p))
	}
	return out
}

func FromProfileShowcaseAccountPreferenceProto(p *protobuf.ProfileShowcaseAccountPreference) *ProfileShowcaseAccountPreference {
	return &ProfileShowcaseAccountPreference{
		Address:            p.GetAddress(),
		Name:               p.GetName(),
		ColorID:            p.GetColorId(),
		Emoji:              p.GetEmoji(),
		ShowcaseVisibility: ProfileShowcaseVisibility(p.ShowcaseVisibility),
		Order:              int(p.Order),
	}
}

func FromProfileShowcaseAccountsPreferencesProto(preferences []*protobuf.ProfileShowcaseAccountPreference) []*ProfileShowcaseAccountPreference {
	out := make([]*ProfileShowcaseAccountPreference, 0, len(preferences))
	for _, p := range preferences {
		out = append(out, FromProfileShowcaseAccountPreferenceProto(p))
	}
	return out
}

func ToProfileShowcaseAccountPreferenceProto(p *ProfileShowcaseAccountPreference) *protobuf.ProfileShowcaseAccountPreference {
	return &protobuf.ProfileShowcaseAccountPreference{
		Address:            p.Address,
		Name:               p.Name,
		ColorId:            p.ColorID,
		Emoji:              p.Emoji,
		ShowcaseVisibility: protobuf.ProfileShowcaseVisibility(p.ShowcaseVisibility),
		Order:              int32(p.Order),
	}
}

func ToProfileShowcaseAccountsPreferenceProto(preferences []*ProfileShowcaseAccountPreference) []*protobuf.ProfileShowcaseAccountPreference {
	out := make([]*protobuf.ProfileShowcaseAccountPreference, 0, len(preferences))
	for _, p := range preferences {
		out = append(out, ToProfileShowcaseAccountPreferenceProto(p))
	}
	return out
}

func FromProfileShowcaseCollectiblePreferenceProto(p *protobuf.ProfileShowcaseCollectiblePreference) *ProfileShowcaseCollectiblePreference {
	return &ProfileShowcaseCollectiblePreference{
		ContractAddress:    p.GetContractAddress(),
		ChainID:            p.GetChainId(),
		TokenID:            p.GetTokenId(),
		CommunityID:        p.GetCommunityId(),
		AccountAddress:     p.GetAccountAddress(),
		ShowcaseVisibility: ProfileShowcaseVisibility(p.ShowcaseVisibility),
		Order:              int(p.Order),
	}
}

func FromProfileShowcaseCollectiblesPreferencesProto(preferences []*protobuf.ProfileShowcaseCollectiblePreference) []*ProfileShowcaseCollectiblePreference {
	out := make([]*ProfileShowcaseCollectiblePreference, 0, len(preferences))
	for _, p := range preferences {
		out = append(out, FromProfileShowcaseCollectiblePreferenceProto(p))
	}
	return out
}

func ToProfileShowcaseCollectiblePreferenceProto(p *ProfileShowcaseCollectiblePreference) *protobuf.ProfileShowcaseCollectiblePreference {
	return &protobuf.ProfileShowcaseCollectiblePreference{
		ContractAddress:    p.ContractAddress,
		ChainId:            p.ChainID,
		TokenId:            p.TokenID,
		CommunityId:        p.CommunityID,
		AccountAddress:     p.AccountAddress,
		ShowcaseVisibility: protobuf.ProfileShowcaseVisibility(p.ShowcaseVisibility),
		Order:              int32(p.Order),
	}
}

func ToProfileShowcaseCollectiblesPreferenceProto(preferences []*ProfileShowcaseCollectiblePreference) []*protobuf.ProfileShowcaseCollectiblePreference {
	out := make([]*protobuf.ProfileShowcaseCollectiblePreference, 0, len(preferences))
	for _, p := range preferences {
		out = append(out, ToProfileShowcaseCollectiblePreferenceProto(p))
	}
	return out
}

func FromProfileShowcaseVerifiedTokenPreferenceProto(p *protobuf.ProfileShowcaseVerifiedTokenPreference) *ProfileShowcaseVerifiedTokenPreference {
	return &ProfileShowcaseVerifiedTokenPreference{
		Symbol:             p.GetSymbol(),
		ShowcaseVisibility: ProfileShowcaseVisibility(p.ShowcaseVisibility),
		Order:              int(p.Order),
	}
}

func FromProfileShowcaseVerifiedTokensPreferencesProto(preferences []*protobuf.ProfileShowcaseVerifiedTokenPreference) []*ProfileShowcaseVerifiedTokenPreference {
	out := make([]*ProfileShowcaseVerifiedTokenPreference, 0, len(preferences))
	for _, p := range preferences {
		out = append(out, FromProfileShowcaseVerifiedTokenPreferenceProto(p))
	}
	return out
}

func ToProfileShowcaseVerifiedTokenPreferenceProto(p *ProfileShowcaseVerifiedTokenPreference) *protobuf.ProfileShowcaseVerifiedTokenPreference {
	return &protobuf.ProfileShowcaseVerifiedTokenPreference{
		Symbol:             p.Symbol,
		ShowcaseVisibility: protobuf.ProfileShowcaseVisibility(p.ShowcaseVisibility),
		Order:              int32(p.Order),
	}

}

func ToProfileShowcaseVerifiedTokensPreferenceProto(preferences []*ProfileShowcaseVerifiedTokenPreference) []*protobuf.ProfileShowcaseVerifiedTokenPreference {
	out := make([]*protobuf.ProfileShowcaseVerifiedTokenPreference, 0, len(preferences))
	for _, p := range preferences {
		out = append(out, ToProfileShowcaseVerifiedTokenPreferenceProto(p))
	}
	return out
}

func FromProfileShowcaseUnverifiedTokenPreferenceProto(p *protobuf.ProfileShowcaseUnverifiedTokenPreference) *ProfileShowcaseUnverifiedTokenPreference {
	return &ProfileShowcaseUnverifiedTokenPreference{
		ContractAddress:    p.GetContractAddress(),
		ChainID:            p.GetChainId(),
		CommunityID:        p.GetCommunityId(),
		ShowcaseVisibility: ProfileShowcaseVisibility(p.ShowcaseVisibility),
		Order:              int(p.Order),
	}
}

func FromProfileShowcaseUnverifiedTokensPreferencesProto(preferences []*protobuf.ProfileShowcaseUnverifiedTokenPreference) []*ProfileShowcaseUnverifiedTokenPreference {
	out := make([]*ProfileShowcaseUnverifiedTokenPreference, 0, len(preferences))
	for _, p := range preferences {
		out = append(out, FromProfileShowcaseUnverifiedTokenPreferenceProto(p))
	}
	return out
}

func ToProfileShowcaseUnverifiedTokenPreferenceProto(p *ProfileShowcaseUnverifiedTokenPreference) *protobuf.ProfileShowcaseUnverifiedTokenPreference {
	return &protobuf.ProfileShowcaseUnverifiedTokenPreference{
		ContractAddress:    p.ContractAddress,
		ChainId:            p.ChainID,
		CommunityId:        p.CommunityID,
		ShowcaseVisibility: protobuf.ProfileShowcaseVisibility(p.ShowcaseVisibility),
		Order:              int32(p.Order),
	}
}

func ToProfileShowcaseUnverifiedTokensPreferenceProto(preferences []*ProfileShowcaseUnverifiedTokenPreference) []*protobuf.ProfileShowcaseUnverifiedTokenPreference {
	out := make([]*protobuf.ProfileShowcaseUnverifiedTokenPreference, 0, len(preferences))
	for _, p := range preferences {
		out = append(out, ToProfileShowcaseUnverifiedTokenPreferenceProto(p))
	}
	return out
}

func FromProfileShowcasePreferencesProto(p *protobuf.SyncProfileShowcasePreferences) *ProfileShowcasePreferences {
	return &ProfileShowcasePreferences{
		Clock:            p.GetClock(),
		Communities:      FromProfileShowcaseCommunitiesPreferencesProto(p.Communities),
		Accounts:         FromProfileShowcaseAccountsPreferencesProto(p.Accounts),
		Collectibles:     FromProfileShowcaseCollectiblesPreferencesProto(p.Collectibles),
		VerifiedTokens:   FromProfileShowcaseVerifiedTokensPreferencesProto(p.VerifiedTokens),
		UnverifiedTokens: FromProfileShowcaseUnverifiedTokensPreferencesProto(p.UnverifiedTokens),
	}
}

func ToProfileShowcasePreferencesProto(p *ProfileShowcasePreferences) *protobuf.SyncProfileShowcasePreferences {
	return &protobuf.SyncProfileShowcasePreferences{
		Clock:            p.Clock,
		Communities:      ToProfileShowcaseCommunitiesPreferencesProto(p.Communities),
		Accounts:         ToProfileShowcaseAccountsPreferenceProto(p.Accounts),
		Collectibles:     ToProfileShowcaseCollectiblesPreferenceProto(p.Collectibles),
		VerifiedTokens:   ToProfileShowcaseVerifiedTokensPreferenceProto(p.VerifiedTokens),
		UnverifiedTokens: ToProfileShowcaseUnverifiedTokensPreferenceProto(p.UnverifiedTokens),
	}
}
