package communities

import (
	"crypto/ecdsa"
	"errors"
	"sync"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

var membersReevaluationTick = 10 * time.Second
var membersReevaluationInterval = 8 * time.Hour
var membersReevaluationCooldown = 5 * time.Minute

type reevaluationExecutionType int

const (
	reevaluationExecutionRegular reevaluationExecutionType = iota
	reevaluationExecutionOnDemand
	reevaluationExecutionForced
)

type reevaluationFunc = func(reevaluationExecutionType) (stop bool, err error)

type membersReevaluationTask struct {
	startedAt  time.Time
	endedAt    time.Time
	demandedAt time.Time
	execute    reevaluationFunc
	mutex      sync.Mutex
}

type membersReevaluationScheduler struct {
	tasks  sync.Map // stores `membersReevaluationTask`
	forces sync.Map // stores `chan struct{}`
	quit   chan struct{}
	logger *zap.Logger
}

func (t *membersReevaluationTask) shouldExecute(force bool) *reevaluationExecutionType {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if force {
		result := reevaluationExecutionForced
		return &result
	}

	now := time.Now()

	if !t.endedAt.After(now.Add(-membersReevaluationCooldown)) {
		return nil
	}

	if t.endedAt.After(now.Add(-membersReevaluationInterval)) {
		result := reevaluationExecutionRegular
		return &result
	}

	if t.startedAt.Before(t.demandedAt) {
		result := reevaluationExecutionOnDemand
		return &result
	}

	return nil
}

func (t *membersReevaluationTask) setStartTime(time time.Time) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.startedAt = time
}

func (t *membersReevaluationTask) setEndTime(time time.Time) (elapsed time.Duration) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.endedAt = time
	return t.endedAt.Sub(t.startedAt)
}

func (t *membersReevaluationTask) setDemandTime(time time.Time) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.demandedAt = time
}

func (s *membersReevaluationScheduler) getTask(communityID string) (*membersReevaluationTask, error) {
	t, exists := s.tasks.Load(communityID)
	if !exists {
		return nil, errors.New("task doesn't exist")
	}

	task, ok := t.(*membersReevaluationTask)
	if !ok {
		return nil, errors.New("invalid task type")
	}

	return task, nil
}

func (s *membersReevaluationScheduler) iterate(communityID string, force bool) (stop bool) {
	task, err := s.getTask(communityID)
	if err != nil {
		return true
	}

	executionType := task.shouldExecute(force)
	if executionType == nil {
		return false
	}

	task.setStartTime(time.Now())

	stop, err = task.execute(*executionType)
	if err != nil {
		s.logger.Error("can't reevaluate members", zap.Error(err))
		return stop
	}

	elapsed := task.setEndTime(time.Now())

	s.logger.Info("reevaluation finished",
		zap.String("communityID", communityID),
		zap.Duration("elapsed", elapsed),
	)

	return stop
}

func (s *membersReevaluationScheduler) loop(communityID string, reevaluator reevaluationFunc, setupDone chan struct{}) {
	_, exists := s.tasks.Load(communityID)
	if exists {
		setupDone <- struct{}{}
		return
	}

	s.tasks.Store(communityID, &membersReevaluationTask{execute: reevaluator})
	defer s.tasks.Delete(communityID)

	force := make(chan struct{}, 10)
	s.forces.Store(communityID, force)
	defer s.forces.Delete(communityID)

	ticker := time.NewTicker(membersReevaluationTick)
	defer ticker.Stop()

	setupDone <- struct{}{}

	// Perform the first iteration immediately
	stop := s.iterate(communityID, true)
	if stop {
		return
	}

	for {
		select {
		case <-ticker.C:
			stop := s.iterate(communityID, false)
			if stop {
				return
			}

		case <-force:
			stop := s.iterate(communityID, true)
			if stop {
				return
			}

		case <-s.quit:
			return
		}
	}
}

func (s *membersReevaluationScheduler) Start(communityID string, reevaluator reevaluationFunc) {
	setupDone := make(chan struct{})
	go s.loop(communityID, reevaluator, setupDone)
	<-setupDone
}

func (s *membersReevaluationScheduler) Push(communityID string) error {
	task, err := s.getTask(communityID)
	if err != nil {
		return err
	}
	task.setDemandTime(time.Now())
	return nil
}

func (s *membersReevaluationScheduler) Force(communityID string) error {
	t, exists := s.forces.Load(communityID)
	if !exists {
		return errors.New("scheduler not started yet")
	}

	force, ok := t.(chan struct{})
	if !ok {
		return errors.New("invalid cast")
	}

	force <- struct{}{}
	return nil
}

type reevaluationScopeController struct {
	evaluatedPermissions map[string]TokenPermissions
}

func newReevaluationScopeController() *reevaluationScopeController {
	return &reevaluationScopeController{
		evaluatedPermissions: map[string]TokenPermissions{},
	}
}

