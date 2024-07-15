package raft

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/din-mukhammed/simple-raft/internal/entities"
	"github.com/din-mukhammed/simple-raft/pkg/config"
)

const (
	followerStatus = iota
	candidateStatus
	leaderStatus
)

var (
	ErrRedirectToLeader = errors.New("redirect to leader")
)

type RemoteClient interface {
	AppendEntries(entities.AppendEntriesRequest) (*entities.AppendEntriesResponse, error)
	RequestVote(entities.VoteRequest) (*entities.VoteResponse, error)
	Uri() string
}

type Node struct {
	Id     int
	Client RemoteClient
}

type Service struct {
	name            string
	state           state
	candidateStopCh chan struct{}

	nodes []Node // should be interface
}

func NewRaft(opts ...Option) *Service {
	srv := &Service{
		name:            "default",
		candidateStopCh: make(chan struct{}),
		state: state{
			logs:          make(entities.Logs, 0),
			votedFor:      -1,
			ackedLength:   map[int]int{},
			sentLength:    map[int]int{},
			votesReceived: map[int]struct{}{},
		},
	}

	for _, o := range opts {
		o(srv)
	}

	go srv.startCandidateTicker()
	go srv.startHearbeating()

	return srv
}

func (s *Service) startHearbeating() {
	t := time.NewTicker(config.Viper().GetDuration("heartbeat_duration"))
	defer t.Stop()

	for ; true; <-t.C {
		if !s.state.isLeader() {
			continue
		}
		for _, c := range s.nodes {
			if c.Id == s.state.id {
				continue
			}

			s.state.sentLength[c.Id] = len(s.state.logs)
			s.state.ackedLength[c.Id] = 0

			err := s.replicateLog(c)
			if err != nil {
				slog.Error("replicate log", "client", c.Id, "err", err)
				continue
			}
		}
	}
}

func (s *Service) startCandidateTicker() {
	d := config.Viper().
		GetDuration("initial_delay") +
		time.Duration(
			rand.Intn(100)*int(time.Millisecond),
		)
	t := time.NewTicker(d)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if s.state.isFollower() {
				s.toCandidate()
			}
		case <-s.candidateStopCh:
			t.Reset(d)
		}
	}
}

func (s *Service) toCandidate() {
	slog.Info("starting to be candidate", "me", s.name)
	s.state.status = candidateStatus
	s.state.currentTerm++
	s.state.votedFor = s.state.id
	s.state.votesReceived[s.state.id] = struct{}{}

	s.requestVotes()
}

func (s *Service) toLeader() {
	slog.Info("I'M LEADER NOW", "name", s.name, "got votes", len(s.state.votesReceived))
	s.state.status = leaderStatus
	s.state.currentLeaderId = s.state.id
}

func (s *Service) resetElectionTimer() {
	s.candidateStopCh <- struct{}{}
}

func (s *Service) replicateLog(c Node) error {
	prefixLen := s.state.sentLength[c.Id]
	suffix := s.state.logs[prefixLen:]
	prefixTerm := 0
	if prefixLen > 0 {
		prefixTerm = s.state.logs[prefixLen-1].Term
	}
	// ReplicateLog
	ae, err := c.Client.AppendEntries(entities.AppendEntriesRequest{
		LeaderId:     s.state.id,
		Term:         s.state.currentTerm,
		PrefixLength: prefixLen,
		PrefixTerm:   prefixTerm,
		Suffix:       suffix,
		CommitLength: s.state.commitLength,
	})
	if err != nil {
		return err
	}
	if s.state.currentTerm == ae.Term && s.state.isLeader() {
		if ae.Success && ae.NumAcked >= s.state.ackedLength[c.Id] {
			s.state.sentLength[c.Id] = ae.NumAcked
			s.state.ackedLength[c.Id] = ae.NumAcked
			s.commitEntries()
		} else if s.state.sentLength[c.Id] > 0 {
			s.state.sentLength[c.Id] = s.state.sentLength[c.Id] - 1
			return s.replicateLog(c)
		}
	} else if s.state.currentTerm < ae.Term {
		s.state.status = followerStatus
		s.state.currentTerm = ae.Term
		s.state.votedFor = -1
		s.resetElectionTimer()
	}
	return nil
}

func (s *Service) commitEntries() {
	for s.state.commitLength < s.state.logs.Len() {
		acks := 0
		for _, c := range s.nodes {
			if s.state.ackedLength[c.Id] > s.state.commitLength {
				acks++
			}
		}

		if acks >= (len(s.nodes)+1)/2 {
			slog.Info("committed log", "msg", s.state.logs[s.state.commitLength])
			s.state.commitLength++
		}
	}
}

func (s *Service) addVote(id int) {
	s.state.votesReceived[id] = struct{}{}
}

func (s *Service) totalVotes() int {

	return len(s.state.votesReceived)
}

