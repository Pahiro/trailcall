# TrailCall

A lightweight PWA for hiking club attendance tracking using QR code scanning.

## Project Overview

TrailCall simplifies hike check-ins by allowing leaders to scan QR codes on membership cards. It replaces manual roll-calls with a quick scan workflow that works on any smartphone.

## Tech Stack

### Frontend (PWA)
- Vanilla JavaScript with minimal dependencies
- html5-qrcode library for barcode/QR scanning
- Service worker for offline capability
- Simple, mobile-first UI

### Backend
- Go (single binary deployment)
- SQLite database
- Hosted alongside Invoice Ninja on existing club Linux server
- Served via cloudflared tunnel

## Core Features (MVP)

### Member Management
- Add/edit/remove members
- Generate QR codes for membership cards (encode member ID)
- Print-friendly card template with QR code

### Check-in
- Camera-based QR scanning
- Log check-in with timestamp and hike identifier
- Visual/audio feedback on successful scan
- Display member name on scan for verification
- Offline queue with sync when back online

### Hike Management
- Create a new hike (date, name, location)
- View who's checked in (live list)
- Close hike and finalise attendance

### Reporting
- Attendance history per member (chronological list of hikes attended)
- Attendance list per hike
- Export any view to CSV

## Data Model

### members
- id (primary key)
- membership_number (unique, used in QR)
- first_name
- last_name
- email (optional)
- phone (optional)
- active (boolean)
- created_at
- updated_at

### hikes
- id (primary key)
- name
- date
- location (optional)
- notes (optional)
- status (open/closed)
- created_at

### checkins
- id (primary key)
- hike_id (foreign key)
- member_id (foreign key)
- checked_in_at
- synced (boolean, for offline handling)

## API Endpoints

### Members
- GET /api/members - list all active members
- GET /api/members/{id} - get member details
- POST /api/members - create member
- PUT /api/members/{id} - update member
- DELETE /api/members/{id} - soft delete (set inactive)
- GET /api/members/{id}/qr - generate QR code image

### Hikes
- GET /api/hikes - list hikes (with filters)
- GET /api/hikes/{id} - get hike with attendance
- POST /api/hikes - create hike
- PUT /api/hikes/{id} - update hike
- POST /api/hikes/{id}/close - close hike

### Check-ins
- POST /api/checkins - log a check-in
- POST /api/checkins/bulk - sync offline check-ins
- GET /api/hikes/{id}/checkins - get all check-ins for a hike

### Reports
- GET /api/reports/member/{id} - attendance history for member (JSON)
- GET /api/reports/member/{id}/csv - export member attendance history as CSV
- GET /api/reports/hike/{id}/csv - export hike attendance as CSV
- GET /api/reports/hikes/csv - export all hikes summary as CSV

## Project Structure

```
trailcall/
├── main.go                  # Entry point
├── handlers/
│   ├── members.go
│   ├── hikes.go
│   ├── checkins.go
│   └── reports.go
├── db/
│   ├── db.go                # SQLite setup and connection
│   └── migrations.go        # Schema migrations
├── models/
│   └── models.go            # Data structures
├── frontend/
│   ├── index.html           # Main app shell
│   ├── css/
│   │   └── style.css
│   ├── js/
│   │   ├── app.js           # Main app logic
│   │   ├── scanner.js       # QR scanning module
│   │   ├── api.js           # API client
│   │   └── offline.js       # Offline queue and sync
│   ├── manifest.json        # PWA manifest
│   └── sw.js                # Service worker
├── go.mod
├── go.sum
└── spec.md                  # This file
```

## QR Code Format

Simple plain text containing the membership number:
```
TC-001
```

Prefix `TC-` helps validate it's a TrailCall code and not random noise.

## Offline Behaviour

1. Service worker caches app shell and member list
2. Check-ins stored in IndexedDB when offline
3. Sync indicator shows pending check-ins count
4. Auto-sync when connection restored
5. Manual sync button as fallback

## UI Screens

1. **Home** - Start new hike or continue open hike
2. **Scanner** - Camera view with scan button, shows last scanned member
3. **Attendance** - List of checked-in members for current hike, download CSV button
4. **Members** - Member list with search, add/edit capability
5. **Member History** - All hikes attended by a member (chronological), download CSV button
6. **Hikes** - Hike history with attendance counts, download CSV button
7. **Hike Detail** - Attendee list for a specific hike, download CSV button
8. **Settings** - Club branding, sync status

## Navigation

- **Members list** → tap member → **Member History** (hikes they attended)
- **Member History** → tap hike → **Hike Detail** (who else was there)
- **Hikes list** → tap hike → **Hike Detail** (attendee list)
- **Hike Detail** → tap member → **Member History**
- Each list view has a download CSV button in the header/toolbar

## Design Notes

- Mobile-first, thumb-friendly buttons
- High contrast for outdoor visibility
- Large scan button, clear feedback
- Club logo and colours in header
- Minimal text input required during check-in

## Future Enhancements (Post-MVP)

- Member self-registration via QR link
- Integration with Invoice Ninja for membership fees
- Hike leader assignment and permissions
- GPS logging of check-in location
- Statistics dashboard (most active members, attendance trends)
- Email notifications for upcoming hikes
- Bulk member import from CSV

## Development Notes

- Ben is comfortable with Python and Linux administration
- Server already runs Invoice Ninja, has existing backup workflows
- Keep dependencies minimal
- SQLite is sufficient for club scale (~100s of members)
- Prioritise reliability over features
