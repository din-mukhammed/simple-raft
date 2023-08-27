package main

import (
	"fmt"
	"simple-raft/pkg/raft/server"
	"sync"
)

func main() {
	n := 3 // TODO: use cfg
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		j := i
		wg.Add(1)
		go func() {
			server.
				New(int64(j), fmt.Sprintf(":808%d", j)).
				Serve()
			wg.Done()
		}()
	}

	wg.Wait()
}
