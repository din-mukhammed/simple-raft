package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"simple-raft/pkg/raft"
)

type impl struct{}

func New() *impl {
	return &impl{}
}

func (c *impl) RequestVote(req *raft.VoteRequest) (*raft.VoteResponse, error) {
	bb, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(req.Uri+"/vote", "application/json", bytes.NewReader(bb))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var vote *raft.VoteResponse
	if err := json.Unmarshal(body, vote); err != nil {
		return nil, err
	}

	return vote, nil
}

func (c *impl) AppendEntries() {
}
