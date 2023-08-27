package raft

type Client interface {
	RequestVote(*VoteRequest) (*VoteResponse, error)
	AppendEntries()
}

type VoteRequest struct {
	Uri string

	Term         int64 `json:"term"`
	CandidateId  int64 `json:"candidate_id"`
	LastLogIndex int64 `json:"last_log_index"`
	LastLogTerm  int64 `json:"last_log_term"`
}

type VoteResponse struct {
	Term        int64 `json:"term"`
	VoteGranted bool  `json:"vote_granted"`
}
