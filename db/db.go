package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"trailcall/models"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	// Enable foreign keys
	_, err = DB.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	return migrate()
}

func migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS members (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		membership_number TEXT UNIQUE NOT NULL,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		email TEXT,
		phone TEXT,
		active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS hikes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		date TEXT NOT NULL,
		location TEXT,
		notes TEXT,
		status TEXT DEFAULT 'open',
		rsvp_open INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS checkins (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hike_id INTEGER NOT NULL,
		member_id INTEGER NOT NULL,
		checked_in_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		synced INTEGER DEFAULT 1,
		FOREIGN KEY (hike_id) REFERENCES hikes(id),
		FOREIGN KEY (member_id) REFERENCES members(id),
		UNIQUE(hike_id, member_id)
	);

	CREATE TABLE IF NOT EXISTS rsvps (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hike_id INTEGER NOT NULL,
		member_id INTEGER,
		guest_name TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (hike_id) REFERENCES hikes(id),
		FOREIGN KEY (member_id) REFERENCES members(id),
		UNIQUE(hike_id, member_id),
		UNIQUE(hike_id, guest_name)
	);

	CREATE TABLE IF NOT EXISTS activities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hike_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (hike_id) REFERENCES hikes(id)
	);

	CREATE TABLE IF NOT EXISTS activity_participants (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		activity_id INTEGER NOT NULL,
		checkin_id INTEGER,
		rsvp_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (activity_id) REFERENCES activities(id) ON DELETE CASCADE,
		FOREIGN KEY (checkin_id) REFERENCES checkins(id) ON DELETE CASCADE,
		FOREIGN KEY (rsvp_id) REFERENCES rsvps(id) ON DELETE CASCADE,
		UNIQUE(activity_id, checkin_id),
		UNIQUE(activity_id, rsvp_id)
	);

	CREATE INDEX IF NOT EXISTS idx_members_membership_number ON members(membership_number);
	CREATE INDEX IF NOT EXISTS idx_checkins_hike_id ON checkins(hike_id);
	CREATE INDEX IF NOT EXISTS idx_checkins_member_id ON checkins(member_id);
	CREATE INDEX IF NOT EXISTS idx_rsvps_hike_id ON rsvps(hike_id);
	CREATE INDEX IF NOT EXISTS idx_activities_hike_id ON activities(hike_id);
	CREATE INDEX IF NOT EXISTS idx_activity_participants_activity_id ON activity_participants(activity_id);
	`
	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	// Migrations for existing databases
	// Add rsvp_open column to hikes table (ignore error if already exists)
	DB.Exec("ALTER TABLE hikes ADD COLUMN rsvp_open INTEGER DEFAULT 1")
	// Add checked_in_at column to rsvps table for guest check-ins
	DB.Exec("ALTER TABLE rsvps ADD COLUMN checked_in_at DATETIME")
	// Add is_leader and is_sweeper columns to checkins table
	DB.Exec("ALTER TABLE checkins ADD COLUMN is_leader INTEGER DEFAULT 0")
	DB.Exec("ALTER TABLE checkins ADD COLUMN is_sweeper INTEGER DEFAULT 0")

	return nil
}

// Member operations

func GetAllMembers(activeOnly bool) ([]models.Member, error) {
	query := "SELECT id, membership_number, first_name, last_name, email, phone, active, created_at, updated_at FROM members"
	if activeOnly {
		query += " WHERE active = 1"
	}
	query += " ORDER BY last_name, first_name"

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.Member
	for rows.Next() {
		var m models.Member
		var email, phone sql.NullString
		err := rows.Scan(&m.ID, &m.MembershipNumber, &m.FirstName, &m.LastName, &email, &phone, &m.Active, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, err
		}
		m.Email = email.String
		m.Phone = phone.String
		members = append(members, m)
	}
	return members, nil
}

func GetMemberByID(id int64) (*models.Member, error) {
	var m models.Member
	var email, phone sql.NullString
	err := DB.QueryRow(
		"SELECT id, membership_number, first_name, last_name, email, phone, active, created_at, updated_at FROM members WHERE id = ?",
		id,
	).Scan(&m.ID, &m.MembershipNumber, &m.FirstName, &m.LastName, &email, &phone, &m.Active, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	m.Email = email.String
	m.Phone = phone.String
	return &m, nil
}

func GetMemberByMembershipNumber(num string) (*models.Member, error) {
	var m models.Member
	var email, phone sql.NullString
	err := DB.QueryRow(
		"SELECT id, membership_number, first_name, last_name, email, phone, active, created_at, updated_at FROM members WHERE membership_number = ?",
		num,
	).Scan(&m.ID, &m.MembershipNumber, &m.FirstName, &m.LastName, &email, &phone, &m.Active, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	m.Email = email.String
	m.Phone = phone.String
	return &m, nil
}

func CreateMember(req models.CreateMemberRequest) (*models.Member, error) {
	result, err := DB.Exec(
		"INSERT INTO members (membership_number, first_name, last_name, email, phone) VALUES (?, ?, ?, ?, ?)",
		req.MembershipNumber, req.FirstName, req.LastName, req.Email, req.Phone,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return GetMemberByID(id)
}

func UpdateMember(id int64, req models.UpdateMemberRequest) (*models.Member, error) {
	m, err := GetMemberByID(id)
	if err != nil {
		return nil, err
	}

	if req.MembershipNumber != "" {
		m.MembershipNumber = req.MembershipNumber
	}
	if req.FirstName != "" {
		m.FirstName = req.FirstName
	}
	if req.LastName != "" {
		m.LastName = req.LastName
	}
	if req.Email != "" {
		m.Email = req.Email
	}
	if req.Phone != "" {
		m.Phone = req.Phone
	}
	if req.Active != nil {
		m.Active = *req.Active
	}

	_, err = DB.Exec(
		"UPDATE members SET membership_number = ?, first_name = ?, last_name = ?, email = ?, phone = ?, active = ?, updated_at = ? WHERE id = ?",
		m.MembershipNumber, m.FirstName, m.LastName, m.Email, m.Phone, m.Active, time.Now(), id,
	)
	if err != nil {
		return nil, err
	}
	return GetMemberByID(id)
}

func DeleteMember(id int64) error {
	_, err := DB.Exec("UPDATE members SET active = 0, updated_at = ? WHERE id = ?", time.Now(), id)
	return err
}

// Hike operations

func GetAllHikes() ([]models.Hike, error) {
	rows, err := DB.Query(`
		SELECT h.id, h.name, h.date, h.location, h.notes, h.status, h.rsvp_open, h.created_at,
		       (SELECT COUNT(*) FROM checkins WHERE hike_id = h.id) as attendee_count,
		       (SELECT COUNT(*) FROM rsvps WHERE hike_id = h.id) as rsvp_count
		FROM hikes h
		ORDER BY h.date DESC, h.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hikes []models.Hike
	for rows.Next() {
		var h models.Hike
		var location, notes sql.NullString
		err := rows.Scan(&h.ID, &h.Name, &h.Date, &location, &notes, &h.Status, &h.RSVPOpen, &h.CreatedAt, &h.AttendeeCount, &h.RSVPCount)
		if err != nil {
			return nil, err
		}
		h.Location = location.String
		h.Notes = notes.String
		hikes = append(hikes, h)
	}
	return hikes, nil
}

