package types

// DisputeStatus defines the status of a dispute
type DisputeStatus int32

const (
	DisputeStatus_UNSPECIFIED DisputeStatus = 0
	DisputeStatus_OPEN        DisputeStatus = 1
	DisputeStatus_RESOLVED    DisputeStatus = 2
	DisputeStatus_FAILED      DisputeStatus = 3
)

// AppealStatus defines the status of an appeal
type AppealStatus int32

const (
	AppealStatus_UNSPECIFIED AppealStatus = 0
	AppealStatus_PENDING     AppealStatus = 1
	AppealStatus_APPROVED    AppealStatus = 2
	AppealStatus_REJECTED    AppealStatus = 3
)

// DisputeVoteOption defines the vote option for a dispute
type DisputeVoteOption int32

const (
	DisputeVoteOption_UNSPECIFIED DisputeVoteOption = 0
	DisputeVoteOption_YES         DisputeVoteOption = 1
	DisputeVoteOption_NO          DisputeVoteOption = 2
	DisputeVoteOption_ABSTAIN     DisputeVoteOption = 3
)
