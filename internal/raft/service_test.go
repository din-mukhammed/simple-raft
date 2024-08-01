package raft

import (
	. "gopkg.in/check.v1"

	"time"
)

func (s *RaftSuite) TestLeaderTransformation(c *C) {
	candidateTimeout := 100 * time.Millisecond
	heartbeatTick := 5 * time.Millisecond
	rt0 := New(
		WithName("node-0"),
		WithId(0),
		WithLogsRepo(s.logRepo),
		WithCandidateTimeout(candidateTimeout),
		WithHeartbeatTick(heartbeatTick),
	)
	rt1 := New(
		WithName("node-1"),
		WithId(1),
		WithLogsRepo(s.logRepo),
		WithCandidateTimeout(candidateTimeout*2),
		WithHeartbeatTick(heartbeatTick),
	)
	rt2 := New(
		WithName("node-2"),
		WithId(2),
		WithLogsRepo(s.logRepo),
		WithCandidateTimeout(candidateTimeout*2),
		WithHeartbeatTick(heartbeatTick),
	)

	nodes := []Node{
		{
			Id:     0,
			Client: &mockClient{rt0},
		},
		{
			Id:     1,
			Client: &mockClient{rt1},
		},
		{
			Id:     2,
			Client: &mockClient{rt2},
		},
	}
	rt0.nodes = nodes
	rt1.nodes = nodes
	rt2.nodes = nodes

	go rt0.Start()
	go rt1.Start()
	go rt2.Start()

	time.Sleep(2 * candidateTimeout)

	c.Assert(rt0.state.isLeader(), Equals, true)
	c.Assert(rt1.state.isFollower(), Equals, true)
	c.Assert(rt2.state.isFollower(), Equals, true)
}
