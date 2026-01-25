package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"trailcall/db"
	"trailcall/models"
)

// HandleActivities handles /api/hikes/{id}/activities
func HandleActivities(w http.ResponseWriter, r *http.Request, hikeID int64) {
	switch r.Method {
	case http.MethodGet:
		listActivities(w, r, hikeID)
	case http.MethodPost:
		createActivity(w, r, hikeID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listActivities(w http.ResponseWriter, r *http.Request, hikeID int64) {
	activities, err := db.GetActivitiesForHike(hikeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if activities == nil {
		activities = []models.Activity{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activities)
}

func createActivity(w http.ResponseWriter, r *http.Request, hikeID int64) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Activity name is required", http.StatusBadRequest)
		return
	}

	activity, err := db.CreateActivity(hikeID, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(activity)
}

// HandleActivity handles /api/activities/{id}
func HandleActivity(w http.ResponseWriter, r *http.Request) {
	// Extract activity ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/activities/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		http.Error(w, "Activity ID required", http.StatusBadRequest)
		return
	}

	activityID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid activity ID", http.StatusBadRequest)
		return
	}

	// Check for sub-routes
	if len(parts) >= 2 {
		switch parts[1] {
		case "participants":
			handleActivityParticipants(w, r, activityID)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		getActivity(w, r, activityID)
	case http.MethodDelete:
		deleteActivity(w, r, activityID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getActivity(w http.ResponseWriter, r *http.Request, activityID int64) {
	activity, err := db.GetActivityByID(activityID)
	if err != nil {
		http.Error(w, "Activity not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activity)
}

func deleteActivity(w http.ResponseWriter, r *http.Request, activityID int64) {
	if err := db.DeleteActivity(activityID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleActivityParticipants(w http.ResponseWriter, r *http.Request, activityID int64) {
	switch r.Method {
	case http.MethodGet:
		participants, err := db.GetActivityParticipants(activityID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if participants == nil {
			participants = []models.ActivityParticipant{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(participants)

	case http.MethodPost:
		var req struct {
			CheckinID *int64 `json:"checkin_id"`
			RSVPID    *int64 `json:"rsvp_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.CheckinID == nil && req.RSVPID == nil {
			http.Error(w, "checkin_id or rsvp_id required", http.StatusBadRequest)
			return
		}

		if err := db.AddActivityParticipant(activityID, req.CheckinID, req.RSVPID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	case http.MethodDelete:
		var req struct {
			CheckinID *int64 `json:"checkin_id"`
			RSVPID    *int64 `json:"rsvp_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := db.RemoveActivityParticipant(activityID, req.CheckinID, req.RSVPID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
