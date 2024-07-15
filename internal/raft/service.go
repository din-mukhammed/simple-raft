package raft

import (
	"errors"
	"log/slog"
	"math/rand"
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

type RemoteClient interface {
	AppendEntries(entities.AppendEntriesRequest) (*entities.AppendEntriesResponse, error)
	RequestVote(entities.VoteRequest) (*entities.VoteResponse, error)
}

type State struct {
	id int

	currentTerm  int
	votedFor     int
	logs         entities.Logs // logs repo
	commitLength int

	currentLeaderId int
	status          int
	votesReceived   map[int]struct{}
	sentLength      map[int]int
	ackedLength     map[int]int
}

type Node struct {
	Id     int
	Client RemoteClient
}

type Service struct {
	name            string
	state           State
	candidateStopCh chan struct{}

	cc []Node // should be interface
}

func NewRaft(opts ...Option) *Service {
	srv := &Service{
		name:            "default",
		candidateStopCh: make(chan struct{}),
		state: State{
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
	t := time.NewTicker(1000 * time.Millisecond)
	defer t.Stop()

	for ; true; <-t.C {
		if !s.isLeader() {
			continue
		}
		for _, c := range s.cc {
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
			if s.isFollower() {
				s.toCandidate()
			}
		case <-s.candidateStopCh:
			t.Reset(d)
		}
	}
}

func (s *Service) isFollower() bool {
	return s.state.status == followerStatus
}

func (s *Service) isCandidate() bool {
	return s.state.status == candidateStatus
}

func (s *Service) isLeader() bool {
	return s.state.status == leaderStatus
}

func (s *Service) lastLogInd() int {
	n := len(s.state.logs)
	if n == 0 {
		return 0
	}
	return s.state.logs[n-1].Ind
}

func (s *Service) lastLogTerm() int {
	n := len(s.state.logs)
	if n == 0 {
		return 0
	}
	return s.state.logs[n-1].Term
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
	// slog.Info("response from repliation", "to", c.Id, "resp", ae)
	if s.state.currentTerm == ae.Term && s.isLeader() {
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
		s.resetCandidateTimer()
	}
	return nil
}

func (s *Service) commitEntries() {
	for s.state.commitLength < s.state.logs.Len() {
		acks := 0
		for _, c := range s.cc {
			if s.state.ackedLength[c.Id] > s.state.commitLength {
				acks++
			}
		}

		if acks >= (len(s.cc)+1)/2 {
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
	)
	ct := s.state.currentTerm

	for _, c := range s.cc {
		if c.Id == s.state.id {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()

			vr, err := c.Client.RequestVote(entities.VoteRequest{
				Term:        s.state.currentTerm,
				CandidateId: s.state.id,
				LastLogInd:  s.lastLogInd(),
				LastLogTerm: s.lastLogTerm(),
			})
			if err != nil {
				slog.Error("request vote", "err", err)
				return
			}
			if s.isCandidate() && vr.VoteGranted && vr.Term == ct {
				s.addVote(c.Id)
				return
			}
			if vr.Term > ct {
				s.state.votedFor = -1
				s.state.status = followerStatus
				s.state.currentTerm = vr.Term

				s.resetCandidateTimer()

				return
			}
		}()
	}
	// TODO: don't wait all
	wg.Wait()

	totalVotes := s.totalVotes()
	if totalVotes >= (len(s.cc)+1)/2 {
		s.toLeader()
		return
	}
	slog.Info("not enough votes to become leader", "name", s.name, "got votes", totalVotes)
}

func (s *Service) resetCandidateTimer() {
	s.candidateStopCh <- struct{}{}
}

func (s *Service) RcvRequestVote(
	req entities.VoteRequest,
) (int, bool, error) {
	s.resetCandidateTimer()

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

	lt := s.lastLogTerm()
	logOk := (lastLogTerm > lt) || (lastLogTerm == lt && lastLogInd >= s.lastLogInd())

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
		s.resetCandidateTimer()
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

func (s *Service) RcvBroadcastMsg(req entities.BroadcastMsg) error {
	if !s.isLeader() {
		return errors.New("implement forwarding to leader")
	}

	s.state.logs.Append(entities.Log{
		Ind:  len(s.state.logs),
		Term: s.state.currentTerm,
		Msg:  req.Msg,
	})

	s.state.ackedLength[s.state.id] = s.state.logs.Len()

	wg := sync.WaitGroup{}
	for _, c := range s.cc {
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

	return nil
}
