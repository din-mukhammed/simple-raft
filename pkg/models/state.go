package models

import (
	"fmt"
)

type Status int

const (
	Follower Status = iota
	Candidate
	Leader
)

func (s Status) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Follower"
	case Leader:
		return "Leader"
	default:
		panic(fmt.Sprintf("Unknown status: %d", s))
	}
}

type Log struct {
	Term int64
	Val  string
}

type State struct {
	// Persistent state
	CurrentTerm int64
	VotedFor    int64
	Logs        []Log

	// Volatile state
	CommitIndex int64
	LastApplied int64

	// Volatile state for leaders
	NextIndex  []int64
	MatchIndex []int64

	Status Status
}

func NewState() *State {
	st := &State{
		CurrentTerm: 0,
		VotedFor:    -1,
		Logs:        make([]Log, 1),

		CommitIndex: 0,
		LastApplied: 0,

		// TODO: may be nextIndex, matchIndex no need?
		Status: Follower,
	}

	return st
}

func (s *State) LastLogIndex() int64 {
	return int64(len(s.Logs) - 1)
}

func (s *State) LastLogTerm() int64 {
	return s.Logs[s.LastLogIndex()].Term
}

func (s *State) IsCandidate() bool {
	return s.Status == Candidate
}

func (s *State) IsFollower() bool {
	return s.Status == Follower
}

func (s *State) IsLeader() bool {
	return s.Status == Leader
}
