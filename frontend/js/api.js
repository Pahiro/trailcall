// API client for TrailCall backend

const API = {
    async request(method, path, data = null) {
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
            },
            credentials: 'same-origin',
        };

        if (data) {
            options.body = JSON.stringify(data);
        }

        const response = await fetch(`/api${path}`, options);

        if (response.status === 401) {
            // Unauthorized - redirect to login
            window.location.hash = '#login';
            throw new Error('Unauthorized');
        }

        if (response.status === 204) {
            return null;
        }

        const json = await response.json();

        if (!response.ok) {
            throw new Error(json.error || response.statusText);
        }

        return json;
    },

    // Auth
    async login(pin) {
        return this.request('POST', '/auth', { pin });
    },

    async logout() {
        return this.request('POST', '/auth/logout');
    },

    async checkAuth() {
        return this.request('GET', '/auth/check');
    },

    // Members
    async getMembers(activeOnly = true) {
        const query = activeOnly ? '' : '?active=false';
        return this.request('GET', `/members${query}`);
    },

    async getMember(id) {
        return this.request('GET', `/members/${id}`);
    },

    async createMember(data) {
        return this.request('POST', '/members', data);
    },

    async updateMember(id, data) {
        return this.request('PUT', `/members/${id}`, data);
    },

    async deleteMember(id) {
        return this.request('DELETE', `/members/${id}`);
    },

    getMemberQRUrl(id) {
        return `/api/members/${id}/qr`;
    },

    async importMembersCSV(file) {
        const formData = new FormData();
        formData.append('file', file);

        const response = await fetch('/api/members/import', {
            method: 'POST',
            credentials: 'same-origin',
            body: formData,
        });

        if (response.status === 401) {
            window.location.hash = '#login';
            throw new Error('Unauthorized');
        }

        return response.json();
    },

    // Hikes
    async getHikes() {
        return this.request('GET', '/hikes');
    },

    async getHike(id) {
        return this.request('GET', `/hikes/${id}`);
    },

    async getOpenHike() {
        return this.request('GET', '/hikes/open');
    },

    async createHike(data) {
        return this.request('POST', '/hikes', data);
    },

    async updateHike(id, data) {
        return this.request('PUT', `/hikes/${id}`, data);
    },

    async closeHike(id) {
        return this.request('POST', `/hikes/${id}/close`);
    },

    async getHikeCheckins(id) {
        return this.request('GET', `/hikes/${id}/checkins`);
    },

    // Check-ins
    async createCheckin(hikeId, membershipNumber) {
        return this.request('POST', '/checkins', {
            hike_id: hikeId,
            membership_number: membershipNumber,
        });
    },

    async bulkCheckin(checkins) {
        return this.request('POST', '/checkins/bulk', { checkins });
    },

    // Reports
    async getMemberHistory(id) {
        return this.request('GET', `/reports/member/${id}`);
    },

    getMemberHistoryCSVUrl(id) {
        return `/api/reports/member/${id}/csv`;
    },

    getHikeCSVUrl(id) {
        return `/api/reports/hike/${id}/csv`;
    },

    getAllHikesCSVUrl() {
        return '/api/reports/hikes/csv';
    },

    getFullAttendanceCSVUrl(year) {
        const y = year || new Date().getFullYear();
        return `/api/reports/attendance?year=${y}`;
    },

    // RSVPs
    async getHikeRSVPs(hikeId) {
        return this.request('GET', `/hikes/${hikeId}/rsvps`);
    },

    async closeRSVPs(hikeId) {
        return this.request('POST', `/hikes/${hikeId}/rsvps/close`);
    },

    async openRSVPs(hikeId) {
        return this.request('POST', `/hikes/${hikeId}/rsvps/open`);
    },

    getRSVPLink(hikeId) {
        return `${window.location.origin}/rsvp/${hikeId}`;
    },

    async checkinRSVP(rsvpId) {
        return this.request('POST', `/rsvps/${rsvpId}/checkin`);
    },

    async undoCheckinRSVP(rsvpId) {
        return this.request('POST', `/rsvps/${rsvpId}/undo`);
    },

    // Activities
    async getActivities(hikeId) {
        return this.request('GET', `/hikes/${hikeId}/activities`);
    },

    async createActivity(hikeId, name) {
        return this.request('POST', `/hikes/${hikeId}/activities`, { name });
    },

    async deleteActivity(activityId) {
        return this.request('DELETE', `/activities/${activityId}`);
    },

    async getActivityParticipants(activityId) {
        return this.request('GET', `/activities/${activityId}/participants`);
    },

    async addActivityParticipant(activityId, checkinId, rsvpId) {
        return this.request('POST', `/activities/${activityId}/participants`, {
            checkin_id: checkinId,
            rsvp_id: rsvpId
        });
    },

    async removeActivityParticipant(activityId, checkinId, rsvpId) {
        return this.request('DELETE', `/activities/${activityId}/participants`, {
            checkin_id: checkinId,
            rsvp_id: rsvpId
        });
    },
};
