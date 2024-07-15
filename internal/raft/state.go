package raft

import "github.com/din-mukhammed/simple-raft/internal/entities"

type state struct {
	id int

	currentTerm  int
	votedFor     int
	logs         entities.Logs // logs repo
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

func (s state) lastLogInd() int {
	n := len(s.logs)
	if n == 0 {
		return 0
	}
	return s.logs[n-1].Ind
}

func (s state) lastLogTerm() int {
	n := len(s.logs)
	if n == 0 {
		return 0
	}
	return s.logs[n-1].Term
}
