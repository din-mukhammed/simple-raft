package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"simple-raft/pkg/raft"
	"simple-raft/pkg/raft/client"
	"syscall"

	log "github.com/mgutz/logxi/v1"

	"github.com/gorilla/mux"
)

type Interface interface {
	Serve()
}

type impl struct {
	Id   int64
	Addr string
}

func New(id int64, addr string) *impl {
	return &impl{id, addr}
}

func (s *impl) Serve() {
	var (
		router = mux.NewRouter()
		web    = http.Server{
			Addr:    s.Addr,
			Handler: router,
		}

		serverContext = context.Background()
		signalChan    = make(chan os.Signal, 1)
	)

	go func() {
		signal.Notify(signalChan, os.Interrupt)
		signal.Notify(signalChan, syscall.SIGTERM)
		log.Info("assigned signal handlers")

		if s, ok := <-signalChan; ok {
			log.Info("signal found", "value", s.String())
		} else {
			log.Info("signal channel closed")
		}
		log.Info("terminating web server...")

		if err := web.Shutdown(serverContext); err != nil {
			log.Error("web server shutdown error", "error", err)
		}
		log.Info("web server terminated")
	}()

	raft.New(s.Id, client.New())
	router.HandleFunc("/append", AppendEntriesHandler)
	router.HandleFunc("/vote", RequestVoteHandler)

	log.Info("starting web server ", "addr", s.Addr)
	if err := web.ListenAndServe(); err != http.ErrServerClosed {
		panic(err)
	}
}
