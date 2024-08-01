package controller

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/din-mukhammed/simple-raft/internal/entities"
	"github.com/din-mukhammed/simple-raft/internal/raft"
)

type httpController struct {
	rt RaftService
}

func NewHTTPController(raft RaftService) *httpController {
	hc := &httpController{
		rt: raft,
	}

	return hc
}

func (hc *httpController) AddRoutes(mux *http.ServeMux) {
	mux.HandleFunc("PUT /append", func(w http.ResponseWriter, r *http.Request) {
		var aeReq entities.AppendEntriesRequest

		bb, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := json.Unmarshal(bb, &aeReq); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp, err := hc.rt.OnAppendEntries(aeReq)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		bb, err = json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(bb)
	})

	mux.HandleFunc("POST /broadcast", func(w http.ResponseWriter, r *http.Request) {
		bb, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var msg entities.BroadcastMsg
		if err := json.Unmarshal(bb, &msg); err != nil {
			slog.Error("unmarshal msg", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if redirectUrl, err := hc.rt.BroadcastMsg(msg); err != nil {
			if errors.Is(err, raft.ErrRedirectToLeader) {
				http.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
				return
			}
			slog.Error("broacast msg", "err", err, "msg", msg)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("PUT /vote", func(w http.ResponseWriter, r *http.Request) {
		var voteReq entities.VoteRequest

		bb, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := json.Unmarshal(bb, &voteReq); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp, err := hc.rt.OnRequestVote(voteReq)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		bb, err = json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(bb)
	})
}
