package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/din-mukhammed/simple-raft/internal/entities"
)

var hc = http.Client{
	Timeout: time.Second,
}

type httpClient struct {
	uri string
}

func NewHTTPClient(uri string) httpClient {
	return httpClient{
		uri: uri,
	}

}

func (c httpClient) RequestVote(
	ctx context.Context,
	voteReq entities.VoteRequest,
) (*entities.VoteResponse, error) {
	data, err := json.Marshal(voteReq)
	if err != nil {
		return nil, fmt.Errorf("marshalling: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.voteUrl(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	resp, err := hc.Do(req)
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

func (c httpClient) voteUrl() string {
	return fmt.Sprintf("%s/vote", c.uri)
}

func (c httpClient) AppendEntries(
	ctx context.Context,
	aeReq entities.AppendEntriesRequest,
) (*entities.AppendEntriesResponse, error) {
	data, err := json.Marshal(aeReq)
	if err != nil {
		return nil, fmt.Errorf("marshalling: %w", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		c.appendUrl(),
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, err
	}
	resp, err := hc.Do(req)
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

func (c httpClient) appendUrl() string {
	return fmt.Sprintf("%s/append", c.uri)
}

func (c httpClient) Addr() string {
	return c.uri
}
