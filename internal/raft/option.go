package raft

import "time"

type Option func(*Service)

func WithNode(c Node) Option {
	return func(s *Service) {
		s.nodes = append(s.nodes, c)
	}
}

func WithNodes(cc []Node) Option {
	return func(s *Service) {
		s.nodes = cc
	}
}

func WithName(name string) Option {
	return func(s *Service) {
		s.name = name
	}
}

func WithId(id int) Option {
	return func(s *Service) {
		s.state.id = id
	}
}

func WithLogsRepo(lr LogsRepo) Option {
	return func(s *Service) {
		s.state.logs = lr
	}
}

func WithCandidateTimeout(t time.Duration) Option {
	return func(s *Service) {
		s.candidateTimeout = t
	}
}

func WithHeartbeatTick(t time.Duration) Option {
	return func(s *Service) {
		s.heartbeatTick = t
	}
}