func (s *Service) requestVotes() {
	var (
		wg = sync.WaitGroup{}
		ct = s.state.currentTerm
	)

	for _, c := range s.nodes {
		if c.Id == s.state.id {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()

			vr, err := c.Client.RequestVote(entities.VoteRequest{
				Term:        s.state.currentTerm,
				CandidateId: s.state.id,
				LastLogInd:  s.state.lastLogInd(),
				LastLogTerm: s.state.lastLogTerm(),
			})
			if err != nil {
				slog.Error("request vote", "err", err)
				return
			}
			if s.state.isCandidate() && vr.VoteGranted && vr.Term == ct {
				s.addVote(c.Id)
				return
			}
			if vr.Term > ct {
				s.state.votedFor = -1
				s.state.status = followerStatus
				s.state.currentTerm = vr.Term

				s.resetElectionTimer()

				return
			}
		}()
	}
	// TODO: don't wait all
	wg.Wait()

	totalVotes := s.totalVotes()
	if totalVotes >= (len(s.nodes)+1)/2 {
		s.toLeader()
		return
	}
	slog.Info("not enough votes to become leader", "name", s.name, "got votes", totalVotes)
}

func (s *Service) RcvRequestVote(
	req entities.VoteRequest,
) (int, bool, error) {
	s.resetElectionTimer()

	var (
		candidateTerm = req.Term
		candidateId   = req.CandidateId
		lastLogInd    = req.LastLogInd
		lastLogTerm   = req.LastLogTerm
	)

	if candidateTerm > s.state.currentTerm {
		s.state.currentTerm = candidateTerm
		s.state.status = followerStatus
		s.state.votedFor = -1
	}

	lt := s.state.lastLogTerm()
	logOk := (lastLogTerm > lt) || (lastLogTerm == lt && lastLogInd >= s.state.lastLogInd())

	if candidateTerm == s.state.currentTerm && logOk &&
		(s.state.votedFor == -1 || s.state.votedFor == candidateId) {
		s.state.votedFor = candidateId
		return s.state.currentTerm, true, nil
	}

	return s.state.currentTerm, false, nil
}

func (s *Service) RcvAppendEntries(
	req entities.AppendEntriesRequest,
) (int, bool, int, error) {
	if req.Term > s.state.currentTerm {
		s.state.currentTerm = req.Term
	}

	if s.state.currentTerm == req.Term {
		s.state.status = followerStatus
		s.state.currentLeaderId = req.LeaderId
		s.resetElectionTimer()
	}

	logOk := (s.state.logs.Len() >= req.PrefixLength) &&
		(req.PrefixLength == 0 || s.state.logs[req.PrefixLength-1].Term == req.Term)

	if req.Term == s.state.currentTerm && logOk {
		s.appendEntries(req.PrefixLength, req.CommitLength, req.Suffix)
		acked := len(req.Suffix) + req.PrefixLength
		return s.state.currentTerm, true, acked, nil
	}
	return s.state.currentTerm, false, 0, nil
}

func (s *Service) appendEntries(prefixLen, leaderCommit int, suffix entities.Logs) {
	if suffix.Len() > 0 && s.state.logs.Len() > prefixLen {
		ind := min(s.state.logs.Len(), prefixLen+suffix.Len()) - 1
		if s.state.logs[ind].Term != suffix[ind-prefixLen].Term {
			s.state.logs = s.state.logs[:ind]
		}
	}
	if prefixLen+suffix.Len() > s.state.logs.Len() {
		for i := s.state.logs.Len() - prefixLen; i < suffix.Len(); i++ {
			s.state.logs.Append(suffix[i])
		}
	}

	if leaderCommit > s.state.commitLength {
		for i := s.state.commitLength; i < leaderCommit; i++ {
			slog.Info("commited msg", "log ind", s.state.logs[i].Ind, "msg", s.state.logs[i].Msg)
		}
		s.state.commitLength = leaderCommit
	}
}

func (s *Service) RcvBroadcastMsg(
	w http.ResponseWriter,
	req entities.BroadcastMsg,
) (string, error) {
	if !s.state.isLeader() {
		c := s.clientById(s.state.currentLeaderId)
		if c == nil {
			return "", fmt.Errorf("unknown leader: %v", s.state.currentLeaderId)
		}
		return fmt.Sprintf(
			"%s/broadcast",
			c.Uri(),
		), ErrRedirectToLeader
	}

	s.state.logs.Append(entities.Log{
		Ind:  len(s.state.logs),
		Term: s.state.currentTerm,
		Msg:  req.Msg,
	})

	s.state.ackedLength[s.state.id] = s.state.logs.Len()

	wg := sync.WaitGroup{}
	for _, c := range s.nodes {
		if c.Id == s.state.id {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			err := s.replicateLog(c)
			if err != nil {
				slog.Error("replicating log", "to", c.Id, "err", err)
			}
		}()
	}

	wg.Wait()

	return "", nil
}

func (s *Service) clientById(id int) RemoteClient {
	for _, n := range s.nodes {
		if n.Id == id {
			return n.Client
		}
	}
	return nil
}
