package raft

type state struct {
	id int

	currentTerm  int
	votedFor     int
	logs         LogsRepo
	commitLength int

	currentLeaderId int
	status          int
	votesReceived   map[int]struct{}
	sentLength      map[int]int
	ackedLength     map[int]int
}

func (s state) isFollower() bool {
	return s.status == followerStatus
}

func (s state) isCandidate() bool {
	return s.status == candidateStatus
}

func (s state) isLeader() bool {
	return s.status == leaderStatus
}
