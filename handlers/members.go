package handlers

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"

	"trailcall/db"
	"trailcall/models"
)

func HandleMembers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listMembers(w, r)
	case http.MethodPost:
		createMember(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleMember(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/members/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/members/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		http.Error(w, "Member ID required", http.StatusBadRequest)
		return
	}

	// Check if it's a QR request: /api/members/{id}/qr
	if len(parts) == 2 && parts[1] == "qr" {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			http.Error(w, "Invalid member ID", http.StatusBadRequest)
			return
		}
		generateQR(w, r, id)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getMember(w, r, id)
	case http.MethodPut:
		updateMember(w, r, id)
	case http.MethodDelete:
		deleteMember(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listMembers(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") != "false"
	members, err := db.GetAllMembers(activeOnly)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if members == nil {
		members = []models.Member{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

func getMember(w http.ResponseWriter, r *http.Request, id int64) {
	member, err := db.GetMemberByID(id)
	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(member)
}

func createMember(w http.ResponseWriter, r *http.Request) {
	var req models.CreateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.MembershipNumber == "" || req.FirstName == "" || req.LastName == "" {
		http.Error(w, "membership_number, first_name, and last_name are required", http.StatusBadRequest)
		return
	}

	member, err := db.CreateMember(req)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			http.Error(w, "Membership number already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(member)
}

func updateMember(w http.ResponseWriter, r *http.Request, id int64) {
	var req models.UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	member, err := db.UpdateMember(id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(member)
}

func deleteMember(w http.ResponseWriter, r *http.Request, id int64) {
	if err := db.DeleteMember(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func generateQR(w http.ResponseWriter, r *http.Request, id int64) {
	member, err := db.GetMemberByID(id)
	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	// Generate QR code with membership number (e.g., "TC-001")
	png, err := qrcode.Encode(member.MembershipNumber, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "inline; filename=\""+member.MembershipNumber+".png\"")
	w.Write(png)
}

func HandleMembersImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header row
	header, err := reader.Read()
	if err != nil {
		http.Error(w, "Failed to read CSV header", http.StatusBadRequest)
		return
	}

	// Map column names to indices
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Required columns
	numCol, hasNum := colMap["membership_number"]
	if !hasNum {
		numCol, hasNum = colMap["membership number"]
	}
	if !hasNum {
		numCol, hasNum = colMap["number"]
	}

	firstCol, hasFirst := colMap["first_name"]
	if !hasFirst {
		firstCol, hasFirst = colMap["first name"]
	}
	if !hasFirst {
		firstCol, hasFirst = colMap["firstname"]
	}

	lastCol, hasLast := colMap["last_name"]
	if !hasLast {
		lastCol, hasLast = colMap["last name"]
	}
	if !hasLast {
		lastCol, hasLast = colMap["lastname"]
	}
	if !hasLast {
		lastCol, hasLast = colMap["surname"]
	}

	if !hasNum || !hasFirst || !hasLast {
		http.Error(w, "CSV must have columns: membership_number, first_name, last_name", http.StatusBadRequest)
		return
	}

	// Optional columns
	emailCol, hasEmail := colMap["email"]
	phoneCol, hasPhone := colMap["phone"]

	var imported, skipped, errors int
	var errorMsgs []string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors++
			continue
		}

		memberNum := strings.TrimSpace(record[numCol])
		firstName := strings.TrimSpace(record[firstCol])
		lastName := strings.TrimSpace(record[lastCol])

		if memberNum == "" || firstName == "" || lastName == "" {
			skipped++
			continue
		}

		// Check for duplicate
		existing, _ := db.GetMemberByMembershipNumber(memberNum)
		if existing != nil {
			skipped++
			continue
		}

		req := models.CreateMemberRequest{
			MembershipNumber: memberNum,
			FirstName:        firstName,
			LastName:         lastName,
		}

		if hasEmail && emailCol < len(record) {
			req.Email = strings.TrimSpace(record[emailCol])
		}
		if hasPhone && phoneCol < len(record) {
			req.Phone = strings.TrimSpace(record[phoneCol])
		}

		_, err = db.CreateMember(req)
		if err != nil {
			errors++
			errorMsgs = append(errorMsgs, memberNum+": "+err.Error())
		} else {
			imported++
		}
	}

	response := map[string]interface{}{
		"imported": imported,
		"skipped":  skipped,
		"errors":   errors,
	}
	if len(errorMsgs) > 0 {
		response["error_messages"] = errorMsgs
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
