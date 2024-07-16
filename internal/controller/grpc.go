package controller

import (
	"context"

	"github.com/din-mukhammed/simple-raft/api"
	"github.com/din-mukhammed/simple-raft/internal/entities"
)

type RaftService interface {
	RequestVote(entities.VoteRequest) (entities.VoteResponse, error)
	AppendEntries(entities.AppendEntriesRequest) (entities.AppendEntriesResponse, error)
	BroadcastMsg(entities.BroadcastMsg) (string, error)
}

type grpcController struct {
	api.UnimplementedRaftServiceServer

	raftService RaftService
}

func NewGRPCController(raft RaftService) *grpcController {
	return &grpcController{
		raftService: raft,
	}
}

func (s *grpcController) RequestVote(
	ctx context.Context,
	req *api.VoteRequest,
) (*api.VoteResponse, error) {
	resp, err := s.raftService.RequestVote(entities.VoteRequest{
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

func (s *grpcController) AppendEntries(
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
	resp, err := s.raftService.AppendEntries(entities.AppendEntriesRequest{
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

func (s *grpcController) BroadcastMsg(
	ctx context.Context,
	req *api.BroadcastMsgRequest,
) (*api.BroadcastMsgResponse, error) {
	// TODO: handle redirect
	_, err := s.raftService.BroadcastMsg(entities.BroadcastMsg{Msg: req.GetMsg()})
	if err != nil {
		return nil, err
	}
	return &api.BroadcastMsgResponse{}, nil
}
