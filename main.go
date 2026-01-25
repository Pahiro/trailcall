package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"trailcall/db"
	"trailcall/handlers"
)

// handleRSVPRoutes handles /api/rsvps/{id}/checkin
func handleRSVPRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/rsvps/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	rsvpID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid RSVP ID", http.StatusBadRequest)
		return
	}

	switch parts[1] {
	case "checkin":
		handlers.HandleRSVPCheckin(w, r, rsvpID)
		return
	case "undo":
		handlers.HandleRSVPUndoCheckin(w, r, rsvpID)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

func main() {
	// Parse flags
	genHash := flag.String("gen-hash", "", "Generate bcrypt hash for a PIN")
	port := flag.Int("port", 2468, "Port to listen on")
	dbPath := flag.String("db", "trailcall.db", "Path to SQLite database")
	resetDB := flag.Bool("reset-db", false, "Delete all data and reset database")
	flag.Parse()

	// If generating a hash, do that and exit
	if *genHash != "" {
		hash, err := handlers.GeneratePINHash(*genHash)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Set this as TRAILCALL_PIN_HASH:")
		fmt.Println(hash)
		return
	}

	// If resetting database
	if *resetDB {
		if err := os.Remove(*dbPath); err != nil && !os.IsNotExist(err) {
			log.Fatal("Failed to delete database:", err)
		}
		log.Println("Database reset. Run again without -reset-db to start fresh.")
		return
	}

	// Check for required environment variable
	if os.Getenv("TRAILCALL_PIN_HASH") == "" {
		log.Println("Warning: TRAILCALL_PIN_HASH not set. Run with -gen-hash <pin> to generate one.")
	}

	// Initialize database
	if err := db.Init(*dbPath); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	log.Println("Database initialized:", *dbPath)

	// Set up routes
	mux := http.NewServeMux()

	// Auth endpoints (no auth required)
	mux.HandleFunc("/api/auth", handlers.HandleLogin)
	mux.HandleFunc("/api/auth/logout", handlers.HandleLogout)
	mux.HandleFunc("/api/auth/check", handlers.HandleCheckAuth)

	// Public RSVP endpoint (no auth required)
	mux.HandleFunc("/rsvp/", handlers.HandleRSVP)

	// Protected API endpoints
	mux.Handle("/api/members", handlers.AuthMiddleware(http.HandlerFunc(handlers.HandleMembers)))
	mux.Handle("/api/members/import", handlers.AuthMiddleware(http.HandlerFunc(handlers.HandleMembersImport)))
	mux.Handle("/api/members/", handlers.AuthMiddleware(http.HandlerFunc(handlers.HandleMember)))
	mux.Handle("/api/hikes", handlers.AuthMiddleware(http.HandlerFunc(handlers.HandleHikes)))
	mux.Handle("/api/hikes/", handlers.AuthMiddleware(http.HandlerFunc(handlers.HandleHike)))
	mux.Handle("/api/checkins", handlers.AuthMiddleware(http.HandlerFunc(handlers.HandleCheckins)))
	mux.Handle("/api/checkins/", handlers.AuthMiddleware(http.HandlerFunc(handlers.HandleCheckins)))
	mux.Handle("/api/rsvps/", handlers.AuthMiddleware(http.HandlerFunc(handleRSVPRoutes)))
	mux.Handle("/api/activities/", handlers.AuthMiddleware(http.HandlerFunc(handlers.HandleActivity)))
	mux.Handle("/api/reports/", handlers.AuthMiddleware(http.HandlerFunc(handlers.HandleReports)))

	// Serve frontend static files
	frontend := http.FileServer(http.Dir("frontend"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// For SPA: serve index.html for all non-file routes
		if !strings.Contains(r.URL.Path, ".") && r.URL.Path != "/" {
			http.ServeFile(w, r, "frontend/index.html")
			return
		}
		frontend.ServeHTTP(w, r)
	})

	// Start server
	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	log.Printf("Starting TrailCall server on http://0.0.0.0:%d", *port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