func (r *reevaluationScopeController) setAsEvaluated(communityID string, permissions TokenPermissions) {
	r.evaluatedPermissions[communityID] = permissions
}

func (r *reevaluationScopeController) permissionsToEvaluate(communityID string, permissions TokenPermissions, t reevaluationExecutionType) TokenPermissions {
	switch t {
	case reevaluationExecutionRegular, reevaluationExecutionForced:
		return permissions // evaluate all permissions
	case reevaluationExecutionOnDemand:
		previouslyEvaluated, ok := r.evaluatedPermissions[communityID]
		if !ok {
			return permissions
		}
		return permissionsToEvaluate(previouslyEvaluated, permissions)
	}
	return nil
}

func permissionsToEvaluate(prev, current TokenPermissions) TokenPermissions {
	result := TokenPermissions{}

	changes := evaluatePermissionsChanges(prev, current)

	// Evaluate all newly added permissions
	maps.Copy(result, changes.Added)

	return nil
}

// if token master permission is added, then check it for all members who are not token-masters yet, and nominate them if satisfied
// if admin permission is added, then check it for all members who are not admins or token-masters yet, and nominate them if satisfied
// if become member is permission added, then do nothing
// if view and post channel permission is added, then check it for all members who are not view&post yet for given channel, and add/nominate them if satisfied
// if view channel permission is added, then check it for all members who are not view&post or view yet, and add/nominate them if satisfied

// if token master permission is modified, then check it for all members, if it is not satisfied for a member, then check all other token-master permissions and nominate/drop them if needed
// if admin permission is added, then check it for all members (except token-masters), if it is not satisfied, then check all other admin permissions and nominate/drop them if satisfied
// if become member permission is modified, then check it for all members (except token-masters and admins), if it is not satisfied, check all other permissions and drop members not satisfied

// if token-master permission is removed, then check remaining token-master permissions for token-masters, if not satisfied, then check admin permission, then become member permissions
// if admin permission is removed, then check remaining admin permissions for admins, if not satisfied, then check  become member permissions
// if become member permission is removed, then check remaining become member permissions for members, if not satisfied, remove them

type tokenPermissionChangesByType struct {
	BecomeTokenMaster     TokenPermissionChanges
	BecomeAdmin           TokenPermissionChanges
	BecomeMember          TokenPermissionChanges
	CanViewAndPostChannel TokenPermissionChanges
	CanViewChannel        TokenPermissionChanges
}

func NewTokenPermissionChangesByType(all TokenPermissionChanges) (*tokenPermissionChangesByType, error) {
	result := &tokenPermissionChangesByType{
		BecomeTokenMaster:     NewTokenPermissionChanges(),
		BecomeAdmin:           NewTokenPermissionChanges(),
		BecomeMember:          NewTokenPermissionChanges(),
		CanViewAndPostChannel: NewTokenPermissionChanges(),
		CanViewChannel:        NewTokenPermissionChanges(),
	}

	resultElementByType := func(t protobuf.CommunityTokenPermission_Type) (*TokenPermissionChanges, error) {
		switch t {
		case protobuf.CommunityTokenPermission_BECOME_ADMIN:
			return &result.BecomeAdmin, nil
		case protobuf.CommunityTokenPermission_BECOME_MEMBER:
			return &result.BecomeMember, nil
		case protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER:
			return &result.BecomeTokenMaster, nil
		case protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL:
			return &result.CanViewAndPostChannel, nil
		case protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL:
			return &result.CanViewChannel, nil
		}
		return nil, errors.New("unexpected permission type")
	}

	for id, permission := range all.Added {
		r, err := resultElementByType(permission.Type)
		if err != nil {
			return nil, err
		}
		r.Added[id] = permission
	}

	for id, permission := range all.Modified {
		r, err := resultElementByType(permission.Type)
		if err != nil {
			return nil, err
		}
		r.Modified[id] = permission
	}

	for id, permission := range all.Removed {
		r, err := resultElementByType(permission.Type)
		if err != nil {
			return nil, err
		}
		r.Removed[id] = permission
	}

	return result, nil
}

