package signal

const (
	MemberReevaluationStatus = "community.memberReevaluationStatus"
)

type ReevaluationStatus uint

const (
	None ReevaluationStatus = iota
	InProgress
	Done
)

type CommunityMemberReevaluationSignal struct {
	CommunityID string             `json:"communityId"`
	Status      ReevaluationStatus `json:"status"`
}

func SendCommunityMemberReevaluationStarted(communityID string) {
	send(MemberReevaluationStatus, CommunityMemberReevaluationSignal{CommunityID: communityID, Status: InProgress})
}

func SendCommunityMemberReevaluationEnded(communityID string) {
	send(MemberReevaluationStatus, CommunityMemberReevaluationSignal{CommunityID: communityID, Status: Done})
}
