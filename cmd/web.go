package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/din-mukhammed/simple-raft/internal/client"
	"github.com/din-mukhammed/simple-raft/internal/controller"
	"github.com/din-mukhammed/simple-raft/internal/raft"
	"github.com/din-mukhammed/simple-raft/internal/repositories/logs"
	"github.com/din-mukhammed/simple-raft/pkg/config"
	"github.com/spf13/cobra"
)

var (
	httpRaftCmd = &cobra.Command{
		Use:   "http",
		Short: "Starts raft http server",
		Run: func(cmd *cobra.Command, args []string) {
			var (
				ctx = context.Background()

				id    = config.Viper().GetInt("SERVER_ID")
				name  = config.Viper().GetString("SERVER_NAME")
				port  = config.Viper().GetInt("APPLICATION_PORT")
				ss    = config.Viper().GetStringSlice("http_addrs")
				nodes = []raft.Node{}
			)
			for i, s := range ss {
				c := client.NewHTTPClient(s)
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
				cntr = controller.NewHTTPController(rt)

				mux = http.NewServeMux()
			)
			cntr.AddRoutes(mux)

			web := &http.Server{
				Addr:    fmt.Sprintf("127.0.0.1:%d", port),
				Handler: mux,
			}

			go func() {
				slog.Info("starting web server")
				if err := web.ListenAndServe(); err != nil {
					slog.Error("http serve listener", "err", err)
				}
			}()

			go rt.Start()

			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)
			signal.Notify(signalChan, syscall.SIGTERM)

			select {
			case sc := <-signalChan:
				slog.Info("signal found", "value", sc.String())
			case <-ctx.Done():
				slog.Info("ctx is done")
			}

			slog.Info("terminating web server...")
			if err := web.Shutdown(ctx); err != nil {
				slog.Error("server shutdown", "err", err)
				return
			}
			slog.Error("web server terminated")

		},
	}
)

func init() {
	rootCmd.AddCommand(httpRaftCmd)
}
