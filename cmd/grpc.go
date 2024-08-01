package main

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/din-mukhammed/simple-raft/api"
	"github.com/din-mukhammed/simple-raft/internal/client"
	"github.com/din-mukhammed/simple-raft/internal/controller"
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

				cntr = controller.NewGRPCController(rt)
			)

			var opts []grpc.ServerOption
			opts = append(opts, grpc.ChainUnaryInterceptor(interceptor.UnaryServerLogger()))
			gs := grpc.NewServer(opts...)
			api.RegisterRaftServiceServer(gs, cntr)

			go rt.Start()

			slog.Info("starting grpc server", "port", port)
			lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
			if err != nil {
				panic(fmt.Errorf("listen on port: %d, err: %w", port, err))
			}

			if err := gs.Serve(lis); err != nil {
				slog.Error("grpc serve", "err", err)
				return
			}
			// TODO: gracefull shutdown
		},
	}
)

func init() {
	rootCmd.AddCommand(grpcRaftCmd)
}