func GetHikeByID(id int64) (*models.Hike, error) {
	var h models.Hike
	var location, notes sql.NullString
	err := DB.QueryRow(`
		SELECT h.id, h.name, h.date, h.location, h.notes, h.status, h.rsvp_open, h.created_at,
		       (SELECT COUNT(*) FROM checkins WHERE hike_id = h.id) as attendee_count,
		       (SELECT COUNT(*) FROM rsvps WHERE hike_id = h.id) as rsvp_count
		FROM hikes h
		WHERE h.id = ?
	`, id).Scan(&h.ID, &h.Name, &h.Date, &location, &notes, &h.Status, &h.RSVPOpen, &h.CreatedAt, &h.AttendeeCount, &h.RSVPCount)
	if err != nil {
		return nil, err
	}
	h.Location = location.String
	h.Notes = notes.String
	return &h, nil
}

func GetOpenHike() (*models.Hike, error) {
	var h models.Hike
	var location, notes sql.NullString
	err := DB.QueryRow(`
		SELECT h.id, h.name, h.date, h.location, h.notes, h.status, h.rsvp_open, h.created_at,
		       (SELECT COUNT(*) FROM checkins WHERE hike_id = h.id) as attendee_count,
		       (SELECT COUNT(*) FROM rsvps WHERE hike_id = h.id) as rsvp_count
		FROM hikes h
		WHERE h.status = 'open'
		ORDER BY h.created_at DESC
		LIMIT 1
	`).Scan(&h.ID, &h.Name, &h.Date, &location, &notes, &h.Status, &h.RSVPOpen, &h.CreatedAt, &h.AttendeeCount, &h.RSVPCount)
	if err != nil {
		return nil, err
	}
	h.Location = location.String
	h.Notes = notes.String
	return &h, nil
}

