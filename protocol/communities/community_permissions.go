package communities

import (
	"github.com/status-im/status-go/protocol/protobuf"
)

func (o *Community) CreateCommunityPermission(permissionID string, isAllowedTo int32, private bool, chatIDs []string) (*Permission, *CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, nil, ErrNotAdmin
	}

	if o.config.CommunityDescription.CommunityPermissions == nil {
		o.config.CommunityDescription.CommunityPermissions = make(map[string]*protobuf.CommunityPermission)
	}
	if _, ok := o.config.CommunityDescription.CommunityPermissions[permissionID]; ok {
		return nil, nil, ErrCommunityPermissionAlreadyExists
	}

	for _, cid := range chatIDs {
		_, exists := o.config.CommunityDescription.Chats[cid]
		if !exists {
			return nil, nil, ErrChatNotFound
		}
	}

	changes := o.emptyCommunityChanges()

	o.config.CommunityDescription.CommunityPermissions[permissionID] = &protobuf.CommunityPermission{
		CommunityId:  o.IDString(),
		PermissionId: permissionID,
		IsAllowedTo:  protobuf.CommunityPermission_AllowedTypes(isAllowedTo),
		Private:      private,
		ChatIds:      chatIDs,
	}

	permission := &Permission{
		PermissionID: permissionID,
		IsAllowedTo:  protobuf.CommunityPermission_AllowedTypes(isAllowedTo),
		Private:      private,
		ChatIds:      chatIDs,
	}

	o.config.Permissions = append(o.config.Permissions, permission)

	for _, cid := range chatIDs {
		hasPermission := false
		for _, pid := range o.config.CommunityDescription.Chats[cid].PermissionIds {
			if pid == permissionID {
				hasPermission = true
			}
		}
		if !hasPermission {
			o.config.CommunityDescription.Chats[cid].PermissionIds = append(o.config.CommunityDescription.Chats[cid].PermissionIds, permissionID)
		}
	}

	o.increaseClock()

	changes.CommunityPermissionsAdded[permissionID] = o.config.CommunityDescription.CommunityPermissions[permissionID]

	return permission, changes, nil

}

func (o *Community) UpdateCommunityPermission(permissionID string, communityID string, private bool, isAllowedTo int32, chatIDs []string) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	if _, ok := o.config.CommunityDescription.CommunityPermissions[permissionID]; !ok {
		return nil, ErrPermissionNotFound
	}

	for _, cid := range o.config.CommunityDescription.CommunityPermissions[permissionID].ChatIds {
		c, exists := o.config.CommunityDescription.Chats[cid]
		if !exists {
			return nil, ErrChatNotFound
		}

		for _, id := range c.PermissionIds {
			if id == permissionID {

			}
		}
	}

	changes := o.emptyCommunityChanges()

	o.config.CommunityDescription.CommunityPermissions[permissionID] = &protobuf.CommunityPermission{
		CommunityId:  communityID,
		PermissionId: permissionID,
		IsAllowedTo:  protobuf.CommunityPermission_AllowedTypes(isAllowedTo),
		Private:      private,
		ChatIds:      chatIDs,
	}

	for _, cid := range chatIDs {
		hasPermission := false
		for _, pid := range o.config.CommunityDescription.Chats[cid].PermissionIds {
			if pid == permissionID {
				hasPermission = true
			}
		}
		if !hasPermission {
			o.config.CommunityDescription.Chats[cid].PermissionIds = append(o.config.CommunityDescription.Chats[cid].PermissionIds, permissionID)
		}
	}

	o.increaseClock()

	changes.CommunityPermissionsModified[permissionID] = o.config.CommunityDescription.CommunityPermissions[permissionID]

	return changes, nil

}

func (o *Community) DeleteCommunityPermission(permissionID string) (*CommunityChanges, error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.config.PrivateKey == nil {
		return nil, ErrNotAdmin
	}

	if _, exists := o.config.CommunityDescription.CommunityPermissions[permissionID]; !exists {
		return nil, ErrPermissionNotFound
	}

	changes := o.emptyCommunityChanges()

	delete(o.config.CommunityDescription.CommunityPermissions, permissionID)

	changes.CommunityPermissionsRemoved = append(changes.CommunityPermissionsRemoved, permissionID)

	o.increaseClock()

	return changes, nil
}
