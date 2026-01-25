# TrailCall 🥾

TrailCall is a lightweight Progressive Web App (PWA) designed for hiking club attendance tracking. It simplifies the check-in process for hike leaders using QR code scanning and manual check-ins, with a strong focus on offline reliability in the field.

## Key Features

- **Quick Check-in**: Scan member QR codes for instant attendance logging.
- **Combined Interface**: QR scanner and manual check-in list on the same page for efficiency.
- **Offline First**: Works without internet access. Check-ins are queued locally and synchronized automatically when back online.
- **Member Management**: Add and manage club members, and generate printable QR code cards.
- **RSVP Tracking**: Integration with RSVP links to pre-populate attendee lists.
- **Activity Tracking**: Group attendees into sub-activities (e.g., different pace groups).
- **Reporting**: Export attendance data for specific hikes or member histories to CSV.

## Tech Stack

- **Frontend**: Vanilla JavaScript (PWA), HTML5-QRCode for scanning, IndexedDB for offline storage.
- **Backend**: Go (single binary deployment).
- **Database**: SQLite.
- **Deployment**: Designed for easy hosting on Linux servers (includes Systemd service template).

## Getting Started

### Prerequisites
- Go 1.21+
- A modern web browser with PWA support (Chrome, Safari on iOS, etc.)

### Installation
1. Clone the repository.
2. Build the backend:
   ```bash
   go build -o trailcall
   ```
3. Set up your `.env` file with a `PIN` for authentication.
4. Run the application:
   ```bash
   ./trailcall
   ```

## Usage

1. **Create a Hike**: Start a new hike from the Home screen.
2. **Scan & Check-in**: Use the Scan tab to check in members via QR codes or select them from the manual RSVP list.
3. **Offline Mode**: If you lose signal, continue checking in. The app will show a "Pending Sync" indicator and upload data once connection is restored.
4. **Export**: Go to the Hikes list or a specific Hike Detail page to download attendance as a CSV.

## Development

The project is structured into:
- `handlers/`: Go API endpoints.
- `models/`: Database schema and Go structs.
- `frontend/`: PWA files, styles, and client-side logic.
- `db/`: SQLite initialization and migrations.

---
*Created for Centurion Hiking Club.*
