// space bounty hunters handlers: /api/criminals and /api/hunts
package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"problemlog/internal/store"
)

func (s *Server) listCriminals(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.CriminalFilter{Sort: q.Get("sort")}
	if f.Sort != "" && f.Sort != "reward" && f.Sort != "-reward" {
		writeError(w, http.StatusBadRequest, "sort must be reward or -reward")
		return
	}

	var ok bool
	if f.MinReward, ok = intParam(w, q.Get("min_reward"), "min_reward"); !ok {
		return
	}
	if f.MaxReward, ok = intParam(w, q.Get("max_reward"), "max_reward"); !ok {
		return
	}
	if f.Armed, ok = boolParam(w, q.Get("armed"), "armed"); !ok {
		return
	}
	if f.HasCrew, ok = boolParam(w, q.Get("has_crew"), "has_crew"); !ok {
		return
	}
	if f.EscapeMaster, ok = boolParam(w, q.Get("escape_master"), "escape_master"); !ok {
		return
	}
	if f.MinReward != nil && f.MaxReward != nil && *f.MinReward > *f.MaxReward {
		writeError(w, http.StatusBadRequest, "min_reward must not be bigger than max_reward")
		return
	}

	list, err := s.store.ListCriminals(r.Context(), f)
	if err != nil {
		writeBountyError(w, err, "criminal")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getCriminal(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	c, err := s.store.GetCriminal(r.Context(), id)
	if err != nil {
		writeBountyError(w, err, "criminal")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) listHunts(w http.ResponseWriter, r *http.Request) {
	var criminalID *int64
	if raw := r.URL.Query().Get("criminal_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "criminal_id must be a positive integer")
			return
		}
		criminalID = &id
	}
	list, err := s.store.ListHunts(r.Context(), criminalID)
	if err != nil {
		writeBountyError(w, err, "hunt log")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) getHunt(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	h, err := s.store.GetHunt(r.Context(), id)
	if err != nil {
		writeBountyError(w, err, "hunt log")
		return
	}
	writeJSON(w, http.StatusOK, h)
}

// no created_at field here, so sending it gives 400 (unknown field)
type createHuntRequest struct {
	CriminalID *int64 `json:"criminal_id"`
	Success    *bool  `json:"success"`
	Notes      string `json:"notes"`
}

func (s *Server) createHunt(w http.ResponseWriter, r *http.Request) {
	var req createHuntRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	notes := strings.TrimSpace(req.Notes)
	switch {
	case req.CriminalID == nil || *req.CriminalID <= 0:
		writeError(w, http.StatusBadRequest, "criminal_id is required and must be a positive integer")
		return
	case req.Success == nil:
		writeError(w, http.StatusBadRequest, "success is required (true or false)")
		return
	case len(notes) > 2000:
		writeError(w, http.StatusBadRequest, "notes must be at most 2000 characters")
		return
	}

	h, err := s.store.CreateHunt(r.Context(), *req.CriminalID, *req.Success, notes)
	if err != nil {
		writeBountyError(w, err, "hunt log")
		return
	}
	writeJSON(w, http.StatusCreated, h)
}

// only success and notes can change
type patchHuntRequest struct {
	Success *bool   `json:"success"`
	Notes   *string `json:"notes"`
}

func (s *Server) patchHunt(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req patchHuntRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Success == nil && req.Notes == nil {
		writeError(w, http.StatusBadRequest, "send success and/or notes")
		return
	}
	if req.Notes != nil {
		n := strings.TrimSpace(*req.Notes)
		if len(n) > 2000 {
			writeError(w, http.StatusBadRequest, "notes must be at most 2000 characters")
			return
		}
		req.Notes = &n
	}

	h, err := s.store.UpdateHunt(r.Context(), id, req.Success, req.Notes)
	if err != nil {
		writeBountyError(w, err, "hunt log")
		return
	}
	writeJSON(w, http.StatusOK, h)
}

func (s *Server) deleteHunt(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteHunt(r.Context(), id); err != nil {
		writeBountyError(w, err, "hunt log")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func boolParam(w http.ResponseWriter, raw, name string) (*bool, bool) {
	if raw == "" {
		return nil, true
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, name+" must be true or false")
		return nil, false
	}
	return &b, true
}

func writeBountyError(w http.ResponseWriter, err error, what string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, what+" not found")
	case errors.Is(err, store.ErrBadReference):
		writeError(w, http.StatusBadRequest, "criminal with this criminal_id does not exist")
	default:
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
