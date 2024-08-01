package raft

import (
	"context"

	"github.com/din-mukhammed/simple-raft/internal/entities"
)

type mockClient struct {
	rt *Service
}

func (m *mockClient) Addr() string { return "" }

func (m *mockClient) AppendEntries(
	ctx context.Context,
	req entities.AppendEntriesRequest,
) (*entities.AppendEntriesResponse, error) {
	res, err := m.rt.OnAppendEntries(req)
	return &res, err
}

func (m *mockClient) RequestVote(
	ctx context.Context,
	req entities.VoteRequest,
) (*entities.VoteResponse, error) {
	res, err := m.rt.OnRequestVote(req)
	return &res, err
}
