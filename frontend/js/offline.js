// Offline support with IndexedDB for pending check-ins

const OfflineStore = {
    dbName: 'trailcall',
    dbVersion: 2,
    db: null,

    async init() {
        return new Promise((resolve, reject) => {
            const request = indexedDB.open(this.dbName, this.dbVersion);

            request.onerror = () => reject(request.error);
            request.onsuccess = () => {
                this.db = request.result;
                resolve();
            };

            request.onupgradeneeded = (event) => {
                const db = event.target.result;

                // Store for pending check-ins
                if (!db.objectStoreNames.contains('pendingCheckins')) {
                    const store = db.createObjectStore('pendingCheckins', { keyPath: 'id', autoIncrement: true });
                    store.createIndex('hikeId', 'hikeId', { unique: false });
                }

                // Store for cached members (for offline display)
                if (!db.objectStoreNames.contains('members')) {
                    db.createObjectStore('members', { keyPath: 'id' });
                }

                // Store for cached hike (current open hike)
                if (!db.objectStoreNames.contains('currentHike')) {
                    db.createObjectStore('currentHike', { keyPath: 'id' });
                }

                // Store for cached RSVPs
                if (!db.objectStoreNames.contains('rsvps')) {
                    const rsvpStore = db.createObjectStore('rsvps', { keyPath: 'id' });
                    rsvpStore.createIndex('hikeId', 'hikeId', { unique: false });
                }
            };
        });
    },

    // Pending check-ins
    async addPendingCheckin(hikeId, membershipNumber) {
        const tx = this.db.transaction('pendingCheckins', 'readwrite');
        const store = tx.objectStore('pendingCheckins');

        await store.add({
            hikeId,
            membershipNumber,
            timestamp: new Date().toISOString(),
        });

        return new Promise((resolve, reject) => {
            tx.oncomplete = resolve;
            tx.onerror = () => reject(tx.error);
        });
    },

    async getPendingCheckins() {
        const tx = this.db.transaction('pendingCheckins', 'readonly');
        const store = tx.objectStore('pendingCheckins');
        const request = store.getAll();

        return new Promise((resolve, reject) => {
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    },

    async clearPendingCheckins(ids) {
        const tx = this.db.transaction('pendingCheckins', 'readwrite');
        const store = tx.objectStore('pendingCheckins');

        for (const id of ids) {
            store.delete(id);
        }

        return new Promise((resolve, reject) => {
            tx.oncomplete = resolve;
            tx.onerror = () => reject(tx.error);
        });
    },

    async getPendingCount() {
        const pending = await this.getPendingCheckins();
        return pending.length;
    },

    // Member cache
    async cacheMembers(members) {
        const tx = this.db.transaction('members', 'readwrite');
        const store = tx.objectStore('members');

        // Clear existing
        store.clear();

        // Add new
        for (const member of members) {
            store.add(member);
        }

        return new Promise((resolve, reject) => {
            tx.oncomplete = resolve;
            tx.onerror = () => reject(tx.error);
        });
    },

    async getCachedMembers() {
        const tx = this.db.transaction('members', 'readonly');
        const store = tx.objectStore('members');
        const request = store.getAll();

        return new Promise((resolve, reject) => {
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    },

    async getCachedMemberByNumber(membershipNumber) {
        const members = await this.getCachedMembers();
        return members.find(m => m.membership_number === membershipNumber);
    },

    // Current hike cache
    async cacheCurrentHike(hike) {
        const tx = this.db.transaction('currentHike', 'readwrite');
        const store = tx.objectStore('currentHike');

        // Clear existing and add new
        store.clear();
        if (hike) {
            store.add(hike);
        }

        return new Promise((resolve, reject) => {
            tx.oncomplete = resolve;
            tx.onerror = () => reject(tx.error);
        });
    },

    async getCachedCurrentHike() {
        const tx = this.db.transaction('currentHike', 'readonly');
        const store = tx.objectStore('currentHike');
        const request = store.getAll();

        return new Promise((resolve, reject) => {
            request.onsuccess = () => resolve(request.result[0] || null);
            request.onerror = () => reject(request.error);
        });
    },

    async clearCurrentHike() {
        const tx = this.db.transaction('currentHike', 'readwrite');
        const store = tx.objectStore('currentHike');
        store.clear();

        return new Promise((resolve, reject) => {
            tx.oncomplete = resolve;
            tx.onerror = () => reject(tx.error);
        });
    },

    // RSVP cache
    async cacheRSVPs(hikeId, rsvps) {
        const tx = this.db.transaction('rsvps', 'readwrite');
        const store = tx.objectStore('rsvps');
        const index = store.index('hikeId');

        // Remove existing RSVPs for this hike
        const request = index.getAllKeys(hikeId);
        request.onsuccess = () => {
            const keys = request.result;
            keys.forEach(key => store.delete(key));

            // Add new RSVPs
            rsvps.forEach(rsvp => {
                store.add({
                    ...rsvp,
                    hikeId: parseInt(hikeId)
                });
            });
        };

        return new Promise((resolve, reject) => {
            tx.oncomplete = resolve;
            tx.onerror = () => reject(tx.error);
        });
    },

    async getCachedRSVPs(hikeId) {
        const tx = this.db.transaction('rsvps', 'readonly');
        const store = tx.objectStore('rsvps');
        const index = store.index('hikeId');
        const request = index.getAll(parseInt(hikeId));

        return new Promise((resolve, reject) => {
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    },

    async updateCachedRSVPStatus(rsvpId, checkedIn) {
        const tx = this.db.transaction('rsvps', 'readwrite');
        const store = tx.objectStore('rsvps');
        const request = store.get(rsvpId);

        request.onsuccess = () => {
            const rsvp = request.result;
            if (rsvp) {
                rsvp.checked_in = checkedIn;
                store.put(rsvp);
            }
        };

        return new Promise((resolve, reject) => {
            tx.oncomplete = resolve;
            tx.onerror = () => reject(tx.error);
        });
    },
};

// Sync manager
const SyncManager = {
    async sync() {
        if (!navigator.onLine) {
            console.log('Offline - sync skipped');
            return { synced: 0, failed: 0 };
        }

        const pending = await OfflineStore.getPendingCheckins();
        if (pending.length === 0) {
            return { synced: 0, failed: 0 };
        }

        console.log(`Syncing ${pending.length} pending check-ins...`);

        try {
            const checkins = pending.map(p => ({
                hike_id: p.hikeId,
                membership_number: p.membershipNumber,
            }));

            const result = await API.bulkCheckin(checkins);

            // Clear successfully synced
            const syncedIds = pending.map(p => p.id);
            await OfflineStore.clearPendingCheckins(syncedIds);

            const synced = result.checkins?.length || 0;
            const failed = result.errors?.length || 0;

            if (synced > 0) {
                Toast.show(`Synced ${synced} check-in${synced > 1 ? 's' : ''}`, 'success');
            }
            if (failed > 0) {
                console.error('Sync errors:', result.errors);
            }

            return { synced, failed };
        } catch (err) {
            console.error('Sync failed:', err);
            return { synced: 0, failed: pending.length };
        }
    },

    async refreshMemberCache() {
        if (!navigator.onLine) return;

        try {
            const members = await API.getMembers();
            await OfflineStore.cacheMembers(members);
            console.log(`Cached ${members.length} members`);
        } catch (err) {
            console.error('Failed to cache members:', err);
        }
    },

    async refreshHikeCache() {
        if (!navigator.onLine) return;

        try {
            const hike = await API.getOpenHike();
            await OfflineStore.cacheCurrentHike(hike);
            console.log('Cached current hike:', hike?.name || 'none');

            if (hike) {
                await this.refreshRSVPCache(hike.id);
            }
        } catch (err) {
            // No open hike is not an error
            await OfflineStore.clearCurrentHike();
        }
    },

    async refreshRSVPCache(hikeId) {
        if (!navigator.onLine) return;

        try {
            const rsvps = await API.getHikeRSVPs(hikeId);
            await OfflineStore.cacheRSVPs(hikeId, rsvps || []);
            console.log(`Cached ${rsvps?.length || 0} RSVPs for hike ${hikeId}`);
        } catch (err) {
            console.error('Failed to cache RSVPs:', err);
        }
    },
};

// Offline indicator
const OfflineIndicator = {
    element: null,

    init() {
        // Create indicator element
        this.element = document.createElement('div');
        this.element.id = 'offline-indicator';
        this.element.innerHTML = '📡 Offline Mode - Check-ins will sync when back online';
        this.element.style.cssText = `
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            background: #ff9800;
            color: white;
            text-align: center;
            padding: 8px;
            font-size: 0.875rem;
            font-weight: 500;
            z-index: 9999;
        `;
        document.body.prepend(this.element);

        // Check initial state
        this.update();
    },

    update() {
        if (this.element) {
            this.element.style.display = navigator.onLine ? 'none' : 'block';
            // Adjust body padding when indicator is shown
            document.body.style.paddingTop = navigator.onLine ? '0' : '36px';
        }
    },

    show() {
        if (this.element) {
            this.element.style.display = 'block';
            document.body.style.paddingTop = '36px';
        }
    },

    hide() {
        if (this.element) {
            this.element.style.display = 'none';
            document.body.style.paddingTop = '0';
        }
    }
};

// Initialize offline support
document.addEventListener('DOMContentLoaded', async () => {
    try {
        await OfflineStore.init();
        console.log('Offline store initialized');

        // Initialize offline indicator
        OfflineIndicator.init();

        // Sync and cache on load if online
        if (navigator.onLine) {
            await SyncManager.sync();
            await SyncManager.refreshMemberCache();
            await SyncManager.refreshHikeCache();
        }
    } catch (err) {
        console.error('Failed to initialize offline store:', err);
    }
});

// Sync when coming back online
window.addEventListener('online', async () => {
    console.log('Back online - syncing...');
    OfflineIndicator.hide();
    Toast.show('Back online - syncing...');
    await SyncManager.sync();
    await SyncManager.refreshMemberCache();
    await SyncManager.refreshHikeCache();
});

// Notify when going offline
window.addEventListener('offline', () => {
    OfflineIndicator.show();
    Toast.show('You are offline. Check-ins will be saved locally.', 'warning');
});
