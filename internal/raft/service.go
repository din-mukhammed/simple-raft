package raft

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/din-mukhammed/simple-raft/internal/entities"
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
	AppendEntries(
		context.Context,
		entities.AppendEntriesRequest,
	) (*entities.AppendEntriesResponse, error)
	RequestVote(context.Context, entities.VoteRequest) (*entities.VoteResponse, error)
	Addr() string
}

type LogsRepo interface {
	Add(entities.Log)
	Len() int
	Suffix(int) entities.Logs
	Get(int) entities.Log
	Cut(int)
	LastLogInd() int
	LastLogTerm() int
}

type Node struct {
	Id     int
	Client RemoteClient
}

type Service struct {
	name            string
	state           state
	candidateStopCh chan struct{}

	candidateTimeout time.Duration
	heartbeatTick    time.Duration

	nodes []Node // should be interface
}

func New(opts ...Option) *Service {
	srv := &Service{
		name:             "default",
		candidateStopCh:  make(chan struct{}),
		candidateTimeout: 2 * time.Second,
		heartbeatTick:    time.Second,
		state: state{
			votedFor:      -1,
			ackedLength:   map[int]int{},
			sentLength:    map[int]int{},
			votesReceived: map[int]struct{}{},
		},
	}

	for _, o := range opts {
		o(srv)
	}

	return srv
}

func (s *Service) Start() {
	go s.startCandidateTicker()
	s.startHearbeating()
}

func (s *Service) startHearbeating() {
	t := time.NewTicker(s.heartbeatTick)
	defer t.Stop()

	for ; true; <-t.C {
		if !s.state.isLeader() {
			continue
		}
		wg := sync.WaitGroup{}
		for _, c := range s.nodes {
			if c.Id == s.state.id {
				continue
			}

			s.state.withLock(func() {
				s.state.sentLength[c.Id] = s.state.logs.Len()
				s.state.ackedLength[c.Id] = 0
			})

			wg.Add(1)
			go func() {
				defer wg.Done()

				err := s.replicateLog(c)
				if err != nil {
					slog.Error("replicate log", "client", c.Id, "err", err)
					return
				}
			}()
		}

		wg.Wait()
	}
}

func (s *Service) startCandidateTicker() {
	d := s.candidateTimeout + time.Duration(rand.Intn(10)*int(time.Millisecond))
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
	slog.Info("starting election", "me", s.name)
	s.state.withLock(func() {
		s.state.status = candidateStatus
		s.state.currentTerm++
		s.state.votedFor = s.state.id
		s.state.votesReceived[s.state.id] = struct{}{}
	})

	s.requestVotes()
}

func (s *Service) toLeader() {
	slog.Info("I'M LEADER NOW", "name", s.name, "got votes", len(s.state.votesReceived))
	s.state.withLock(func() {
		s.state.status = leaderStatus
		s.state.currentLeaderId = s.state.id
	})
}

func (s *Service) resetElectionTimer() {
	s.candidateStopCh <- struct{}{}
}

func (s *Service) replicateLog(c Node) error {
	var (
		prefixLen   = s.state.sentLengthById(c.Id)
		suffix      = s.state.logs.Suffix(prefixLen)
		prefixTerm  = 0
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	)
	defer cancel()
	if prefixLen > 0 {
		prefixTerm = s.state.logs.Get(prefixLen - 1).Term
	}
	// ReplicateLog
	ae, err := c.Client.AppendEntries(ctx, entities.AppendEntriesRequest{
		LeaderId:     s.state.id,
		Term:         s.state.getCurrentTerm(),
		PrefixLength: prefixLen,
		PrefixTerm:   prefixTerm,
		Suffix:       suffix,
		CommitLength: s.state.commitLength,
	})
	if err != nil {
		return err
	}
	if s.state.getCurrentTerm() == ae.Term && s.state.isLeader() {
		if ae.Success && ae.NumAcked >= s.state.ackedLengthById(c.Id) {
			s.state.withLock(func() {
				s.state.sentLength[c.Id] = ae.NumAcked
				s.state.ackedLength[c.Id] = ae.NumAcked

				s.commitEntries()
			})
		} else if s.state.sentLengthById(c.Id) > 0 {
			s.state.withLock(func() {
				s.state.sentLength[c.Id] = s.state.sentLength[c.Id] - 1
			})
			return s.replicateLog(c)
		}
	} else if s.state.getCurrentTerm() < ae.Term {
		s.state.withLock(func() {
			s.state.status = followerStatus
			s.state.currentTerm = ae.Term
			s.state.votedFor = -1
		})
		s.resetElectionTimer()
	}
	return nil
}

// runs under lock
func (s *Service) commitEntries() {
	for s.state.commitLength < s.state.logs.Len() {
		acks := 0
		for _, c := range s.nodes {
			if s.state.ackedLength[c.Id] > s.state.commitLength {
				acks++
			}
		}

		if acks >= (len(s.nodes)+1)/2 {
			slog.Info("committed log", "msg", s.state.logs.Get(s.state.commitLength))
			s.state.commitLength++
		}
	}
}

func (s *Service) addVote(id int) {
	s.state.withLock(func() {
		s.state.votesReceived[id] = struct{}{}
	})
}

