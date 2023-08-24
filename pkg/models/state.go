package models

type State struct {
	// Persistent state
	CurrentTerm int64
	VotedFor    int64
	Logs        []int64

	// Volatile state
	CommitIndex int64
	LastApplied int64

	// Volatile state for leaders
	NextIndex  []int64
	MatchIndex []int64
}
