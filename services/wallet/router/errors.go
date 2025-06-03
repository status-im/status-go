package router

import (
	"github.com/status-im/status-go/errors"
)

// Abbreviation `WR` for the error code stands for Wallet Router
var (
	ErrNotEnoughTokenBalance                      = &errors.ErrorResponse{Code: errors.ErrorCode("WR-001"), Details: "not enough token balance, token: %s, chainId: %d"}
	ErrNotEnoughNativeBalance                     = &errors.ErrorResponse{Code: errors.ErrorCode("WR-002"), Details: "not enough native balance, token: %s, chainId: %d"}
	ErrNativeTokenNotFound                        = &errors.ErrorResponse{Code: errors.ErrorCode("WR-003"), Details: "native token not found"}
	ErrTokenNotFound                              = &errors.ErrorResponse{Code: errors.ErrorCode("WR-004"), Details: "token not found"}
	ErrNoBestRouteFound                           = &errors.ErrorResponse{Code: errors.ErrorCode("WR-005"), Details: "no best route found"}
	ErrCannotCheckBalance                         = &errors.ErrorResponse{Code: errors.ErrorCode("WR-006"), Details: "cannot check balance"}
	ErrLowAmountInForHopBridge                    = &errors.ErrorResponse{Code: errors.ErrorCode("WR-007"), Details: "bonder fee greater than estimated received, a higher amount is needed to cover fees"}
	ErrNoPositiveBalance                          = &errors.ErrorResponse{Code: errors.ErrorCode("WR-008"), Details: "no positive balance"}
	ErrCustomFeeModeCannotBeSetThisWay            = &errors.ErrorResponse{Code: errors.ErrorCode("WR-009"), Details: "custom fee mode cannot be set this way"}
	ErrOnlyCustomFeeModeCanBeSetThisWay           = &errors.ErrorResponse{Code: errors.ErrorCode("WR-010"), Details: "only custom fee mode can be set this way"}
	ErrTxIdentityNotProvided                      = &errors.ErrorResponse{Code: errors.ErrorCode("WR-011"), Details: "transaction identity not provided"}
	ErrTxCustomParamsNotProvided                  = &errors.ErrorResponse{Code: errors.ErrorCode("WR-012"), Details: "transaction custom params not provided"}
	ErrCannotCustomizeIfNoRoute                   = &errors.ErrorResponse{Code: errors.ErrorCode("WR-013"), Details: "cannot customize params if no route"}
	ErrCannotFindPathForProvidedIdentity          = &errors.ErrorResponse{Code: errors.ErrorCode("WR-014"), Details: "cannot find path for provided identity"}
	ErrPathNotSupportedForProvidedChain           = &errors.ErrorResponse{Code: errors.ErrorCode("WR-015"), Details: "path not supported for provided chain"}
	ErrPathNotSupportedBetweenProvidedChains      = &errors.ErrorResponse{Code: errors.ErrorCode("WR-016"), Details: "path not supported between provided chains"}
	ErrPathNotAvaliableForProvidedParameters      = &errors.ErrorResponse{Code: errors.ErrorCode("WR-017"), Details: "path not available for provided parameters"}
	ErrCannotFindPathProcessorForProvidedIdentity = &errors.ErrorResponse{Code: errors.ErrorCode("WR-018"), Details: "cannot find path processor for provided identity"}
)
