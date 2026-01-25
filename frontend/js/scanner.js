// QR Scanner using html5-qrcode library

const Scanner = {
    html5QrCode: null,
    onScanCallback: null,
    lastScannedCode: null,
    scanCooldown: false,

    async init(elementId, onScan) {
        this.onScanCallback = onScan;
        this.lastScannedCode = null;

        // Load html5-qrcode library if not already loaded
        if (typeof Html5Qrcode === 'undefined') {
            await this.loadScript('https://unpkg.com/html5-qrcode@2.3.8/html5-qrcode.min.js');
        }

        // Initialize scanner
        this.html5QrCode = new Html5Qrcode(elementId);

        try {
            await this.html5QrCode.start(
                { facingMode: "environment" },
                {
                    fps: 10,
                    qrbox: { width: 250, height: 250 },
                    aspectRatio: 1.0,
                },
                (decodedText) => this.handleScan(decodedText),
                (errorMessage) => {
                    // Ignore scanning errors (no QR found in frame)
                }
            );
        } catch (err) {
            console.error('Failed to start scanner:', err);
            document.getElementById(elementId).innerHTML = `
                <div style="padding: 40px; text-align: center; color: white;">
                    <p>Camera access required</p>
                    <p style="font-size: 0.875rem; opacity: 0.7;">${err.message || 'Please allow camera access'}</p>
                </div>
            `;
        }
    },

    handleScan(code) {
        // Debounce - prevent multiple scans of the same code
        if (this.scanCooldown || code === this.lastScannedCode) {
            return;
        }

        this.lastScannedCode = code;
        this.scanCooldown = true;

        // Allow same code to be scanned again after 3 seconds
        setTimeout(() => {
            this.scanCooldown = false;
        }, 2000);

        // Reset last scanned code after 5 seconds to allow re-scanning
        setTimeout(() => {
            if (this.lastScannedCode === code) {
                this.lastScannedCode = null;
            }
        }, 5000);

        if (this.onScanCallback) {
            this.onScanCallback(code);
        }
    },

    async stop() {
        if (this.html5QrCode) {
            try {
                await this.html5QrCode.stop();
            } catch (err) {
                console.error('Failed to stop scanner:', err);
            }
            this.html5QrCode = null;
        }
    },

    loadScript(src) {
        return new Promise((resolve, reject) => {
            const script = document.createElement('script');
            script.src = src;
            script.onload = resolve;
            script.onerror = reject;
            document.head.appendChild(script);
        });
    },
};

// Stop scanner when navigating away
window.addEventListener('hashchange', () => {
    if (!window.location.hash.includes('scanner')) {
        Scanner.stop();
    }
});
