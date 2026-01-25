package models

import "time"

type Member struct {
	ID               int64     `json:"id"`
	MembershipNumber string    `json:"membership_number"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	Email            string    `json:"email,omitempty"`
	Phone            string    `json:"phone,omitempty"`
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Hike struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Date          string    `json:"date"`
	Location      string    `json:"location,omitempty"`
	Notes         string    `json:"notes,omitempty"`
	Status        string    `json:"status"` // "open" or "closed"
	RSVPOpen      bool      `json:"rsvp_open"`
	CreatedAt     time.Time `json:"created_at"`
	AttendeeCount int       `json:"attendee_count"`
	RSVPCount     int       `json:"rsvp_count"`
}

type Checkin struct {
	ID          int64     `json:"id"`
	HikeID      int64     `json:"hike_id"`
	MemberID    int64     `json:"member_id"`
	CheckedInAt time.Time `json:"checked_in_at"`
	Synced      bool      `json:"synced"`
	IsLeader    bool      `json:"is_leader"`
	IsSweeper   bool      `json:"is_sweeper"`
	// Joined fields for display
	MemberName       string `json:"member_name,omitempty"`
	MembershipNumber string `json:"membership_number,omitempty"`
}

type Activity struct {
	ID               int64     `json:"id"`
	HikeID           int64     `json:"hike_id"`
	Name             string    `json:"name"`
	CreatedAt        time.Time `json:"created_at"`
	ParticipantCount int       `json:"participant_count"`
}

type ActivityParticipant struct {
	ID         int64  `json:"id"`
	ActivityID int64  `json:"activity_id"`
	CheckinID  *int64 `json:"checkin_id,omitempty"`
	RSVPID     *int64 `json:"rsvp_id,omitempty"`
	// Joined fields
	Name             string `json:"name"`
	MembershipNumber string `json:"membership_number,omitempty"`
	IsGuest          bool   `json:"is_guest"`
}

// Request/response types

type CreateMemberRequest struct {
	MembershipNumber string `json:"membership_number"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	Email            string `json:"email,omitempty"`
	Phone            string `json:"phone,omitempty"`
}

type UpdateMemberRequest struct {
	MembershipNumber string `json:"membership_number,omitempty"`
	FirstName        string `json:"first_name,omitempty"`
	LastName         string `json:"last_name,omitempty"`
	Email            string `json:"email,omitempty"`
	Phone            string `json:"phone,omitempty"`
	Active           *bool  `json:"active,omitempty"`
}

type CreateHikeRequest struct {
	Name     string `json:"name"`
	Date     string `json:"date"`
	Location string `json:"location,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

type UpdateHikeRequest struct {
	Name     string `json:"name,omitempty"`
	Date     string `json:"date,omitempty"`
	Location string `json:"location,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

type CreateCheckinRequest struct {
	HikeID           int64  `json:"hike_id"`
	MembershipNumber string `json:"membership_number"`
}

type BulkCheckinRequest struct {
	Checkins []CreateCheckinRequest `json:"checkins"`
}

type AuthRequest struct {
	PIN string `json:"pin"`
}

type MemberHistory struct {
	Member Member `json:"member"`
	Hikes  []Hike `json:"hikes"`
}

type HikeDetail struct {
	Hike      Hike     `json:"hike"`
	Attendees []Member `json:"attendees"`
}

type AttendanceRecord struct {
	CheckinID        int64     `json:"checkin_id"`
	HikeDate         string    `json:"hike_date"`
	HikeName         string    `json:"hike_name"`
	HikeLocation     string    `json:"hike_location"`
	MembershipNumber string    `json:"membership_number"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	CheckedInAt      time.Time `json:"checked_in_at"`
}

type RSVP struct {
	ID        int64     `json:"id"`
	HikeID    int64     `json:"hike_id"`
	MemberID  *int64    `json:"member_id,omitempty"`
	GuestName string    `json:"guest_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	// Joined fields
	MemberName       string `json:"member_name,omitempty"`
	MembershipNumber string `json:"membership_number,omitempty"`
	CheckedIn        bool   `json:"checked_in"`
}

type RSVPRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type RSVPResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	MatchedName  string `json:"matched_name,omitempty"`
	MemberNumber string `json:"member_number,omitempty"`
	IsGuest      bool   `json:"is_guest"`
}
