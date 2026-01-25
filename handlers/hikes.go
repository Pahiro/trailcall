package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"trailcall/db"
	"trailcall/models"
)

func HandleHikes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listHikes(w, r)
	case http.MethodPost:
		createHike(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleHike(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/hikes/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/hikes/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		http.Error(w, "Hike ID required", http.StatusBadRequest)
		return
	}

	// Handle special paths
	if parts[0] == "open" {
		getOpenHike(w, r)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid hike ID", http.StatusBadRequest)
		return
	}

	// Check for sub-routes: /api/hikes/{id}/close, /api/hikes/{id}/checkins, /api/hikes/{id}/rsvps, /api/hikes/{id}/activities
	if len(parts) >= 2 {
		switch parts[1] {
		case "close":
			closeHike(w, r, id)
			return
		case "checkins":
			getHikeCheckins(w, r, id)
			return
		case "rsvps":
			if len(parts) == 3 {
				switch parts[2] {
				case "close":
					HandleCloseRSVPs(w, r, id)
					return
				case "open":
					HandleOpenRSVPs(w, r, id)
					return
				}
			}
			HandleHikeRSVPs(w, r, id)
			return
		case "activities":
			HandleActivities(w, r, id)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		getHike(w, r, id)
	case http.MethodPut:
		updateHike(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listHikes(w http.ResponseWriter, r *http.Request) {
	hikes, err := db.GetAllHikes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if hikes == nil {
		hikes = []models.Hike{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hikes)
}

func getHike(w http.ResponseWriter, r *http.Request, id int64) {
	hike, err := db.GetHikeByID(id)
	if err != nil {
		http.Error(w, "Hike not found", http.StatusNotFound)
		return
	}

	attendees, err := db.GetAttendeesForHike(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if attendees == nil {
		attendees = []models.Member{}
	}

	detail := models.HikeDetail{
		Hike:      *hike,
		Attendees: attendees,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func getOpenHike(w http.ResponseWriter, r *http.Request) {
	hike, err := db.GetOpenHike()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hike)
}

func createHike(w http.ResponseWriter, r *http.Request) {
	var req models.CreateHikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Date == "" {
		http.Error(w, "name and date are required", http.StatusBadRequest)
		return
	}

	hike, err := db.CreateHike(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(hike)
}

func updateHike(w http.ResponseWriter, r *http.Request, id int64) {
	var req models.UpdateHikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	hike, err := db.UpdateHike(id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hike)
}

func closeHike(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hike, err := db.CloseHike(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hike)
}

func getHikeCheckins(w http.ResponseWriter, r *http.Request, id int64) {
	checkins, err := db.GetCheckinsForHike(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if checkins == nil {
		checkins = []models.Checkin{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checkins)
}
