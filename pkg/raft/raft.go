package raft

import (
	"fmt"
	"math/rand"
	"simple-raft/pkg/models"
	"sync"
	"time"

	log "github.com/mgutz/logxi/v1"
)

type server struct {
	mu sync.Mutex

	state         *models.State
	c             Client
	electionTimer *timer

	Id int64
}

func New(id int64, client Client) *server {
	sv := &server{
		Id:    id,
		state: models.NewState(),
		c:     client,
	}
	rand.Seed(id)
	sv.electionTimer = newTimer(
		time.Duration(2+rand.Intn(3))*time.Second,
		sv.toCandidate,
	)
	sv.electionTimer.Start()
	return sv
}

func (s *server) AppendEntries() {
}

func (s *server) toCandidate() {
	log.Debug("toCandidate() started", "id", s.Id, "state", s.state)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.state.IsFollower() {
		return
	}
	s.state.CurrentTerm++
	s.state.VotedFor = s.Id
	s.RequestVote()
	log.Debug("done request voting")
}

// TODO: synchronize
func (s *server) RequestVote() {
	// TODO: reset candidate timer
	for i := 0; i < 3; i++ { // TODO: use config
		req := &VoteRequest{
			Uri:          fmt.Sprintf("http://localhost:808%d", i), // fill with host addr from cfg
			Term:         s.state.CurrentTerm,
			CandidateId:  s.Id,
			LastLogIndex: s.state.LastLogIndex(),
			LastLogTerm:  s.state.LastLogTerm(),
		}
		resp, err := s.c.RequestVote(req)
		if err != nil {
			// log.Error("couldnt request vote", "err", err)
		}
		log.Debug("got response", "VoteResponse", resp)
	}
}

type timer struct {
	duration time.Duration
	ticker   *time.Ticker
	done     chan struct{}

	callback func()
}

func newTimer(duration time.Duration, callback func()) *timer {
	log.Debug("new timer", "duration", duration)
	return &timer{
		duration: duration,
		ticker:   time.NewTicker(duration),
		done:     make(chan struct{}),
		callback: callback,
	}
}

func (t *timer) Start() {
	go t.wait()
}

func (t *timer) Stop() {
	t.done <- struct{}{}
	t.ticker.Stop()
}

func (t *timer) wait() {
	for {
		select {
		case <-t.ticker.C:
			log.Debug("timer event occured, callback()")
			t.callback()
		case <-t.done:
			log.Debug("timer done")
			return
		}
	}
}