func CreateHike(req models.CreateHikeRequest) (*models.Hike, error) {
	result, err := DB.Exec(
		"INSERT INTO hikes (name, date, location, notes) VALUES (?, ?, ?, ?)",
		req.Name, req.Date, req.Location, req.Notes,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return GetHikeByID(id)
}

func UpdateHike(id int64, req models.UpdateHikeRequest) (*models.Hike, error) {
	h, err := GetHikeByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		h.Name = req.Name
	}
	if req.Date != "" {
		h.Date = req.Date
	}
	if req.Location != "" {
		h.Location = req.Location
	}
	if req.Notes != "" {
		h.Notes = req.Notes
	}

	_, err = DB.Exec(
		"UPDATE hikes SET name = ?, date = ?, location = ?, notes = ? WHERE id = ?",
		h.Name, h.Date, h.Location, h.Notes, id,
	)
	if err != nil {
		return nil, err
	}
	return GetHikeByID(id)
}

func CloseHike(id int64) (*models.Hike, error) {
	_, err := DB.Exec("UPDATE hikes SET status = 'closed' WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return GetHikeByID(id)
}

// Checkin operations

func CreateCheckin(hikeID int64, membershipNumber string) (*models.Checkin, error) {
	member, err := GetMemberByMembershipNumber(membershipNumber)
	if err != nil {
		return nil, err
	}

	result, err := DB.Exec(
		"INSERT OR IGNORE INTO checkins (hike_id, member_id) VALUES (?, ?)",
		hikeID, member.ID,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	if id == 0 {
		// Already checked in, get existing
		var c models.Checkin
		err := DB.QueryRow(
			"SELECT id, hike_id, member_id, checked_in_at, synced FROM checkins WHERE hike_id = ? AND member_id = ?",
			hikeID, member.ID,
		).Scan(&c.ID, &c.HikeID, &c.MemberID, &c.CheckedInAt, &c.Synced)
		if err != nil {
			return nil, err
		}
		c.MemberName = member.FirstName + " " + member.LastName
		c.MembershipNumber = member.MembershipNumber
		return &c, nil
	}

	var c models.Checkin
	err = DB.QueryRow(
		"SELECT id, hike_id, member_id, checked_in_at, synced FROM checkins WHERE id = ?",
		id,
	).Scan(&c.ID, &c.HikeID, &c.MemberID, &c.CheckedInAt, &c.Synced)
	if err != nil {
		return nil, err
	}
	c.MemberName = member.FirstName + " " + member.LastName
	c.MembershipNumber = member.MembershipNumber
	return &c, nil
}

// DeleteCheckin removes a check-in for a member from a hike
func DeleteCheckin(hikeID, memberID int64) error {
	_, err := DB.Exec("DELETE FROM checkins WHERE hike_id = ? AND member_id = ?", hikeID, memberID)
	return err
}

// UndoRSVPCheckin removes a check-in - for members deletes checkin, for guests clears checked_in_at
func UndoRSVPCheckin(rsvpID int64) error {
	// Get the RSVP details
	var memberID sql.NullInt64
	var hikeID int64
	err := DB.QueryRow("SELECT hike_id, member_id FROM rsvps WHERE id = ?", rsvpID).Scan(&hikeID, &memberID)
	if err != nil {
		return err
	}

	if memberID.Valid {
		// Member RSVP - delete the checkin record
		return DeleteCheckin(hikeID, memberID.Int64)
	} else {
		// Guest RSVP - clear the checked_in_at field
		_, err = DB.Exec("UPDATE rsvps SET checked_in_at = NULL WHERE id = ?", rsvpID)
		return err
	}
}

func GetCheckinsForHike(hikeID int64) ([]models.Checkin, error) {
	rows, err := DB.Query(`
		SELECT c.id, c.hike_id, c.member_id, c.checked_in_at, c.synced,
		       c.is_leader, c.is_sweeper,
		       m.first_name || ' ' || m.last_name as member_name, m.membership_number
		FROM checkins c
		JOIN members m ON c.member_id = m.id
		WHERE c.hike_id = ?
		ORDER BY c.checked_in_at DESC
	`, hikeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checkins []models.Checkin
	for rows.Next() {
		var c models.Checkin
		err := rows.Scan(&c.ID, &c.HikeID, &c.MemberID, &c.CheckedInAt, &c.Synced, &c.IsLeader, &c.IsSweeper, &c.MemberName, &c.MembershipNumber)
		if err != nil {
			return nil, fmt.Errorf("scan error at row: %w", err)
		}
		checkins = append(checkins, c)
	}
	return checkins, nil
}

// UpdateCheckinRole toggles a role for a check-in
func UpdateCheckinRole(checkinID int64, role string, value bool) error {
	column := ""
	if role == "leader" {
		column = "is_leader"
	} else if role == "sweeper" {
		column = "is_sweeper"
	} else {
		return fmt.Errorf("invalid role: %s", role)
	}

	val := 0
	if value {
		val = 1
	}

	query := fmt.Sprintf("UPDATE checkins SET %s = ? WHERE id = ?", column)
	_, err := DB.Exec(query, val, checkinID)
	return err
}

func GetAttendeesForHike(hikeID int64) ([]models.Member, error) {
	rows, err := DB.Query(`
		SELECT m.id, m.membership_number, m.first_name, m.last_name, m.email, m.phone, m.active, m.created_at, m.updated_at
		FROM members m
		JOIN checkins c ON m.id = c.member_id
		WHERE c.hike_id = ?
		ORDER BY c.checked_in_at DESC
	`, hikeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.Member
	for rows.Next() {
		var m models.Member
		var email, phone sql.NullString
		err := rows.Scan(&m.ID, &m.MembershipNumber, &m.FirstName, &m.LastName, &email, &phone, &m.Active, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, err
		}
		m.Email = email.String
		m.Phone = phone.String
		members = append(members, m)
	}
	return members, nil
}

// Report queries

func GetMemberAttendanceHistory(memberID int64) ([]models.Hike, error) {
	rows, err := DB.Query(`
		SELECT h.id, h.name, h.date, h.location, h.notes, h.status, h.created_at
		FROM hikes h
		JOIN checkins c ON h.id = c.hike_id
		WHERE c.member_id = ?
		ORDER BY h.date DESC
	`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hikes []models.Hike
	for rows.Next() {
		var h models.Hike
		var location, notes sql.NullString
		err := rows.Scan(&h.ID, &h.Name, &h.Date, &location, &notes, &h.Status, &h.CreatedAt)
		if err != nil {
			return nil, err
		}
		h.Location = location.String
		h.Notes = notes.String
		hikes = append(hikes, h)
	}
	return hikes, nil
}

func GetAllAttendanceForYear(year string) ([]models.AttendanceRecord, error) {
	rows, err := DB.Query(`
		SELECT c.id, h.date, h.name, h.location, m.membership_number, m.first_name, m.last_name, c.checked_in_at
		FROM checkins c
		JOIN hikes h ON c.hike_id = h.id
		JOIN members m ON c.member_id = m.id
		WHERE h.date LIKE ?
		ORDER BY h.date ASC, h.name ASC, m.last_name ASC, m.first_name ASC
	`, year+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.AttendanceRecord
	for rows.Next() {
		var r models.AttendanceRecord
		var location sql.NullString
		err := rows.Scan(&r.CheckinID, &r.HikeDate, &r.HikeName, &location, &r.MembershipNumber, &r.FirstName, &r.LastName, &r.CheckedInAt)
		if err != nil {
			return nil, err
		}
		r.HikeLocation = location.String
		records = append(records, r)
	}
	return records, nil
}

// RSVP operations

func CreateRSVPForMember(hikeID, memberID int64) error {
	_, err := DB.Exec(
		"INSERT OR IGNORE INTO rsvps (hike_id, member_id) VALUES (?, ?)",
		hikeID, memberID,
	)
	return err
}

func CreateRSVPForGuest(hikeID int64, guestName string) error {
	_, err := DB.Exec(
		"INSERT OR IGNORE INTO rsvps (hike_id, guest_name) VALUES (?, ?)",
		hikeID, guestName,
	)
	return err
}

func DeleteRSVP(id int64) error {
	_, err := DB.Exec("DELETE FROM rsvps WHERE id = ?", id)
	return err
}

func GetRSVPsForHike(hikeID int64) ([]models.RSVP, error) {
	rows, err := DB.Query(`
		SELECT r.id, r.hike_id, r.member_id, r.guest_name, r.created_at,
		       m.first_name || ' ' || m.last_name as member_name,
		       m.membership_number,
		       CASE
		           WHEN c.id IS NOT NULL THEN 1
		           WHEN r.checked_in_at IS NOT NULL THEN 1
		           ELSE 0
		       END as checked_in
		FROM rsvps r
		LEFT JOIN members m ON r.member_id = m.id
		LEFT JOIN checkins c ON r.hike_id = c.hike_id AND r.member_id = c.member_id
		WHERE r.hike_id = ?
		ORDER BY COALESCE(m.last_name, r.guest_name), COALESCE(m.first_name, '')
	`, hikeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rsvps []models.RSVP
	for rows.Next() {
		var r models.RSVP
		var memberID sql.NullInt64
		var guestName, memberName, membershipNumber sql.NullString
		err := rows.Scan(&r.ID, &r.HikeID, &memberID, &guestName, &r.CreatedAt, &memberName, &membershipNumber, &r.CheckedIn)
		if err != nil {
			return nil, err
		}
		if memberID.Valid {
			r.MemberID = &memberID.Int64
			r.MemberName = memberName.String
			r.MembershipNumber = membershipNumber.String
		} else {
			r.GuestName = guestName.String
		}
		rsvps = append(rsvps, r)
	}
	return rsvps, nil
}

// CheckInRSVP checks in an RSVP - for members it creates a checkin record, for guests it sets checked_in_at
func CheckInRSVP(rsvpID int64) error {
	// Get the RSVP details
	var memberID sql.NullInt64
	var hikeID int64
	err := DB.QueryRow("SELECT hike_id, member_id FROM rsvps WHERE id = ?", rsvpID).Scan(&hikeID, &memberID)
	if err != nil {
		return err
	}

	if memberID.Valid {
		// Member RSVP - create a checkin record
		_, err = DB.Exec(
			"INSERT OR IGNORE INTO checkins (hike_id, member_id) VALUES (?, ?)",
			hikeID, memberID.Int64,
		)
	} else {
		// Guest RSVP - update the checked_in_at field
		_, err = DB.Exec(
			"UPDATE rsvps SET checked_in_at = CURRENT_TIMESTAMP WHERE id = ?",
			rsvpID,
		)
	}
	return err
}

func CloseRSVPs(hikeID int64) error {
	_, err := DB.Exec("UPDATE hikes SET rsvp_open = 0 WHERE id = ?", hikeID)
	return err
}

func OpenRSVPs(hikeID int64) error {
	_, err := DB.Exec("UPDATE hikes SET rsvp_open = 1 WHERE id = ?", hikeID)
	return err
}

func IsRSVPOpen(hikeID int64) (bool, error) {
	var open bool
	err := DB.QueryRow("SELECT rsvp_open FROM hikes WHERE id = ?", hikeID).Scan(&open)
	return open, err
}

// FuzzyMatchMember finds best matching member by name using simple similarity
func FuzzyMatchMember(firstName, lastName string) (*models.Member, int) {
	members, err := GetAllMembers(true)
	if err != nil || len(members) == 0 {
		return nil, 0
	}

	firstName = strings.ToLower(strings.TrimSpace(firstName))
	lastName = strings.ToLower(strings.TrimSpace(lastName))
	fullName := firstName + " " + lastName

	var bestMatch *models.Member
	bestScore := 0

	for i := range members {
		m := &members[i]
		mFirst := strings.ToLower(m.FirstName)
		mLast := strings.ToLower(m.LastName)
		mFull := mFirst + " " + mLast

		// Exact match
		if mFirst == firstName && mLast == lastName {
			return m, 100
		}

		// Calculate similarity score
		score := similarity(fullName, mFull)
		if score > bestScore {
			bestScore = score
			bestMatch = m
		}
	}

	return bestMatch, bestScore
}

// Simple similarity function (percentage of matching characters)
func similarity(a, b string) int {
	if a == b {
		return 100
	}
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	// Levenshtein-like comparison
	matches := 0
	aRunes := []rune(a)
	bRunes := []rune(b)

	shorter, longer := aRunes, bRunes
	if len(aRunes) > len(bRunes) {
		shorter, longer = bRunes, aRunes
	}

	for i, r := range shorter {
		if i < len(longer) && r == longer[i] {
			matches++
		}
	}

	return (matches * 100) / len(longer)
}

// Activity operations

func CreateActivity(hikeID int64, name string) (*models.Activity, error) {
	result, err := DB.Exec(
		"INSERT INTO activities (hike_id, name) VALUES (?, ?)",
		hikeID, name,
	)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return GetActivityByID(id)
}

func GetActivityByID(id int64) (*models.Activity, error) {
	var a models.Activity
	err := DB.QueryRow(`
		SELECT a.id, a.hike_id, a.name, a.created_at,
		       (SELECT COUNT(*) FROM activity_participants WHERE activity_id = a.id) as participant_count
		FROM activities a
		WHERE a.id = ?
	`, id).Scan(&a.ID, &a.HikeID, &a.Name, &a.CreatedAt, &a.ParticipantCount)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func GetActivitiesForHike(hikeID int64) ([]models.Activity, error) {
	rows, err := DB.Query(`
		SELECT a.id, a.hike_id, a.name, a.created_at,
		       (SELECT COUNT(*) FROM activity_participants WHERE activity_id = a.id) as participant_count
		FROM activities a
		WHERE a.hike_id = ?
		ORDER BY a.name
	`, hikeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []models.Activity
	for rows.Next() {
		var a models.Activity
		err := rows.Scan(&a.ID, &a.HikeID, &a.Name, &a.CreatedAt, &a.ParticipantCount)
		if err != nil {
			return nil, err
		}
		activities = append(activities, a)
	}
	return activities, nil
}

func DeleteActivity(id int64) error {
	_, err := DB.Exec("DELETE FROM activities WHERE id = ?", id)
	return err
}

func AddActivityParticipant(activityID int64, checkinID *int64, rsvpID *int64) error {
	_, err := DB.Exec(
		"INSERT OR IGNORE INTO activity_participants (activity_id, checkin_id, rsvp_id) VALUES (?, ?, ?)",
		activityID, checkinID, rsvpID,
	)
	return err
}

func RemoveActivityParticipant(activityID int64, checkinID *int64, rsvpID *int64) error {
	if checkinID != nil {
		_, err := DB.Exec("DELETE FROM activity_participants WHERE activity_id = ? AND checkin_id = ?", activityID, *checkinID)
		return err
	}
	if rsvpID != nil {
		_, err := DB.Exec("DELETE FROM activity_participants WHERE activity_id = ? AND rsvp_id = ?", activityID, *rsvpID)
		return err
	}
	return nil
}

func GetActivityParticipants(activityID int64) ([]models.ActivityParticipant, error) {
	rows, err := DB.Query(`
		SELECT ap.id, ap.activity_id, ap.checkin_id, ap.rsvp_id,
		       COALESCE(m.first_name || ' ' || m.last_name, r.guest_name) as name,
		       COALESCE(m.membership_number, '') as membership_number,
		       CASE WHEN r.guest_name IS NOT NULL AND r.member_id IS NULL THEN 1 ELSE 0 END as is_guest
		FROM activity_participants ap
		LEFT JOIN checkins c ON ap.checkin_id = c.id
		LEFT JOIN members m ON c.member_id = m.id
		LEFT JOIN rsvps r ON ap.rsvp_id = r.id
		LEFT JOIN members m2 ON r.member_id = m2.id
		WHERE ap.activity_id = ?
		ORDER BY name
	`, activityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []models.ActivityParticipant
	for rows.Next() {
		var p models.ActivityParticipant
		var checkinID, rsvpID sql.NullInt64
		err := rows.Scan(&p.ID, &p.ActivityID, &checkinID, &rsvpID, &p.Name, &p.MembershipNumber, &p.IsGuest)
		if err != nil {
			return nil, err
		}
		if checkinID.Valid {
			p.CheckinID = &checkinID.Int64
		}
		if rsvpID.Valid {
			p.RSVPID = &rsvpID.Int64
		}
		participants = append(participants, p)
	}
	return participants, nil
}

// GetCheckinIDForMember returns the checkin ID for a member in a hike
func GetCheckinIDForMember(hikeID, memberID int64) (*int64, error) {
	var id int64
	err := DB.QueryRow("SELECT id FROM checkins WHERE hike_id = ? AND member_id = ?", hikeID, memberID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// GetRSVPIDForGuest returns the RSVP ID for a guest in a hike
func GetRSVPIDForGuest(hikeID int64, guestName string) (*int64, error) {
	var id int64
	err := DB.QueryRow("SELECT id FROM rsvps WHERE hike_id = ? AND guest_name = ?", hikeID, guestName).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// GetActivitiesForCheckin returns activity names for a checkin
func GetActivitiesForCheckin(checkinID int64) ([]string, error) {
	rows, err := DB.Query(`
		SELECT a.name FROM activities a
		JOIN activity_participants ap ON a.id = ap.activity_id
		WHERE ap.checkin_id = ?
		ORDER BY a.name
	`, checkinID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		activities = append(activities, name)
	}
	return activities, nil
}

// GetActivitiesForRSVP returns activity names for a guest RSVP
func GetActivitiesForRSVP(rsvpID int64) ([]string, error) {
	rows, err := DB.Query(`
		SELECT a.name FROM activities a
		JOIN activity_participants ap ON a.id = ap.activity_id
		WHERE ap.rsvp_id = ?
		ORDER BY a.name
	`, rsvpID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		activities = append(activities, name)
	}
	return activities, nil
}