func (m *Manager) reevaluateMembers2(communityID types.HexBytes, changes TokenPermissionChanges) (*Community, map[protobuf.CommunityMember_Roles][]*ecdsa.PublicKey, error) {
	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, nil, err
	}

	if !community.IsControlNode() {
		return nil, nil, ErrNotEnoughPermissions
	}

	changesByType, err := NewTokenPermissionChangesByType(changes)
	if err != nil {
		return nil, nil, err
	}

	membersAccounts, err := m.persistence.GetCommunityRequestsToJoinRevealedAddresses(community.ID())
	if err != nil {
		return nil, nil, err
	}

	result := &reevaluateMembersResult{
		membersToRemove:             map[string]struct{}{},
		membersRoles:                map[string]*reevaluateMemberRole{},
		membersToRemoveFromChannels: map[string]map[string]struct{}{},
		membersToAddToChannels:      map[string]map[string]protobuf.CommunityMember_ChannelRole{},
	}

	collectiblesOwners := CollectiblesOwners{}
	fetchCollectiblesIfNeeded := func(communityData map[protobuf.CommunityTokenPermission_Type]*PreParsedCommunityPermissionsData, channelData map[string]*PreParsedCommunityPermissionsData) error {
		collectiblesToFetch := map[walletcommon.ChainID]map[gethcommon.Address]struct{}{}

		for chainID, addresses := range CollectibleAddressesFromPreParsedPermissionsData(communityData, channelData) {
			_, ok := collectiblesOwners[chainID]
			if !ok {
				collectiblesToFetch[chainID] = addresses
				continue
			}

			for address := range addresses {
				_, ok := collectiblesOwners[chainID][address]
				if !ok {
					collectiblesToFetch[chainID][address] = struct{}{}
				}
			}
		}

		if len(collectiblesToFetch) == 0 {
			return nil
		}

		fetchedCollectibles, err := m.fetchCollectiblesOwners(collectiblesToFetch)
		if err != nil {
			return err
		}

		// Merge fetched collectibles
		for chainID, ownersRhs := range fetchedCollectibles {
			ownersLhs, ok := collectiblesOwners[chainID]
			if !ok {
				collectiblesOwners[chainID] = ownersRhs
				continue
			}
			for address, ownership := range ownersRhs {
				ownersLhs[address] = ownership
			}
		}

		return nil
	}

	for memberKey := range community.Members() {
		memberPubKey, err := common.HexToPubkey(memberKey)
		if err != nil {
			return nil, nil, err
		}

		if memberKey == common.PubkeyToHex(&m.identity.PublicKey) || community.IsMemberOwner(memberPubKey) {
			continue
		}

		revealedAccount, memberHasWallet := membersAccounts[memberKey]
		if !memberHasWallet {
			result.membersToRemove[memberKey] = struct{}{}
			continue
		}

		accountsAndChainIDs := revealedAccountsToAccountsAndChainIDsCombination(revealedAccount)

		result.membersRoles[memberKey] = &reevaluateMemberRole{
			old: community.MemberRole(memberPubKey),
			new: protobuf.CommunityMember_ROLE_NONE,
		}

		reprocessAll := len(changesByType.BecomeTokenMaster.Modified) > 0 || len(changesByType.BecomeTokenMaster.Removed) > 0
		if reprocessAll {
			communityPermissionsPreParsedData, channelPermissionsPreParsedData := PreParsePermissionsDataByType(community.tokenPermissions(), protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
			err = fetchCollectiblesIfNeeded(communityPermissionsPreParsedData, channelPermissionsPreParsedData)
			if err != nil {
				return nil, nil, err
			}

			becomeTokenMasterPermissions := communityPermissionsPreParsedData[protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER]
			if becomeTokenMasterPermissions != nil {
				permissionResponse, err := m.PermissionChecker.CheckPermissionsWithPreFetchedData(becomeTokenMasterPermissions, accountsAndChainIDs, true, collectiblesOwners)
				if err != nil {
					return nil, nil, err
				}

				if permissionResponse.Satisfied {
					result.membersRoles[memberKey].new = protobuf.CommunityMember_ROLE_TOKEN_MASTER
					// Skip further validation if user has TokenMaster permissions
					continue
				}
			}
		} else if len(changesByType.BecomeTokenMaster.Added) > 0 {
			if result.membersRoles[memberKey].old == protobuf.CommunityMember_ROLE_TOKEN_MASTER {
				// There is no need to check the permission, it was already satisfied without this new additional permission
				result.membersRoles[memberKey].new = protobuf.CommunityMember_ROLE_TOKEN_MASTER
				continue
			}

			communityPermissionsPreParsedData, channelPermissionsPreParsedData := PreParsePermissionsDataByType(changesByType.BecomeTokenMaster.Added, protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
			err = fetchCollectiblesIfNeeded(communityPermissionsPreParsedData, channelPermissionsPreParsedData)
			if err != nil {
				return nil, nil, err
			}

			becomeTokenMasterPermissions := communityPermissionsPreParsedData[protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER]
			permissionResponse, err := m.PermissionChecker.CheckPermissionsWithPreFetchedData(becomeTokenMasterPermissions, accountsAndChainIDs, true, collectiblesOwners)
			if err != nil {
				return nil, nil, err
			}

			if permissionResponse.Satisfied {
				result.membersRoles[memberKey].new = protobuf.CommunityMember_ROLE_TOKEN_MASTER
				// Skip further validation if user has TokenMaster permissions
				continue
			}
		}
	}

	return nil, nil, nil
}
