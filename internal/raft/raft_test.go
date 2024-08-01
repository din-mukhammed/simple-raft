package raft

import (
	"github.com/din-mukhammed/simple-raft/internal/repositories/logs"
	. "gopkg.in/check.v1"

	"context"
	"testing"
	"time"
)

func Test(t *testing.T) {
	TestingT(t)
}

type RaftSuite struct {
	defaultCtx context.Context
	defaultNow time.Time

	logRepo LogsRepo
}

var _ = Suite(&RaftSuite{
	defaultCtx: context.Background(),
	defaultNow: time.Now(),
	logRepo:    logs.New(),
})
