package client

import (
	"context"
	"time"

	"github.com/din-mukhammed/simple-raft/api"
	"github.com/din-mukhammed/simple-raft/internal/entities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcClient struct {
	client api.RaftServiceClient

	addr string
}

func NewGRPCClient(addr string) (grpcClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return grpcClient{}, err
	}

	c := api.NewRaftServiceClient(conn)
	return grpcClient{
		client: c,
		addr:   addr,
	}, nil
}

func (g grpcClient) RequestVote(
	ctx context.Context,
	voteReq entities.VoteRequest,
) (*entities.VoteResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	resp, err := g.client.RequestVote(reqCtx, &api.VoteRequest{
		Term:        int64(voteReq.Term),
		CandidateId: int64(voteReq.CandidateId),
		LastLogInd:  int64(voteReq.LastLogInd),
		LastLogTerm: int64(voteReq.LastLogTerm),
	})
	if err != nil {
		return nil, err
	}

	return &entities.VoteResponse{
		Term:        int(resp.GetTerm()),
		VoteGranted: resp.VoteGranted,
	}, nil
}

func (g grpcClient) AppendEntries(
	ctx context.Context,
	aeReq entities.AppendEntriesRequest,
) (*entities.AppendEntriesResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	ll := make([]*api.Log, 0, aeReq.Suffix.Len())
	for _, l := range aeReq.Suffix {
		ll = append(ll, &api.Log{
			Term: int64(l.Term),
			Ind:  int64(l.Ind),
			Msg:  l.Msg,
		})
	}

	resp, err := g.client.AppendEntries(reqCtx, &api.AppendEntriesRequest{
		Term:         int64(aeReq.Term),
		LeaderId:     int64(aeReq.LeaderId),
		PrefixLength: int64(aeReq.LeaderId),
		PrefixTerm:   int64(aeReq.PrefixTerm),
		Suffix:       ll,
		CommitLength: int64(aeReq.CommitLength),
	})
	if err != nil {
		return nil, err
	}

	return &entities.AppendEntriesResponse{
		Term:     int(resp.GetTerm()),
		Success:  resp.Success,
		NumAcked: int(resp.GetNumAcked()),
	}, nil
}

func (g grpcClient) Addr() string {
	return g.addr
}
