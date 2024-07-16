package entities

type Logs []Log

func (ll Logs) Len() int {
	return len(ll)
}

type Log struct {
	Term int
	Ind  int
	Msg  string
}

type VoteRequest struct {
	Term        int `json:"term,omitempty"`
	CandidateId int `json:"candidate_id,omitempty"`
	LastLogInd  int `json:"last_log_ind,omitempty"`
	LastLogTerm int `json:"last_log_term,omitempty"`
}

type VoteResponse struct {
	Term        int  `json:"term,omitempty"`
	VoteGranted bool `json:"vote_granted,omitempty"`
}

type AppendEntriesRequest struct {
	Term         int  `json:"term,omitempty"`
	LeaderId     int  `json:"leader_id,omitempty"`
	PrefixLength int  `json:"prefix_length,omitempty"`
	PrefixTerm   int  `json:"prefix_term,omitempty"`
	Suffix       Logs `json:"suffix,omitempty"`
	CommitLength int  `json:"commit_length,omitempty"`
}

type AppendEntriesResponse struct {
	Term     int  `json:"term,omitempty"`
	Success  bool `json:"success,omitempty"`
	NumAcked int  `json:"num_acked,omitempty"`
}

type BroadcastMsg struct {
	Msg string `json:"msg,omitempty"`
}
