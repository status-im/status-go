package localnotifications

const (
	CategoryTransaction            PushCategory = "transaction"
	CategoryMessage                PushCategory = "newMessage"
	CategoryGroupInvite            PushCategory = "groupInvite"
	CategoryCommunityRequestToJoin              = "communityRequestToJoin"
	CategoryCommunityJoined                     = "communityJoined"

	TypeTransaction NotificationType = "transaction"
	TypeMessage     NotificationType = "message"
)
