package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/din-mukhammed/simple-raft/internal/client"
	"github.com/din-mukhammed/simple-raft/internal/entities"
	"github.com/din-mukhammed/simple-raft/internal/raft"
	"github.com/din-mukhammed/simple-raft/internal/repositories/logs"
	"github.com/din-mukhammed/simple-raft/pkg/config"
)

func Start(ctx context.Context) {
	var (
		id    = config.Viper().GetInt("SERVER_ID")
		name  = config.Viper().GetString("SERVER_NAME")
		port  = config.Viper().GetString("APPLICATION_PORT")
		ss    = config.Viper().GetStringSlice("servers")
		nodes = []raft.Node{}
	)
	for i, s := range ss {
		nodes = append(nodes, raft.Node{
			Id:     i,
			Client: client.New(s),
		})
	}
	rt := raft.New(
		raft.WithName(name),
		raft.WithNodes(nodes),
		raft.WithId(id),
		raft.WithLogsRepo(logs.New()),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /append", func(w http.ResponseWriter, r *http.Request) {
		var aeReq entities.AppendEntriesRequest

		bb, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := json.Unmarshal(bb, &aeReq); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		term, success, numAcked, err := rt.RcvAppendEntries(aeReq)

		bb, err = json.Marshal(entities.AppendEntriesResponse{
			Term:     term,
			Success:  success,
			NumAcked: numAcked,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(bb)
	})

	mux.HandleFunc("POST /broadcast", func(w http.ResponseWriter, r *http.Request) {
		bb, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var msg entities.BroadcastMsg
		if err := json.Unmarshal(bb, &msg); err != nil {
			slog.Error("unmarshal msg", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if redirectUrl, err := rt.RcvBroadcastMsg(w, msg); err != nil {
			if errors.Is(err, raft.ErrRedirectToLeader) {
				http.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
				return
			}
			slog.Error("broacast msg", "err", err, "msg", msg)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("PUT /vote", func(w http.ResponseWriter, r *http.Request) {
		var voteReq entities.VoteRequest

		bb, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := json.Unmarshal(bb, &voteReq); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		slog.Info("request", "cmd", "vote", "req", voteReq)
		term, success, err := rt.RcvRequestVote(voteReq)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		bb, err = json.Marshal(entities.VoteResponse{
			Term:        term,
			VoteGranted: success,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(bb)
		slog.Info("response", "cmd", "vote", "term", term, "ok", success, "err", err)
	})

	web := &http.Server{
		Addr:    "127.0.0.1:" + port,
		Handler: mux,
	}

	go func() {
		slog.Info("starting web server")
		if err := web.ListenAndServe(); err != nil {
			slog.Error("http serve listener", "err", err)
		}
	}()

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
}
