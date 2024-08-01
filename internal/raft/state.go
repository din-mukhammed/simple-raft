package raft

import (
	"sync"
)

type state struct {
	id int

	// should be persistent
	currentTerm  int
	votedFor     int
	logs         LogsRepo
	commitLength int

	mu              sync.RWMutex
	currentLeaderId int
	status          int
	votesReceived   map[int]struct{}
	sentLength      map[int]int
	ackedLength     map[int]int
}

func (s *state) isFollower() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.status == followerStatus
}

func (s *state) isCandidate() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.status == candidateStatus
}

func (s *state) isLeader() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.status == leaderStatus
}

func (s *state) withLock(f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f()
}

func (s *state) getCurrentTerm() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.currentTerm
}

func (s *state) getCommitLength() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.commitLength
}

func (s *state) sentLengthById(id int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.sentLength[id]
}

func (s *state) ackedLengthById(id int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.ackedLength[id]
}
