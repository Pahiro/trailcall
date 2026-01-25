package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trailcall/db"
)

func HandleReports(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/reports/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Invalid report path", http.StatusBadRequest)
		return
	}

	switch parts[0] {
	case "member":
		handleMemberReport(w, r, parts[1:])
	case "hike":
		handleHikeReport(w, r, parts[1:])
	case "hikes":
		handleAllHikesReport(w, r)
	case "attendance":
		handleFullAttendanceReport(w, r)
	default:
		http.Error(w, "Unknown report type", http.StatusNotFound)
	}
}

func handleMemberReport(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		http.Error(w, "Member ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	// Check if CSV export requested
	if len(parts) == 2 && parts[1] == "csv" {
		exportMemberHistoryCSV(w, r, id)
		return
	}

	// Return JSON
	member, err := db.GetMemberByID(id)
	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	hikes, err := db.GetMemberAttendanceHistory(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"member": member,
		"hikes":  hikes,
		"total":  len(hikes),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func exportMemberHistoryCSV(w http.ResponseWriter, r *http.Request, memberID int64) {
	member, err := db.GetMemberByID(memberID)
	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	hikes, err := db.GetMemberAttendanceHistory(memberID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("%s_%s_attendance.csv", member.FirstName, member.LastName)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header
	writer.Write([]string{"Date", "Hike Name", "Location", "Status"})

	// Data
	for _, h := range hikes {
		writer.Write([]string{h.Date, h.Name, h.Location, h.Status})
	}
}

func handleHikeReport(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		http.Error(w, "Hike ID required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid hike ID", http.StatusBadRequest)
		return
	}

	// Check if CSV export requested
	if len(parts) == 2 && parts[1] == "csv" {
		exportHikeAttendanceCSV(w, r, id)
		return
	}

	http.Error(w, "Invalid report path", http.StatusBadRequest)
}

func exportHikeAttendanceCSV(w http.ResponseWriter, r *http.Request, hikeID int64) {
	hike, err := db.GetHikeByID(hikeID)
	if err != nil {
		http.Error(w, "Hike not found", http.StatusNotFound)
		return
	}

	checkins, err := db.GetCheckinsForHike(hikeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get checked-in guests
	rsvps, err := db.GetRSVPsForHike(hikeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("%s_%s_attendance.csv", hike.Date, strings.ReplaceAll(hike.Name, " ", "_"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header
	writer.Write([]string{"Membership Number", "First Name", "Last Name", "Type", "Activities"})

	// Data - members
	for _, c := range checkins {
		activities, _ := db.GetActivitiesForCheckin(c.ID)
		activityStr := strings.Join(activities, ", ")
		// Parse member name
		nameParts := strings.SplitN(c.MemberName, " ", 2)
		firstName := ""
		lastName := ""
		if len(nameParts) >= 1 {
			firstName = nameParts[0]
		}
		if len(nameParts) >= 2 {
			lastName = nameParts[1]
		}

		typeStr := "Member"
		if c.IsLeader {
			typeStr += " (Leader)"
		}
		if c.IsSweeper {
			typeStr += " (Sweeper)"
		}

		writer.Write([]string{c.MembershipNumber, firstName, lastName, typeStr, activityStr})
	}

	// Data - guests
	for _, r := range rsvps {
		if r.CheckedIn && r.MemberID == nil {
			activities, _ := db.GetActivitiesForRSVP(r.ID)
			activityStr := strings.Join(activities, ", ")
			writer.Write([]string{"", r.GuestName, "", "Guest", activityStr})
		}
	}
}

func handleAllHikesReport(w http.ResponseWriter, r *http.Request) {
	// Check if CSV export requested
	if strings.HasSuffix(r.URL.Path, "/csv") {
		exportAllHikesCSV(w, r)
		return
	}
	http.Error(w, "Invalid report path", http.StatusBadRequest)
}

func exportAllHikesCSV(w http.ResponseWriter, r *http.Request) {
	hikes, err := db.GetAllHikes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"all_hikes.csv\"")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header
	writer.Write([]string{"Date", "Name", "Location", "Status", "Attendees"})

	// Data
	for _, h := range hikes {
		writer.Write([]string{h.Date, h.Name, h.Location, h.Status, strconv.Itoa(h.AttendeeCount)})
	}
}

func handleFullAttendanceReport(w http.ResponseWriter, r *http.Request) {
	// Get year from query param, default to current year
	year := r.URL.Query().Get("year")
	if year == "" {
		year = fmt.Sprintf("%d", time.Now().Year())
	}

	records, err := db.GetAllAttendanceForYear(year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("attendance_%s.csv", year)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header
	writer.Write([]string{"Date", "Hike Name", "Location", "Membership Number", "First Name", "Last Name", "Check-in Time", "Activities"})

	// Data
	for _, rec := range records {
		activities, _ := db.GetActivitiesForCheckin(rec.CheckinID)
		activityStr := strings.Join(activities, ", ")
		writer.Write([]string{
			rec.HikeDate,
			rec.HikeName,
			rec.HikeLocation,
			rec.MembershipNumber,
			rec.FirstName,
			rec.LastName,
			rec.CheckedInAt.Format("15:04:05"),
			activityStr,
		})
	}
}
