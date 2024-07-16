package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/din-mukhammed/simple-raft/api"
	"github.com/din-mukhammed/simple-raft/internal/client"
	"github.com/din-mukhammed/simple-raft/internal/entities"
	"github.com/din-mukhammed/simple-raft/internal/raft"
	"github.com/din-mukhammed/simple-raft/internal/repositories/logs"
	"github.com/din-mukhammed/simple-raft/pkg/config"
	"github.com/din-mukhammed/simple-raft/pkg/interceptor"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	grpcRaftCmd = &cobra.Command{
		Use:   "grpc",
		Short: "Starts raft grpc server",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				id    = config.Viper().GetInt("SERVER_ID")
				name  = config.Viper().GetString("SERVER_NAME")
				port  = config.Viper().GetInt("APPLICATION_PORT")
				ss    = config.Viper().GetStringSlice("grpc_addrs")
				nodes = []raft.Node{}
			)
			for i, s := range ss {
				c, err := client.NewGRPCClient(s)
				if err != nil {
					panic(fmt.Errorf("create grpc client: %w", err))
				}
				nodes = append(nodes, raft.Node{
					Id:     i,
					Client: c,
				})
			}
			var (
				rt = raft.New(
					raft.WithName(name),
					raft.WithNodes(nodes),
					raft.WithId(id),
					raft.WithLogsRepo(logs.New()),
				)

				srv = NewGrpcService(rt)
			)

			slog.Info("starting grpc server", "port", port)
			lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
			if err != nil {
				panic(fmt.Errorf("listen on port: %d, err: %w", port, err))
			}

			var opts []grpc.ServerOption
			opts = append(opts, grpc.ChainUnaryInterceptor(interceptor.UnaryServerLogger()))
			gs := grpc.NewServer(opts...)
			api.RegisterRaftServiceServer(gs, srv)

			if err := gs.Serve(lis); err != nil {
				slog.Error("grpc serve", "err", err)
				return
			}
			// TODO: gracefull shutdown
		},
	}
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
	slog.Info("starting grpc server", "port", port)
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		panic(fmt.Errorf("listen on port: %d, err: %w", port, err))
	}

	var opts []grpc.ServerOption
	opts = append(opts, grpc.ChainUnaryInterceptor(interceptor.UnaryServerLogger()))
	gs := grpc.NewServer(opts...)
	api.RegisterRaftServiceServer(gs, s)

	if err := gs.Serve(lis); err != nil {
		slog.Error("grpc serve", "err", err)
		return
	}
	// TODO: gracefull shutdown
}

func init() {
	rootCmd.AddCommand(grpcRaftCmd)
}
