package communities

import "errors"

var ErrChatNotFound = errors.New("chat not found")
var ErrOrgNotFound = errors.New("community not found")
var ErrChatAlreadyExists = errors.New("chat already exists")
var ErrCantRequestAccess = errors.New("can't request access")
var ErrInvalidCommunityDescription = errors.New("invalid community description")
var ErrInvalidCommunityDescriptionNoOrgPermissions = errors.New("invalid community description no org permissions")
var ErrInvalidCommunityDescriptionNoChatPermissions = errors.New("invalid community description no chat permissions")
var ErrInvalidCommunityDescriptionUnknownChatAccess = errors.New("invalid community description unknown chat access")
var ErrInvalidCommunityDescriptionUnknownOrgAccess = errors.New("invalid community description unknown org access")
var ErrInvalidCommunityDescriptionMemberInChatButNotInOrg = errors.New("invalid community description member in chat but not in org")
var ErrNotAdmin = errors.New("no admin privileges for this community")
var ErrInvalidGrant = errors.New("invalid grant")
var ErrNotAuthorized = errors.New("not authorized")
var ErrInvalidMessage = errors.New("invalid community description message")
