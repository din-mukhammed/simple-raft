package models

type Server struct {
	state *State
}

func (s *Server) AppendEntries() {}
func (s *Server) RequestVote()   {}
