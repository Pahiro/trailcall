# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TrailCall is a lightweight PWA for hiking club attendance tracking using QR code scanning. See `spec.md` for the complete project specification including data models, API endpoints, and UI screens.

## Tech Stack

- **Backend**: Go with SQLite (single binary deployment)
- **Frontend**: Vanilla JavaScript PWA with html5-qrcode library
- **Deployment**: Headless Linux server via cloudflared tunnel

## Setup

```bash
# Generate PIN hash (run once)
go run . -gen-hash "your-pin-here"

# Set the environment variable with the hash output (use single quotes to avoid $ expansion)
export TRAILCALL_PIN_HASH='$2a$10$...'

# Run the server
go run .
# Or build and run:
go build -o trailcall && ./trailcall
```

Server runs on port 2468 by default, binding to 0.0.0.0 for LAN access.

## Commands

```bash
# Development
go run .

# Build binary
go build -o trailcall

# Run tests
go test ./...

# Run with custom port/db
./trailcall -port 8080 -db /path/to/trailcall.db

# Reset database (delete all data)
go run . -reset-db
```

## Architecture

### Entry Points
- Backend: `main.go`
- Frontend: `frontend/index.html` (PWA shell)

### Core Data Flow
1. PWA captures QR codes via device camera (`js/scanner.js`)
2. QR format: `TC-###` (membership number with prefix)
3. Check-ins stored in IndexedDB when offline (`js/offline.js`)
4. Auto-sync to backend when connection restored
5. Backend validates and persists to SQLite

### Key Modules
- `handlers/auth.go` - PIN authentication with session cookies
- `handlers/members.go` - Member CRUD, QR generation, CSV import
- `handlers/hikes.go` - Hike CRUD, open/close workflow
- `handlers/checkins.go` - Attendance logging with bulk sync support
- `handlers/reports.go` - CSV exports (per-hike, per-member, full year attendance)
- `db/db.go` - SQLite database layer with migrations
- `frontend/js/app.js` - SPA router and view rendering
- `frontend/js/scanner.js` - html5-qrcode integration
- `frontend/js/offline.js` - IndexedDB storage and background sync
- `frontend/sw.js` - Service worker for offline capability

### CSV Import
Members can be imported via CSV with columns: `membership_number`, `first_name`, `last_name`, `email` (optional), `phone` (optional). Duplicates are skipped based on membership number.

### Reports API
- `/api/reports/hike/{id}/csv` - Attendance for a single hike
- `/api/reports/member/{id}/csv` - Hike history for a member
- `/api/reports/hikes/csv` - All hikes summary
- `/api/reports/attendance?year=2026` - Full attendance for the year (all check-ins)

### Offline-First Design
Critical for outdoor hiking scenarios. The PWA must:
- Cache app shell and member list via service worker
- Queue check-ins in IndexedDB when offline
- Sync automatically when connection restored
- Show pending sync count to user

## Deployment

```bash
# On server: copy folder to /opt/trailcall
sudo cp -r trailcall /opt/trailcall
cd /opt/trailcall

# Build the binary
go build -o trailcall

# Create .env file with your PIN hash
sudo cp .env.example .env
sudo nano .env  # Add your actual PIN hash

# Install and start service
sudo cp trailcall.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable trailcall
sudo systemctl start trailcall

# Check status
sudo systemctl status trailcall
sudo journalctl -u trailcall -f
```

## Development Context

- Target: Small hiking club (~100-500 members)
- Deployment: Existing Linux server running Invoice Ninja, served via cloudflared
- Priority: Reliability over features
- Approach: Minimal dependencies, single binary deployment
