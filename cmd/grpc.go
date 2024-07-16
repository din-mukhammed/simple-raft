package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/din-mukhammed/simple-raft/api"
	"github.com/din-mukhammed/simple-raft/internal/entities"
	"github.com/din-mukhammed/simple-raft/internal/raft"
	"google.golang.org/grpc"
)

type grpcServer struct {
	api.UnimplementedRaftServiceServer

	raftService *raft.Service
}

func NewGrpcService(raft *raft.Service) *grpcServer {
	return &grpcServer{
		raftService: raft,
	}

}

func (s *grpcServer) RequestVote(
	ctx context.Context,
	req *api.VoteRequest,
) (*api.VoteResponse, error) {
	resp, err := s.raftService.RcvRequestVote(entities.VoteRequest{
		Term:        int(req.GetTerm()),
		CandidateId: int(req.GetCandidateId()),
		LastLogInd:  int(req.GetLastLogInd()),
		LastLogTerm: int(req.GetLastLogTerm()),
	})
	if err != nil {
		return nil, err
	}

	return &api.VoteResponse{
		VoteGranted: resp.VoteGranted,
		Term:        int64(resp.Term),
	}, nil
}

func (s *grpcServer) AppendEntries(
	ctx context.Context,
	req *api.AppendEntriesRequest,
) (*api.AppendEntriesResponse, error) {
	ll := make(entities.Logs, 0, len(req.GetSuffix()))
	for _, l := range req.GetSuffix() {
		ll = append(ll, entities.Log{
			Term: int(l.GetTerm()),
			Ind:  int(l.GetInd()),
			Msg:  l.GetMsg(),
		})
	}
	resp, err := s.raftService.RcvAppendEntries(entities.AppendEntriesRequest{
		Term:         int(req.GetTerm()),
		LeaderId:     int(req.GetLeaderId()),
		PrefixLength: int(req.GetPrefixLength()),
		PrefixTerm:   int(req.GetPrefixTerm()),
		Suffix:       ll,
		CommitLength: int(req.GetCommitLength()),
	})
	if err != nil {
		return nil, err
	}

	return &api.AppendEntriesResponse{
		Term:     int64(resp.Term),
		Success:  resp.Success,
		NumAcked: int64(resp.NumAcked),
	}, nil
}

func (s *grpcServer) BroadcastMsg(
	ctx context.Context,
	req *api.BroadcastMsgRequest,
) (*api.BroadcastMsgResponse, error) {
	// TODO: handle redirect
	_, err := s.raftService.RcvBroadcastMsg(entities.BroadcastMsg{Msg: req.GetMsg()})
	if err != nil {
		return nil, err
	}
	return &api.BroadcastMsgResponse{}, nil
}

func (s *grpcServer) Run(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		panic(fmt.Errorf("listen on port: %d, err: %w", port, err))
	}

	var opts []grpc.ServerOption
	gs := grpc.NewServer(opts...)
	api.RegisterRaftServiceServer(gs, s)

	if err := gs.Serve(lis); err != nil {
		slog.Error("grpc serve", "err", err)
		return
	}
	// TODO: gracefull shutdown
}
