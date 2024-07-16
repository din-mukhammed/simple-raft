package logs

import "github.com/din-mukhammed/simple-raft/internal/entities"

type inMemoryRepo struct {
	entries entities.Logs
}

func New() *inMemoryRepo {
	return &inMemoryRepo{
		entries: make(entities.Logs, 0),
	}
}

func (r *inMemoryRepo) Add(l entities.Log) {
	r.entries = append(r.entries, l)
}

func (r *inMemoryRepo) Len() int {
	return len(r.entries)
}

func (r *inMemoryRepo) Suffix(from int) entities.Logs {
	return r.entries[from:]
}

func (r *inMemoryRepo) Get(ind int) entities.Log {
	return r.entries[ind]
}

func (r *inMemoryRepo) Cut(to int) {
	r.entries = r.entries[:to]
}

func (r inMemoryRepo) LastLogInd() int {
	n := r.Len()
	if n == 0 {
		return 0
	}
	return r.Get(n - 1).Ind
}

func (r inMemoryRepo) LastLogTerm() int {
	n := r.Len()
	if n == 0 {
		return 0
	}
	return r.Get(n - 1).Term
}
