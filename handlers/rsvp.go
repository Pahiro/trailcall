package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"trailcall/db"
	"trailcall/models"
)

// HandleRSVP handles public RSVP submissions (no auth required)
func HandleRSVP(w http.ResponseWriter, r *http.Request) {
	// Extract hike ID from path: /rsvp/{id}
	path := strings.TrimPrefix(r.URL.Path, "/rsvp/")
	idStr := strings.Split(path, "/")[0]
	hikeID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		// Serve the RSVP HTML page for invalid/missing ID (let JS handle error)
		serveRSVPPage(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Check Accept header - serve HTML for browsers, JSON for API
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "text/html") {
			serveRSVPPage(w, r)
			return
		}
		getRSVPPage(w, r, hikeID)
	case http.MethodPost:
		submitRSVP(w, r, hikeID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// serveRSVPPage serves the RSVP HTML with no-cache headers
func serveRSVPPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	http.ServeFile(w, r, "frontend/rsvp.html")
}

func getRSVPPage(w http.ResponseWriter, r *http.Request, hikeID int64) {
	hike, err := db.GetHikeByID(hikeID)
	if err != nil {
		http.Error(w, "Hike not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"hike":      hike,
		"rsvp_open": hike.RSVPOpen,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	json.NewEncoder(w).Encode(response)
}

func submitRSVP(w http.ResponseWriter, r *http.Request, hikeID int64) {
	// Check if RSVPs are open
	open, err := db.IsRSVPOpen(hikeID)
	if err != nil {
		http.Error(w, "Hike not found", http.StatusNotFound)
		return
	}
	if !open {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.RSVPResponse{
			Success: false,
			Message: "RSVPs are closed for this hike",
		})
		return
	}

	var req models.RSVPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.FirstName == "" || req.LastName == "" {
		http.Error(w, "First name and last name required", http.StatusBadRequest)
		return
	}

	// Try to match to existing member
	member, score := db.FuzzyMatchMember(req.FirstName, req.LastName)

	// High confidence match (>= 80%)
	if member != nil && score >= 80 {
		err := db.CreateRSVPForMember(hikeID, member.ID)
		if err != nil {
			// Likely duplicate
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(models.RSVPResponse{
				Success:      false,
				Message:      "You're already registered for this hike",
				MatchedName:  member.FirstName + " " + member.LastName,
				MemberNumber: member.MembershipNumber,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.RSVPResponse{
			Success:      true,
			Message:      "RSVP confirmed!",
			MatchedName:  member.FirstName + " " + member.LastName,
			MemberNumber: member.MembershipNumber,
			IsGuest:      false,
		})
		return
	}

	// No match or low confidence - register as guest
	guestName := strings.TrimSpace(req.FirstName) + " " + strings.TrimSpace(req.LastName)
	err = db.CreateRSVPForGuest(hikeID, guestName)
	if err != nil {
		// Likely duplicate
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.RSVPResponse{
			Success: false,
			Message: "You're already registered for this hike",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.RSVPResponse{
		Success:     true,
		Message:     "RSVP confirmed as guest",
		MatchedName: guestName,
		IsGuest:     true,
	})
}

// HandleHikeRSVPs handles admin viewing of RSVPs (auth required)
func HandleHikeRSVPs(w http.ResponseWriter, r *http.Request, hikeID int64) {
	switch r.Method {
	case http.MethodGet:
		rsvps, err := db.GetRSVPsForHike(hikeID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rsvps)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleCloseRSVPs closes RSVPs for a hike
func HandleCloseRSVPs(w http.ResponseWriter, r *http.Request, hikeID int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := db.CloseRSVPs(hikeID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleOpenRSVPs reopens RSVPs for a hike
func HandleOpenRSVPs(w http.ResponseWriter, r *http.Request, hikeID int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := db.OpenRSVPs(hikeID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleRSVPCheckin checks in an RSVP (member or guest)
func HandleRSVPCheckin(w http.ResponseWriter, r *http.Request, rsvpID int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := db.CheckInRSVP(rsvpID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleRSVPUndoCheckin undoes a check-in for an RSVP (member or guest)
func HandleRSVPUndoCheckin(w http.ResponseWriter, r *http.Request, rsvpID int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := db.UndoRSVPCheckin(rsvpID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
