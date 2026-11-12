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
	CurrentSession().SendCommunityMemberReevaluationStarted(communityID)
}

// SendCommunityMemberReevaluationStarted sends in the session s.
func (s Session) SendCommunityMemberReevaluationStarted(communityID string) {
	s.send(MemberReevaluationStatus, CommunityMemberReevaluationSignal{CommunityID: communityID, Status: InProgress})
}

func SendCommunityMemberReevaluationEnded(communityID string) {
	CurrentSession().SendCommunityMemberReevaluationEnded(communityID)
}

// SendCommunityMemberReevaluationEnded sends in the session s.
func (s Session) SendCommunityMemberReevaluationEnded(communityID string) {
	s.send(MemberReevaluationStatus, CommunityMemberReevaluationSignal{CommunityID: communityID, Status: Done})
}
