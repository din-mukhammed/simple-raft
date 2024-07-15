package raft

type Option func(*Service)

func WithNode(c Node) Option {
	return func(s *Service) {
		s.cc = append(s.cc, c)
	}
}

func WithNodes(cc []Node) Option {
	return func(s *Service) {
		s.cc = cc
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
