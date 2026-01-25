// TrailCall main app

const App = {
    currentView: null,
    currentHike: null,
    members: [],
    isAuthenticated: false,
    audioCtx: null,

    async init() {
        // Initialize audio context on first user interaction
        const initAudio = () => {
            if (!this.audioCtx) {
                this.audioCtx = new (window.AudioContext || window.webkitAudioContext)();
            }
            if (this.audioCtx.state === 'suspended') {
                this.audioCtx.resume();
            }
        };
        document.addEventListener('click', initAudio, { once: false });
        document.addEventListener('touchstart', initAudio, { once: false });
        // Check authentication
        try {
            const auth = await API.checkAuth();
            this.isAuthenticated = auth.authenticated;
        } catch (e) {
            this.isAuthenticated = false;
        }

        // Set up navigation
        document.querySelectorAll('.nav-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const view = btn.dataset.view;
                window.location.hash = `#${view}`;
            });
        });

        // Logout button
        document.getElementById('logout-btn').addEventListener('click', async () => {
            await API.logout();
            this.isAuthenticated = false;
            window.location.hash = '#login';
        });

        // Handle hash changes
        window.addEventListener('hashchange', () => this.route());

        // Initial route
        this.route();

        // Register service worker
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/sw.js').catch(console.error);
        }
    },

    route() {
        const hash = window.location.hash.slice(1) || 'home';
        const [view, ...params] = hash.split('/');

        if (!this.isAuthenticated && view !== 'login') {
            window.location.hash = '#login';
            return;
        }

        if (this.isAuthenticated && view === 'login') {
            window.location.hash = '#home';
            return;
        }

        // Update nav
        document.querySelectorAll('.nav-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.view === view);
        });

        // Show/hide nav and logout based on auth
        document.getElementById('nav').style.display = this.isAuthenticated ? 'flex' : 'none';
        document.getElementById('logout-btn').style.display = this.isAuthenticated ? 'block' : 'none';

        // Render view
        this.currentView = view;
        switch (view) {
            case 'login':
                this.renderLogin();
                break;
            case 'home':
                this.renderHome();
                break;
            case 'scanner':
                this.renderScanner();
                break;
            case 'attendance':
                this.renderAttendance(params[0]);
                break;
            case 'members':
                this.renderMembers();
                break;
            case 'member':
                this.renderMemberDetail(params[0]);
                break;
            case 'member-history':
                this.renderMemberHistory(params[0]);
                break;
            case 'hikes':
                this.renderHikes();
                break;
            case 'hike':
                this.renderHikeDetail(params[0]);
                break;
            case 'new-hike':
                this.renderNewHike();
                break;
            case 'new-member':
                this.renderNewMember();
                break;
            case 'edit-member':
                this.renderEditMember(params[0]);
                break;
            default:
                this.renderHome();
        }
    },

    renderLogin() {
        const app = document.getElementById('app');
        app.innerHTML = `
            <div class="login-container">
                <img src="https://centurionhikingclub.co.za/wp-content/uploads/2023/02/CHC-logo-2-white-1024x1024-1.jpg" alt="CHC Logo" class="logo">
                <h2>TrailCall</h2>
                <p>Centurion Hiking Club</p>
                <form id="login-form" class="card">
                    <div class="form-group">
                        <label for="pin">Enter PIN</label>
                        <input type="password" id="pin" class="pin-input" placeholder="****" maxlength="20" autocomplete="off">
                    </div>
                    <button type="submit" class="btn btn-primary btn-block">Login</button>
                </form>
            </div>
        `;

        document.getElementById('login-form').addEventListener('submit', async (e) => {
            e.preventDefault();
            const pin = document.getElementById('pin').value;
            try {
                await API.login(pin);
                this.isAuthenticated = true;

                // Cache data for offline use after login
                if (typeof SyncManager !== 'undefined') {
                    Toast.show('Caching data for offline use...');
                    SyncManager.refreshMemberCache();
                    SyncManager.refreshHikeCache();
                }

                window.location.hash = '#home';
            } catch (err) {
                Toast.show('Invalid PIN', 'error');
            }
        });
    },

    async renderHome() {
        const app = document.getElementById('app');
        app.innerHTML = '<div class="empty-state">Loading...</div>';

        try {
            this.currentHike = await API.getOpenHike();
            // Cache for offline use
            if (this.currentHike && typeof OfflineStore !== 'undefined') {
                await OfflineStore.cacheCurrentHike(this.currentHike);
            }
        } catch (e) {
            // If offline, try cached data
            if (!navigator.onLine && typeof OfflineStore !== 'undefined') {
                this.currentHike = await OfflineStore.getCachedCurrentHike();
            } else {
                this.currentHike = null;
            }
        }

        if (this.currentHike) {
            app.innerHTML = `
                <div class="card current-hike">
                    <h2>Current Hike</h2>
                    <h3>${this.currentHike.name}</h3>
                    <p>${this.currentHike.date} ${this.currentHike.location ? '• ' + this.currentHike.location : ''}</p>
                    <div class="attendee-count">${this.currentHike.attendee_count}</div>
                    <p>checked in</p>
                </div>
                <div class="stats-row">
                    <button class="btn btn-accent btn-block btn-large" onclick="window.location.hash='#scanner'">
                        Start Scanning
                    </button>
                </div>
                <div class="stats-row">
                    <button class="btn btn-secondary btn-block" onclick="window.location.hash='#attendance/${this.currentHike.id}'">
                        View Attendance
                    </button>
                    <button class="btn btn-secondary" onclick="App.closeCurrentHike()">
                        Close Hike
                    </button>
                </div>
            `;
        } else {
            app.innerHTML = `
                <div class="card">
                    <div class="empty-state">
                        <div class="empty-state-icon">🥾</div>
                        <h3>No Active Hike</h3>
                        <p>Start a new hike to begin check-ins</p>
                    </div>
                    <button class="btn btn-primary btn-block btn-large" onclick="window.location.hash='#new-hike'">
                        Start New Hike
                    </button>
                </div>
            `;
        }
    },

    async closeCurrentHike() {
        if (!this.currentHike) return;
        if (!confirm(`Close "${this.currentHike.name}"? This cannot be undone.`)) return;

        try {
            await API.closeHike(this.currentHike.id);
            Toast.show('Hike closed');
            this.currentHike = null;
            this.renderHome();
        } catch (err) {
            Toast.show('Failed to close hike', 'error');
        }
    },

    renderNewHike() {
        const today = new Date().toISOString().split('T')[0];
        const app = document.getElementById('app');
        app.innerHTML = `
            <div class="card">
                <h2>Start New Hike</h2>
                <form id="new-hike-form">
                    <div class="form-group">
                        <label for="hike-name">Hike Name *</label>
                        <input type="text" id="hike-name" required placeholder="e.g., Hennops Morning Walk">
                    </div>
                    <div class="form-group">
                        <label for="hike-date">Date *</label>
                        <input type="date" id="hike-date" required value="${today}">
                    </div>
                    <div class="form-group">
                        <label for="hike-location">Location</label>
                        <input type="text" id="hike-location" placeholder="e.g., Hennops Hiking Trail">
                    </div>
                    <div class="form-group">
                        <label for="hike-notes">Notes</label>
                        <textarea id="hike-notes" rows="3" placeholder="Any additional notes..."></textarea>
                    </div>
                    <button type="submit" class="btn btn-primary btn-block">Start Hike</button>
                </form>
            </div>
        `;

        document.getElementById('new-hike-form').addEventListener('submit', async (e) => {
            e.preventDefault();
            try {
                const hike = await API.createHike({
                    name: document.getElementById('hike-name').value,
                    date: document.getElementById('hike-date').value,
                    location: document.getElementById('hike-location').value,
                    notes: document.getElementById('hike-notes').value,
                });
                this.currentHike = hike;
                Toast.show('Hike started!', 'success');
                window.location.hash = '#scanner';
            } catch (err) {
                Toast.show('Failed to create hike', 'error');
            }
        });
    },

    async renderScanner() {
        const app = document.getElementById('app');

        if (!this.currentHike) {
            try {
                this.currentHike = await API.getOpenHike();
                // Cache the hike for offline use
                if (this.currentHike && typeof OfflineStore !== 'undefined') {
                    await OfflineStore.cacheCurrentHike(this.currentHike);
                }
            } catch (e) {
                // If offline, try to use cached hike
                if (!navigator.onLine && typeof OfflineStore !== 'undefined') {
                    this.currentHike = await OfflineStore.getCachedCurrentHike();
                } else {
                    this.currentHike = null;
                }
            }
        }

        if (!this.currentHike) {
            const offlineMsg = !navigator.onLine ? '<p style="color: var(--warning);">You are offline. Cached hike data not available.</p>' : '';
            app.innerHTML = `
                <div class="card">
                    <div class="empty-state">
                        <div class="empty-state-icon">📷</div>
                        <h3>No Active Hike</h3>
                        <p>Start a hike first to begin scanning</p>
                        ${offlineMsg}
                    </div>
                    ${navigator.onLine ? `<button class="btn btn-primary btn-block" onclick="window.location.hash='#new-hike'">
                        Start New Hike
                    </button>` : ''}
                </div>
            `;
            return;
        }

        // Get pending count for offline indicator
        let pendingCount = 0;
        if (typeof OfflineStore !== 'undefined') {
            try {
                pendingCount = await OfflineStore.getPendingCount();
            } catch (e) { }
        }

        const offlineBanner = !navigator.onLine ? `
            <div class="card" style="background: var(--warning); color: var(--text);">
                <p style="margin: 0; text-align: center;">Offline Mode - Check-ins saved locally</p>
            </div>
        ` : '';

        const pendingBanner = pendingCount > 0 ? `
            <div class="sync-indicator" style="margin-bottom: 16px;">
                ${pendingCount} pending sync
            </div>
        ` : '';

        app.innerHTML = `
            ${offlineBanner}
            <div class="card">
                <div class="card-header">
                    <div>
                        <h2>${this.currentHike.name}</h2>
                        <p>${this.currentHike.date}</p>
                    </div>
                </div>
                ${pendingBanner}
            </div>
            <div class="scanner-container">
                <div id="scanner"></div>
            </div>
            <div id="scan-result"></div>

            <div id="manual-checkin" class="card">
                <div class="card-header">
                    <h3>Manual Check-in</h3>
                    <button class="btn btn-small btn-secondary" onclick="App.loadRSVPList()">↻</button>
                </div>
                <ul class="list" id="rsvp-list">
                    <li class="empty-state">Loading RSVPs...</li>
                </ul>
            </div>

            <div id="recent-scans" class="card">
                <h3>${navigator.onLine ? 'Recent Check-ins' : 'Recent Check-ins (offline)'}</h3>
                <ul class="list" id="recent-list"></ul>
            </div>
        `;

        // Initialize scanner
        Scanner.init('scanner', async (code) => {
            await this.handleScan(code);
        });

        // Load data
        this.loadRecentCheckins();
        this.loadRSVPList();
    },

    async loadRSVPList() {
        if (!this.currentHike) return;
        const list = document.getElementById('rsvp-list');
        if (!list) return;

        let rsvps = [];
        try {
            if (navigator.onLine) {
                rsvps = await API.getHikeRSVPs(this.currentHike.id) || [];
                // Cache for offline use
                if (typeof OfflineStore !== 'undefined') {
                    await OfflineStore.cacheRSVPs(this.currentHike.id, rsvps);
                }
            } else if (typeof OfflineStore !== 'undefined') {
                rsvps = await OfflineStore.getCachedRSVPs(this.currentHike.id) || [];
            }
        } catch (err) {
            console.error('Failed to load RSVPs:', err);
            list.innerHTML = '<li class="empty-state">Could not load RSVPs</li>';
            return;
        }

        const notCheckedIn = rsvps.filter(r => !r.checked_in);

        if (notCheckedIn.length === 0) {
            list.innerHTML = '<li class="empty-state">No pending RSVPs</li>';
            return;
        }

        list.innerHTML = notCheckedIn.map(r => `
            <li class="list-item">
                <div class="list-item-content">
                    <div class="list-item-title">${r.member_name || r.guest_name}</div>
                    <div class="list-item-subtitle">${r.membership_number || 'Guest'}</div>
                </div>
                <button class="btn btn-small btn-primary" onclick="App.handleManualCheckin(${JSON.stringify(r).replace(/"/g, '&quot;')})">
                    Check In
                </button>
            </li>
        `).join('');
    },

    async handleManualCheckin(rsvp) {
        try {
            if (navigator.onLine) {
                await API.checkinRSVP(rsvp.id);
                Toast.show(`Checked in ${rsvp.member_name || rsvp.guest_name}`, 'success');
            } else {
                if (typeof OfflineStore === 'undefined') {
                    throw new Error('Offline store not available');
                }

                // If it's a member, we can use their membership number for standard offline check-in
                if (rsvp.membership_number) {
                    await OfflineStore.addPendingCheckin(this.currentHike.id, rsvp.membership_number);
                } else {
                    // It's a guest RSVP being checked in offline. 
                    // This is a bit of a special case since bulkCheckin expects a membership number.
                    // For now, we'll store it as a special "guest" check-in if we wanted to be robust,
                    // but standard bulkCheckin doesn't support guest names yet.
                    // Let's at least mark the RSVP as checked in locally.
                    Toast.show('Guest offline check-in not fully supported yet', 'warning');
                    return;
                }

                // Update local cache to reflect as checked in
                await OfflineStore.updateCachedRSVPStatus(rsvp.id, true);

                const pendingCount = await OfflineStore.getPendingCount();
                Toast.show(`Saved offline (${pendingCount} pending)`, 'warning');
            }

            // Play success beep
            this.playBeep();

            // Refresh lists
            this.loadRSVPList();
            this.loadRecentCheckins();

            // Update attendee count
            if (this.currentHike) {
                this.currentHike.attendee_count++;
            }

        } catch (err) {
            Toast.show(err.message || 'Check-in failed', 'error');
        }
    },

    async handleScan(code) {
        const resultDiv = document.getElementById('scan-result');

        // Validate TC-### format
        if (!code.match(/^TC-\d+$/)) {
            resultDiv.innerHTML = `
                <div class="scan-result error">
                    <h3>Invalid Code</h3>
                    <p>${code}</p>
                </div>
            `;
            return;
        }

        // If online, use the API
        if (navigator.onLine) {
            try {
                const checkin = await API.createCheckin(this.currentHike.id, code);

                resultDiv.innerHTML = `
                    <div class="scan-result success">
                        <h3>${checkin.member_name}</h3>
                        <p>${checkin.membership_number}</p>
                    </div>
                `;

                // Play success beep
                this.playBeep();

                // Update recent list
                this.loadRecentCheckins();

                // Refresh RSVP list
                this.loadRSVPList();

                // Update attendee count
                this.currentHike.attendee_count++;

            } catch (err) {
                resultDiv.innerHTML = `
                    <div class="scan-result error">
                        <h3>Error</h3>
                        <p>${err.message}</p>
                    </div>
                `;
            }
        } else {
            // Offline mode - use cached data
            try {
                if (typeof OfflineStore === 'undefined') {
                    throw new Error('Offline store not available');
                }

                // Look up member from cache
                const member = await OfflineStore.getCachedMemberByNumber(code);

                if (!member) {
                    resultDiv.innerHTML = `
                        <div class="scan-result error">
                            <h3>Member Not Found</h3>
                            <p>${code} not in cached member list</p>
                        </div>
                    `;
                    return;
                }

                // Store pending check-in
                await OfflineStore.addPendingCheckin(this.currentHike.id, code);

                resultDiv.innerHTML = `
                    <div class="scan-result success">
                        <h3>${member.first_name} ${member.last_name}</h3>
                        <p>${member.membership_number}</p>
                        <p style="font-size: 0.75rem; color: var(--warning);">Saved offline - will sync later</p>
                    </div>
                `;

                // Play success beep
                this.playBeep();

                // Show pending count
                const pendingCount = await OfflineStore.getPendingCount();
                Toast.show(`Saved offline (${pendingCount} pending)`, 'warning');

                // Update cached RSVP if applicable
                const rsvps = await OfflineStore.getCachedRSVPs(this.currentHike.id);
                const rsvp = rsvps.find(r => r.membership_number === code);
                if (rsvp) {
                    await OfflineStore.updateCachedRSVPStatus(rsvp.id, true);
                    this.loadRSVPList();
                }

                // Update recent list
                this.loadRecentCheckins();

            } catch (err) {
                resultDiv.innerHTML = `
                    <div class="scan-result error">
                        <h3>Offline Error</h3>
                        <p>${err.message}</p>
                    </div>
                `;
            }
        }
    },

    async loadRecentCheckins() {
        if (!this.currentHike) return;

        const list = document.getElementById('recent-list');
        if (!list) return;

        // If offline, show pending check-ins from IndexedDB
        if (!navigator.onLine && typeof OfflineStore !== 'undefined') {
            try {
                const pending = await OfflineStore.getPendingCheckins();
                const hikeCheckins = pending.filter(p => p.hikeId === this.currentHike.id);

                if (hikeCheckins.length === 0) {
                    list.innerHTML = '<li class="empty-state">No pending check-ins</li>';
                    return;
                }

                // Get member names from cache
                const items = await Promise.all(hikeCheckins.map(async (c) => {
                    const member = await OfflineStore.getCachedMemberByNumber(c.membershipNumber);
                    return {
                        name: member ? `${member.first_name} ${member.last_name}` : 'Unknown',
                        membershipNumber: c.membershipNumber,
                        timestamp: c.timestamp
                    };
                }));

                list.innerHTML = items.slice(0, 10).map(c => `
                    <li class="list-item">
                        <div class="list-item-content">
                            <div class="list-item-title">${c.name}</div>
                            <div class="list-item-subtitle">${c.membershipNumber}</div>
                        </div>
                        <span class="attendance-time" style="color: var(--warning);">Pending</span>
                    </li>
                `).join('');
            } catch (err) {
                list.innerHTML = '<li class="empty-state">Could not load pending check-ins</li>';
            }
            return;
        }

        // Online - fetch from API
        try {
            const checkins = await API.getHikeCheckins(this.currentHike.id);

            if (checkins.length === 0) {
                list.innerHTML = '<li class="empty-state">No check-ins yet</li>';
                return;
            }

            list.innerHTML = checkins.slice(0, 10).map(c => `
                <li class="list-item" onclick="window.location.hash='#member-history/${c.member_id}'">
                    <div class="list-item-content">
                        <div class="list-item-title">${c.member_name}</div>
                        <div class="list-item-subtitle">${c.membership_number}</div>
                    </div>
                    <span class="attendance-time">${new Date(c.checked_in_at).toLocaleTimeString()}</span>
                </li>
            `).join('');
        } catch (err) {
            console.error('Failed to load checkins:', err);
            list.innerHTML = '<li class="empty-state">Could not load check-ins</li>';
        }
    },

    async renderAttendance(hikeId) {
        const app = document.getElementById('app');
        app.innerHTML = '<div class="empty-state">Loading...</div>';

        try {
            const detail = await API.getHike(hikeId);
            const hike = detail.hike;
            const attendees = detail.attendees;

            let rsvps = [];
            try {
                rsvps = await API.getHikeRSVPs(hikeId) || [];
            } catch (e) {
                console.log('Could not load RSVPs');
            }

            let activities = [];
            try {
                activities = await API.getActivities(hikeId) || [];
            } catch (e) {
                console.log('Could not load activities');
            }

            const checkedInIds = new Set(attendees.map(a => a.id));
            const rsvpNotCheckedIn = rsvps.filter(r => !r.checked_in);
            const checkedInGuests = rsvps.filter(r => r.checked_in && !r.member_id);

            app.innerHTML = `
                <div class="card">
                    <div class="card-header">
                        <div>
                            <h2>${hike.name}</h2>
                            <p>${hike.date} ${hike.location ? '• ' + hike.location : ''}</p>
                        </div>
                        <a href="${API.getHikeCSVUrl(hikeId)}" class="download-btn" download>
                            CSV
                        </a>
                    </div>
                    <div class="stats-row">
                        <div class="stat-card">
                            <div class="stat-value">${rsvps.length}</div>
                            <div class="stat-label">RSVPs</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-value">${attendees.length + checkedInGuests.length}</div>
                            <div class="stat-label">Checked In</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-value">${rsvpNotCheckedIn.length}</div>
                            <div class="stat-label">Missing</div>
                        </div>
                    </div>
                </div>

                <div class="card">
                    <div class="card-header">
                        <h3>Activities</h3>
                        <button class="btn btn-small btn-primary" onclick="App.showAddActivityForm(${hikeId})">
                            + Add
                        </button>
                    </div>
                    <div id="add-activity-form" style="display: none; margin-bottom: 12px;">
                        <div style="display: flex; gap: 8px;">
                            <input type="text" id="new-activity-name" placeholder="Activity name..."
                                   style="flex: 1; padding: 8px; border: 1px solid var(--border); border-radius: 6px;">
                            <button class="btn btn-primary btn-small" onclick="App.createActivity(${hikeId})">Add</button>
                            <button class="btn btn-secondary btn-small" onclick="App.hideAddActivityForm()">Cancel</button>
                        </div>
                    </div>
                    ${activities.length === 0 ? '<p style="color: var(--text-light); font-size: 0.875rem;">No activities yet</p>' : `
                    <ul class="list">
                        ${activities.map(a => `
                            <li class="list-item" style="cursor: pointer;" onclick="App.showActivityDetail(${a.id}, ${hikeId})">
                                <div class="list-item-content">
                                    <div class="list-item-title">${a.name}</div>
                                    <div class="list-item-subtitle">${a.participant_count} participants</div>
                                </div>
                                <span>&rarr;</span>
                            </li>
                        `).join('')}
                    </ul>
                    `}
                </div>

                <div class="card">
                    <div class="card-header">
                        <h3>RSVP Link</h3>
                        <button class="btn btn-small ${hike.rsvp_open ? 'btn-danger' : 'btn-success'}"
                                onclick="App.toggleRSVP(${hikeId}, ${hike.rsvp_open})">
                            ${hike.rsvp_open ? 'Close RSVPs' : 'Open RSVPs'}
                        </button>
                    </div>
                    <p style="word-break: break-all; font-size: 0.875rem; color: var(--text-light); margin-bottom: 8px;">
                        ${API.getRSVPLink(hikeId)}
                    </p>
                    <button class="btn btn-secondary btn-small" onclick="App.copyRSVPLink(${hikeId})">
                        Copy Link
                    </button>
                    <span style="margin-left: 8px; font-size: 0.875rem;">
                        ${hike.rsvp_open ? '🟢 Open' : '🔴 Closed'}
                    </span>
                </div>

                ${rsvpNotCheckedIn.length > 0 ? `
                <div class="card">
                    <h3>RSVPed but not checked in (${rsvpNotCheckedIn.length})</h3>
                    <ul class="list">
                        ${rsvpNotCheckedIn.map(r => `
                            <li class="list-item" style="color: var(--error);">
                                <div class="list-item-content">
                                    <div class="list-item-title">${r.member_name || r.guest_name}</div>
                                    <div class="list-item-subtitle">${r.membership_number || 'Guest'}</div>
                                </div>
                                <button class="btn btn-small btn-primary" onclick="App.checkinRSVP(${r.id}, ${hikeId})" style="margin-left: auto;">
                                    Check In
                                </button>
                            </li>
                        `).join('')}
                    </ul>
                </div>
                ` : ''}

                <div class="card">
                    <h3>Checked in (${attendees.length}${checkedInGuests.length > 0 ? ' + ' + checkedInGuests.length + ' guests' : ''})</h3>
                    <ul class="list">
                        ${attendees.length === 0 && checkedInGuests.length === 0 ? '<li class="empty-state">No attendees yet</li>' : ''}
                        ${attendees.map(m => {
                const rsvp = rsvps.find(r => r.member_id === m.id);
                return `
                            <li class="list-item">
                                <div class="list-item-content" onclick="window.location.hash='#member-history/${m.id}'" style="cursor: pointer;">
                                    <div class="list-item-title">${m.first_name} ${m.last_name}</div>
                                    <div class="list-item-subtitle">${m.membership_number}</div>
                                </div>
                                ${rsvp ? `
                                    <button class="btn btn-small btn-secondary" onclick="App.undoCheckinRSVP(${rsvp.id}, ${hikeId})" title="Undo check-in">
                                        Undo
                                    </button>
                                ` : '<span>👋 Walk-in</span>'}
                            </li>
                        `}).join('')}
                        ${checkedInGuests.map(r => `
                            <li class="list-item">
                                <div class="list-item-content">
                                    <div class="list-item-title">${r.guest_name}</div>
                                    <div class="list-item-subtitle">Guest</div>
                                </div>
                                <button class="btn btn-small btn-secondary" onclick="App.undoCheckinRSVP(${r.id}, ${hikeId})" title="Undo check-in">
                                    Undo
                                </button>
                            </li>
                        `).join('')}
                    </ul>
                </div>
            `;
        } catch (err) {
            app.innerHTML = `<div class="card"><div class="empty-state">Failed to load attendance</div></div>`;
        }
    },

    async checkinRSVP(rsvpId, hikeId) {
        try {
            await API.checkinRSVP(rsvpId);
            this.playBeep();
            Toast.show('Checked in!', 'success');
            this.renderAttendance(hikeId);
        } catch (err) {
            Toast.show(err.message || 'Check-in failed', 'error');
        }
    },

    async undoCheckinRSVP(rsvpId, hikeId) {
        try {
            await API.undoCheckinRSVP(rsvpId);
            Toast.show('Check-in undone', 'success');
            this.renderAttendance(hikeId);
        } catch (err) {
            Toast.show(err.message || 'Undo failed', 'error');
        }
    },

    // Activity functions
    showAddActivityForm(hikeId) {
        document.getElementById('add-activity-form').style.display = 'block';
        document.getElementById('new-activity-name').focus();
    },

    hideAddActivityForm() {
        document.getElementById('add-activity-form').style.display = 'none';
        document.getElementById('new-activity-name').value = '';
    },

    async createActivity(hikeId) {
        const name = document.getElementById('new-activity-name').value.trim();
        if (!name) {
            Toast.show('Please enter an activity name', 'error');
            return;
        }

        try {
            await API.createActivity(hikeId, name);
            Toast.show('Activity created', 'success');
            this.renderAttendance(hikeId);
        } catch (err) {
            Toast.show(err.message || 'Failed to create activity', 'error');
        }
    },

    async showActivityDetail(activityId, hikeId) {
        const app = document.getElementById('app');
        app.innerHTML = '<div class="empty-state">Loading...</div>';

        try {
            const activity = await API.request('GET', `/activities/${activityId}`);
            const participants = await API.getActivityParticipants(activityId);
            const detail = await API.getHike(hikeId);
            const attendees = detail.attendees;
            const rsvps = await API.getHikeRSVPs(hikeId) || [];
            const checkedInGuests = rsvps.filter(r => r.checked_in && !r.member_id);

            // Get checkin IDs for attendees
            const checkins = await API.getHikeCheckins(hikeId);

            // Build list of all checked-in people with their checkin_id or rsvp_id
            const allCheckedIn = [
                ...attendees.map(m => {
                    const checkin = checkins.find(c => c.member_id === m.id);
                    return {
                        name: m.first_name + ' ' + m.last_name,
                        membershipNumber: m.membership_number,
                        checkinId: checkin ? checkin.id : null,
                        rsvpId: null,
                        isGuest: false
                    };
                }),
                ...checkedInGuests.map(r => ({
                    name: r.guest_name,
                    membershipNumber: '',
                    checkinId: null,
                    rsvpId: r.id,
                    isGuest: true
                }))
            ];

            // Determine who is already a participant
            const participantCheckinIds = new Set(participants.filter(p => p.checkin_id).map(p => p.checkin_id));
            const participantRsvpIds = new Set(participants.filter(p => p.rsvp_id).map(p => p.rsvp_id));

            const inActivity = allCheckedIn.filter(p =>
                (p.checkinId && participantCheckinIds.has(p.checkinId)) ||
                (p.rsvpId && participantRsvpIds.has(p.rsvpId))
            );
            const notInActivity = allCheckedIn.filter(p =>
                !(p.checkinId && participantCheckinIds.has(p.checkinId)) &&
                !(p.rsvpId && participantRsvpIds.has(p.rsvpId))
            );

            app.innerHTML = `
                <div class="card">
                    <div class="card-header">
                        <div>
                            <button class="btn btn-secondary btn-small" onclick="App.renderAttendance(${hikeId})">&larr; Back</button>
                        </div>
                        <button class="btn btn-danger btn-small" onclick="App.deleteActivity(${activityId}, ${hikeId})">Delete Activity</button>
                    </div>
                    <h2 style="margin-top: 12px;">${activity.name}</h2>
                    <p style="color: var(--text-light);">${participants.length} participants</p>
                </div>

                <div class="card">
                    <h3>In Activity (${inActivity.length})</h3>
                    ${inActivity.length === 0 ? '<p style="color: var(--text-light); font-size: 0.875rem;">No participants yet</p>' : `
                    <ul class="list">
                        ${inActivity.map(p => `
                            <li class="list-item">
                                <div class="list-item-content">
                                    <div class="list-item-title">${p.name}</div>
                                    <div class="list-item-subtitle">${p.isGuest ? 'Guest' : p.membershipNumber}</div>
                                </div>
                                <button class="btn btn-small btn-secondary" onclick="App.removeFromActivity(${activityId}, ${p.checkinId || 'null'}, ${p.rsvpId || 'null'}, ${hikeId})">
                                    Remove
                                </button>
                            </li>
                        `).join('')}
                    </ul>
                    `}
                </div>

                <div class="card">
                    <h3>Not in Activity (${notInActivity.length})</h3>
                    ${notInActivity.length === 0 ? '<p style="color: var(--text-light); font-size: 0.875rem;">Everyone is in this activity</p>' : `
                    <ul class="list">
                        ${notInActivity.map(p => `
                            <li class="list-item">
                                <div class="list-item-content">
                                    <div class="list-item-title">${p.name}</div>
                                    <div class="list-item-subtitle">${p.isGuest ? 'Guest' : p.membershipNumber}</div>
                                </div>
                                <button class="btn btn-small btn-primary" onclick="App.addToActivity(${activityId}, ${p.checkinId || 'null'}, ${p.rsvpId || 'null'}, ${hikeId})">
                                    Add
                                </button>
                            </li>
                        `).join('')}
                    </ul>
                    `}
                </div>
            `;
        } catch (err) {
            app.innerHTML = `<div class="card"><div class="empty-state">Failed to load activity</div></div>`;
            console.error(err);
        }
    },

    async addToActivity(activityId, checkinId, rsvpId, hikeId) {
        try {
            await API.addActivityParticipant(activityId, checkinId, rsvpId);
            Toast.show('Added to activity', 'success');
            this.showActivityDetail(activityId, hikeId);
        } catch (err) {
            Toast.show(err.message || 'Failed to add', 'error');
        }
    },

    async removeFromActivity(activityId, checkinId, rsvpId, hikeId) {
        try {
            await API.removeActivityParticipant(activityId, checkinId, rsvpId);
            Toast.show('Removed from activity', 'success');
            this.showActivityDetail(activityId, hikeId);
        } catch (err) {
            Toast.show(err.message || 'Failed to remove', 'error');
        }
    },

    async deleteActivity(activityId, hikeId) {
        if (!confirm('Delete this activity?')) return;

        try {
            await API.deleteActivity(activityId);
            Toast.show('Activity deleted', 'success');
            this.renderAttendance(hikeId);
        } catch (err) {
            Toast.show(err.message || 'Failed to delete', 'error');
        }
    },

    async renderMembers() {
        const app = document.getElementById('app');
        app.innerHTML = '<div class="empty-state">Loading...</div>';

        try {
            this.members = await API.getMembers();

            app.innerHTML = `
                <div class="toolbar">
                    <input type="search" id="member-search" placeholder="Search members...">
                    <div class="toolbar-buttons">
                        <button class="btn btn-secondary btn-small" onclick="document.getElementById('csv-import').click()">Import CSV</button>
                        <button class="btn btn-primary btn-small" onclick="window.location.hash='#new-member'">+ Add</button>
                    </div>
                    <input type="file" id="csv-import" accept=".csv" style="display:none">
                </div>
                <div class="card">
                    <ul class="list" id="members-list">
                        ${this.renderMembersList(this.members)}
                    </ul>
                </div>
            `;

            document.getElementById('csv-import').addEventListener('change', async (e) => {
                const file = e.target.files[0];
                if (!file) return;

                try {
                    const result = await API.importMembersCSV(file);
                    Toast.show(`Imported ${result.imported}, skipped ${result.skipped} duplicates`, 'success');
                    this.renderMembers(); // Refresh list
                } catch (err) {
                    Toast.show('Import failed: ' + err.message, 'error');
                }
                e.target.value = ''; // Reset file input
            });

            document.getElementById('member-search').addEventListener('input', (e) => {
                const query = e.target.value.toLowerCase();
                const filtered = this.members.filter(m =>
                    m.first_name.toLowerCase().includes(query) ||
                    m.last_name.toLowerCase().includes(query) ||
                    m.membership_number.toLowerCase().includes(query)
                );
                document.getElementById('members-list').innerHTML = this.renderMembersList(filtered);
            });
        } catch (err) {
            app.innerHTML = `<div class="card"><div class="empty-state">Failed to load members</div></div>`;
        }
    },

    renderMembersList(members) {
        if (members.length === 0) {
            return '<li class="empty-state">No members found</li>';
        }
        return members.map(m => `
            <li class="list-item" onclick="window.location.hash='#member/${m.id}'">
                <div class="list-item-content">
                    <div class="list-item-title">${m.first_name} ${m.last_name}</div>
                    <div class="list-item-subtitle">${m.membership_number}</div>
                </div>
            </li>
        `).join('');
    },

    async renderMemberDetail(memberId) {
        const app = document.getElementById('app');
        app.innerHTML = '<div class="empty-state">Loading...</div>';

        try {
            const member = await API.getMember(memberId);

            app.innerHTML = `
                <div class="card">
                    <h2>${member.first_name} ${member.last_name}</h2>
                    <p>${member.membership_number}</p>
                    <div class="qr-display">
                        <img src="${API.getMemberQRUrl(memberId)}" alt="QR Code">
                        <p>Scan this code for check-in</p>
                    </div>
                </div>
                <div class="card">
                    <h3>Details</h3>
                    <p><strong>Email:</strong> ${member.email || 'Not set'}</p>
                    <p><strong>Phone:</strong> ${member.phone || 'Not set'}</p>
                    <p><strong>Status:</strong> ${member.active ? 'Active' : 'Inactive'}</p>
                </div>
                <div class="stats-row">
                    <button class="btn btn-secondary btn-block" onclick="window.location.hash='#member-history/${memberId}'">
                        View Attendance History
                    </button>
                </div>
                <div class="stats-row">
                    <button class="btn btn-secondary btn-block" onclick="window.location.hash='#edit-member/${memberId}'">
                        Edit Member
                    </button>
                </div>
            `;
        } catch (err) {
            app.innerHTML = `<div class="card"><div class="empty-state">Failed to load member</div></div>`;
        }
    },

    async renderMemberHistory(memberId) {
        const app = document.getElementById('app');
        app.innerHTML = '<div class="empty-state">Loading...</div>';

        try {
            const data = await API.getMemberHistory(memberId);
            const member = data.member;
            const hikes = data.hikes || [];

            app.innerHTML = `
                <div class="card">
                    <div class="card-header">
                        <div>
                            <h2>${member.first_name} ${member.last_name}</h2>
                            <p>${member.membership_number}</p>
                        </div>
                        <a href="${API.getMemberHistoryCSVUrl(memberId)}" class="download-btn" download>
                            Download CSV
                        </a>
                    </div>
                    <div class="stat-card">
                        <div class="stat-value">${hikes.length}</div>
                        <div class="stat-label">Hikes Attended</div>
                    </div>
                </div>
                <div class="card">
                    <h3>Attendance History</h3>
                    <ul class="list">
                        ${hikes.length === 0 ? '<li class="empty-state">No hikes attended</li>' :
                    hikes.map(h => `
                                <li class="list-item" onclick="window.location.hash='#hike/${h.id}'">
                                    <div class="list-item-content">
                                        <div class="list-item-title">${h.name}</div>
                                        <div class="list-item-subtitle">${h.date} ${h.location ? '• ' + h.location : ''}</div>
                                    </div>
                                </li>
                            `).join('')
                }
                    </ul>
                </div>
            `;
        } catch (err) {
            app.innerHTML = `<div class="card"><div class="empty-state">Failed to load history</div></div>`;
        }
    },

    renderNewMember() {
        const app = document.getElementById('app');
        app.innerHTML = `
            <div class="card">
                <h2>Add New Member</h2>
                <form id="new-member-form">
                    <div class="form-group">
                        <label for="membership-number">Membership Number *</label>
                        <input type="text" id="membership-number" required placeholder="TC-001">
                    </div>
                    <div class="form-group">
                        <label for="first-name">First Name *</label>
                        <input type="text" id="first-name" required>
                    </div>
                    <div class="form-group">
                        <label for="last-name">Last Name *</label>
                        <input type="text" id="last-name" required>
                    </div>
                    <div class="form-group">
                        <label for="email">Email</label>
                        <input type="email" id="email">
                    </div>
                    <div class="form-group">
                        <label for="phone">Phone</label>
                        <input type="tel" id="phone">
                    </div>
                    <button type="submit" class="btn btn-primary btn-block">Add Member</button>
                </form>
            </div>
        `;

        document.getElementById('new-member-form').addEventListener('submit', async (e) => {
            e.preventDefault();
            try {
                const member = await API.createMember({
                    membership_number: document.getElementById('membership-number').value,
                    first_name: document.getElementById('first-name').value,
                    last_name: document.getElementById('last-name').value,
                    email: document.getElementById('email').value,
                    phone: document.getElementById('phone').value,
                });
                Toast.show('Member added!', 'success');
                window.location.hash = `#member/${member.id}`;
            } catch (err) {
                Toast.show(err.message || 'Failed to add member', 'error');
            }
        });
    },

    async renderEditMember(memberId) {
        const app = document.getElementById('app');
        app.innerHTML = '<div class="empty-state">Loading...</div>';

        try {
            const member = await API.getMember(memberId);

            app.innerHTML = `
                <div class="card">
                    <h2>Edit Member</h2>
                    <form id="edit-member-form">
                        <div class="form-group">
                            <label for="membership-number">Membership Number *</label>
                            <input type="text" id="membership-number" required value="${member.membership_number}">
                        </div>
                        <div class="form-group">
                            <label for="first-name">First Name *</label>
                            <input type="text" id="first-name" required value="${member.first_name}">
                        </div>
                        <div class="form-group">
                            <label for="last-name">Last Name *</label>
                            <input type="text" id="last-name" required value="${member.last_name}">
                        </div>
                        <div class="form-group">
                            <label for="email">Email</label>
                            <input type="email" id="email" value="${member.email || ''}">
                        </div>
                        <div class="form-group">
                            <label for="phone">Phone</label>
                            <input type="tel" id="phone" value="${member.phone || ''}">
                        </div>
                        <button type="submit" class="btn btn-primary btn-block">Save Changes</button>
                    </form>
                </div>
                <div class="card">
                    <button class="btn btn-danger btn-block" onclick="App.deleteMember(${memberId})">
                        Deactivate Member
                    </button>
                </div>
            `;

            document.getElementById('edit-member-form').addEventListener('submit', async (e) => {
                e.preventDefault();
                try {
                    await API.updateMember(memberId, {
                        membership_number: document.getElementById('membership-number').value,
                        first_name: document.getElementById('first-name').value,
                        last_name: document.getElementById('last-name').value,
                        email: document.getElementById('email').value,
                        phone: document.getElementById('phone').value,
                    });
                    Toast.show('Member updated!', 'success');
                    window.location.hash = `#member/${memberId}`;
                } catch (err) {
                    Toast.show(err.message || 'Failed to update member', 'error');
                }
            });
        } catch (err) {
            app.innerHTML = `<div class="card"><div class="empty-state">Failed to load member</div></div>`;
        }
    },

    async deleteMember(memberId) {
        if (!confirm('Deactivate this member? They will no longer appear in the active members list.')) return;

        try {
            await API.deleteMember(memberId);
            Toast.show('Member deactivated');
            window.location.hash = '#members';
        } catch (err) {
            Toast.show('Failed to deactivate member', 'error');
        }
    },

    copyRSVPLink(hikeId) {
        const link = API.getRSVPLink(hikeId);
        navigator.clipboard.writeText(link).then(() => {
            Toast.show('RSVP link copied!', 'success');
        }).catch(() => {
            Toast.show('Could not copy link', 'error');
        });
    },

    async toggleRSVP(hikeId, currentlyOpen) {
        try {
            if (currentlyOpen) {
                await API.closeRSVPs(hikeId);
                Toast.show('RSVPs closed');
            } else {
                await API.openRSVPs(hikeId);
                Toast.show('RSVPs opened');
            }
            this.renderAttendance(hikeId);
        } catch (err) {
            Toast.show('Failed to update RSVPs', 'error');
        }
    },

    playBeep() {
        try {
            if (!this.audioCtx) {
                this.audioCtx = new (window.AudioContext || window.webkitAudioContext)();
            }

            // Resume if suspended (mobile browsers)
            if (this.audioCtx.state === 'suspended') {
                this.audioCtx.resume();
            }

            const oscillator = this.audioCtx.createOscillator();
            const gainNode = this.audioCtx.createGain();

            oscillator.connect(gainNode);
            gainNode.connect(this.audioCtx.destination);

            oscillator.frequency.value = 880; // A5 note
            oscillator.type = 'sine';
            gainNode.gain.value = 0.5;

            oscillator.start();
            gainNode.gain.exponentialRampToValueAtTime(0.01, this.audioCtx.currentTime + 0.3);
            oscillator.stop(this.audioCtx.currentTime + 0.3);
        } catch (e) {
            console.log('Could not play beep:', e);
        }
    },

    async renderHikes() {
        const app = document.getElementById('app');
        app.innerHTML = '<div class="empty-state">Loading...</div>';

        try {
            const hikes = await API.getHikes();

            const currentYear = new Date().getFullYear();
            app.innerHTML = `
                <div class="toolbar">
                    <input type="search" id="hike-search" placeholder="Search hikes...">
                    <a href="${API.getFullAttendanceCSVUrl(currentYear)}" class="download-btn" download title="Full attendance for ${currentYear}">Attendance</a>
                    <a href="${API.getAllHikesCSVUrl()}" class="download-btn" download title="Hike summary">Hikes</a>
                </div>
                <div class="card">
                    <ul class="list" id="hikes-list">
                        ${hikes.length === 0 ? '<li class="empty-state">No hikes yet</li>' :
                    hikes.map(h => `
                                <li class="list-item" onclick="window.location.hash='#hike/${h.id}'">
                                    <div class="list-item-content">
                                        <div class="list-item-title">${h.name}</div>
                                        <div class="list-item-subtitle">${h.date} • ${h.rsvp_count} RSVPs • ${h.attendee_count} checked in</div>
                                    </div>
                                    <span class="list-item-badge ${h.status}">${h.status}</span>
                                </li>
                            `).join('')
                }
                    </ul>
                </div>
            `;

            document.getElementById('hike-search').addEventListener('input', (e) => {
                const query = e.target.value.toLowerCase();
                const filtered = hikes.filter(h =>
                    h.name.toLowerCase().includes(query) ||
                    h.date.includes(query) ||
                    (h.location && h.location.toLowerCase().includes(query))
                );
                document.getElementById('hikes-list').innerHTML = filtered.length === 0 ?
                    '<li class="empty-state">No hikes found</li>' :
                    filtered.map(h => `
                        <li class="list-item" onclick="window.location.hash='#hike/${h.id}'">
                            <div class="list-item-content">
                                <div class="list-item-title">${h.name}</div>
                                <div class="list-item-subtitle">${h.date} • ${h.rsvp_count} RSVPs • ${h.attendee_count} checked in</div>
                            </div>
                            <span class="list-item-badge ${h.status}">${h.status}</span>
                        </li>
                    `).join('');
            });
        } catch (err) {
            app.innerHTML = `<div class="card"><div class="empty-state">Failed to load hikes</div></div>`;
        }
    },

    async renderHikeDetail(hikeId) {
        await this.renderAttendance(hikeId);
    },
};

// Toast notifications
const Toast = {
    show(message, type = '') {
        const toast = document.getElementById('toast');
        toast.textContent = message;
        toast.className = 'toast show ' + type;
        setTimeout(() => {
            toast.className = 'toast';
        }, 3000);
    }
};

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => App.init());
