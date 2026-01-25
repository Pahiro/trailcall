package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"trailcall/db"
	"trailcall/models"
)

func HandleCheckins(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/checkins")

	if path == "/bulk" || path == "/bulk/" {
		handleBulkCheckin(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	createCheckin(w, r)
}

func createCheckin(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCheckinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.HikeID == 0 || req.MembershipNumber == "" {
		http.Error(w, "hike_id and membership_number are required", http.StatusBadRequest)
		return
	}

	checkin, err := db.CreateCheckin(req.HikeID, req.MembershipNumber)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			http.Error(w, "Member not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(checkin)
}

func handleBulkCheckin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.BulkCheckinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var results []models.Checkin
	var errors []string

	for _, c := range req.Checkins {
		checkin, err := db.CreateCheckin(c.HikeID, c.MembershipNumber)
		if err != nil {
			errors = append(errors, c.MembershipNumber+": "+err.Error())
			continue
		}
		results = append(results, *checkin)
	}

	response := map[string]interface{}{
		"checkins": results,
		"errors":   errors,
	}

	w.Header().Set("Content-Type", "application/json")
	if len(errors) > 0 && len(results) == 0 {
		w.WriteHeader(http.StatusBadRequest)
	} else if len(errors) > 0 {
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(response)
}
