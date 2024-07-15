package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/din-mukhammed/simple-raft/internal/entities"
)

var httpClient = http.Client{
	Timeout: time.Second,
}

type Client struct {
	uri string
}

func New(uri string) Client {
	return Client{
		uri: uri,
	}

}

func (c Client) RequestVote(voteReq entities.VoteRequest) (*entities.VoteResponse, error) {
	data, err := json.Marshal(voteReq)
	if err != nil {
		return nil, fmt.Errorf("marshalling: %w", err)
	}
	req, err := http.NewRequest(http.MethodPut, c.voteUrl(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not ok: %v", string(bb))
	}

	var vr entities.VoteResponse
	if err := json.Unmarshal(bb, &vr); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &vr, nil
}

func (c Client) voteUrl() string {
	return fmt.Sprintf("%s/vote", c.uri)
}

func (c Client) AppendEntries(
	aeReq entities.AppendEntriesRequest,
) (*entities.AppendEntriesResponse, error) {
	data, err := json.Marshal(aeReq)
	if err != nil {
		return nil, fmt.Errorf("marshalling: %w", err)
	}
	req, err := http.NewRequest(http.MethodPut, c.appendUrl(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not ok: %v", string(bb))
	}

	var ar entities.AppendEntriesResponse
	if err := json.Unmarshal(bb, &ar); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &ar, nil
}

func (c Client) appendUrl() string {
	return fmt.Sprintf("%s/append", c.uri)
}