func (s *Service) totalVotes() int {
	res := 0
	s.state.withLock(func() {
		res = len(s.state.votesReceived)
	})
	return res
}

func (s *Service) requestVotes() {
	var (
		wg          = sync.WaitGroup{}
		ct          = s.state.currentTerm
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	)
	defer cancel()

	for _, c := range s.nodes {
		if c.Id == s.state.id {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()

			vr, err := c.Client.RequestVote(ctx, entities.VoteRequest{
				Term:        s.state.getCurrentTerm(),
				CandidateId: s.state.id,
				LastLogInd:  s.state.logs.LastLogInd(),
				LastLogTerm: s.state.logs.LastLogTerm(),
			})
			if err != nil {
				slog.Error("request vote", "err", err)
				return
			}
			if s.state.isCandidate() && vr.VoteGranted && vr.Term == s.state.getCurrentTerm() {
				s.addVote(c.Id)
				return
			}
			if vr.Term > ct {
				s.state.withLock(func() {
					s.state.votedFor = -1
					s.state.status = followerStatus
					s.state.currentTerm = vr.Term
				})

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

func (s *Service) OnRequestVote(
	req entities.VoteRequest,
) (entities.VoteResponse, error) {
	s.resetElectionTimer()
	var (
		candidateTerm = req.Term
		candidateId   = req.CandidateId
		lastLogInd    = req.LastLogInd
		lastLogTerm   = req.LastLogTerm
	)

	s.state.withLock(func() {
		if candidateTerm > s.state.currentTerm {
			s.state.currentTerm = candidateTerm
			s.state.status = followerStatus
			s.state.votedFor = -1
		}
	})

	lt := s.state.logs.LastLogTerm()
	logOk := (lastLogTerm > lt) || (lastLogTerm == lt && lastLogInd >= s.state.logs.LastLogInd())

	if candidateTerm == s.state.getCurrentTerm() && logOk &&
		(s.state.votedFor == -1 || s.state.votedFor == candidateId) {
		s.state.votedFor = candidateId
		return entities.VoteResponse{
			Term:        s.state.getCurrentTerm(),
			VoteGranted: true,
		}, nil
	}

	return entities.VoteResponse{
		Term:        s.state.getCurrentTerm(),
		VoteGranted: false,
	}, nil
}

func (s *Service) OnAppendEntries(
	req entities.AppendEntriesRequest,
) (entities.AppendEntriesResponse, error) {
	s.state.withLock(func() {
		if req.Term > s.state.currentTerm {
			s.state.currentTerm = req.Term
		}

		if s.state.currentTerm == req.Term {
			s.state.status = followerStatus
			s.state.currentLeaderId = req.LeaderId
			s.resetElectionTimer()
		}
	})

	logOk := (s.state.logs.Len() >= req.PrefixLength) &&
		(req.PrefixLength == 0 || s.state.logs.Get(req.PrefixLength-1).Term == req.Term)

	if req.Term == s.state.getCurrentTerm() && logOk {
		s.state.withLock(func() {
			s.appendEntries(req.PrefixLength, req.CommitLength, req.Suffix)
		})
		acked := len(req.Suffix) + req.PrefixLength
		return entities.AppendEntriesResponse{
			Term:     s.state.getCurrentTerm(),
			Success:  true,
			NumAcked: acked,
		}, nil
	}
	return entities.AppendEntriesResponse{
		Term:     s.state.currentTerm,
		Success:  false,
		NumAcked: 0,
	}, nil
}

// runs under lock
func (s *Service) appendEntries(prefixLen, leaderCommit int, suffix entities.Logs) {
	if suffix.Len() > 0 && s.state.logs.Len() > prefixLen {
		ind := min(s.state.logs.Len(), prefixLen+suffix.Len()) - 1
		if s.state.logs.Get(ind).Term != suffix[ind-prefixLen].Term {
			s.state.logs.Cut(ind)
		}
	}
	if prefixLen+suffix.Len() > s.state.logs.Len() {
		for i := s.state.logs.Len() - prefixLen; i < suffix.Len(); i++ {
			s.state.logs.Add(suffix[i])
		}
	}

	if leaderCommit > s.state.commitLength {
		for i := s.state.commitLength; i < leaderCommit; i++ {
			slog.Info(
				"commited msg",
				"log ind",
				s.state.logs.Get(i).Ind,
				"msg",
				s.state.logs.Get(i).Msg,
			)
		}
		s.state.commitLength = leaderCommit
	}
}

func (s *Service) BroadcastMsg(
	req entities.BroadcastMsg,
) (string, error) {
	if !s.state.isLeader() {
		c := s.clientById(s.state.currentLeaderId)
		if c == nil {
			return "", fmt.Errorf("unknown leader: %v", s.state.currentLeaderId)
		}
		return fmt.Sprintf(
			"%s/broadcast",
			c.Addr(),
		), ErrRedirectToLeader
	}

	s.state.logs.Add(entities.Log{
		Ind:  s.state.logs.Len(),
		Term: s.state.currentTerm,
		Msg:  req.Msg,
	})

	s.state.withLock(func() {
		s.state.ackedLength[s.state.id] = s.state.logs.Len()
	})

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
