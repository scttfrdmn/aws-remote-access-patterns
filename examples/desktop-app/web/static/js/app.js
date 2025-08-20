// AWS Desktop App JavaScript
class AWSDesktopApp {
    constructor() {
        this.currentPage = 'dashboard';
        this.authStatus = null;
        this.resources = {
            s3: { buckets: [], loading: false },
            ec2: { instances: [], loading: false }
        };
        this.config = {
            theme: 'auto',
            autoRefresh: true,
            refreshInterval: 30,
            notifications: true
        };
        this.refreshTimer = null;
        
        this.init();
    }

    init() {
        this.bindEvents();
        this.initTheme();
        this.loadAuthStatus();
        this.startAutoRefresh();
        
        console.log('AWS Desktop App initialized');
    }

    bindEvents() {
        // Navigation
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const page = link.dataset.page;
                if (page) this.showPage(page);
            });
        });

        // Theme toggle
        const themeToggle = document.getElementById('theme-toggle');
        if (themeToggle) {
            themeToggle.addEventListener('click', () => this.toggleTheme());
        }

        // Refresh button
        const refreshBtn = document.getElementById('refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.refreshAll());
        }

        // Authentication buttons
        this.bindAuthenticationEvents();
        
        // Settings events
        this.bindSettingsEvents();
        
        // Modal events
        this.bindModalEvents();
        
        // Resource card clicks
        document.querySelectorAll('.resource-card').forEach(card => {
            card.addEventListener('click', () => {
                const type = card.dataset.type;
                if (type) this.showPage(type);
            });
        });
    }

    bindAuthenticationEvents() {
        const setupBtn = document.getElementById('setup-auth-btn');
        const testBtn = document.getElementById('test-auth-btn');
        const refreshAuthBtn = document.getElementById('refresh-auth-btn');
        const clearBtn = document.getElementById('clear-auth-btn');

        if (setupBtn) {
            setupBtn.addEventListener('click', () => this.showSetupModal());
        }

        if (testBtn) {
            testBtn.addEventListener('click', () => this.testAuthentication());
        }

        if (refreshAuthBtn) {
            refreshAuthBtn.addEventListener('click', () => this.refreshAuthentication());
        }

        if (clearBtn) {
            clearBtn.addEventListener('click', () => this.clearAuthentication());
        }

        // Setup form
        const setupForm = document.getElementById('setup-form');
        if (setupForm) {
            setupForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.handleSetupForm();
            });
        }

        // Setup method change
        const setupMethod = document.getElementById('setup-method');
        if (setupMethod) {
            setupMethod.addEventListener('change', () => this.toggleSetupFields());
        }
    }

    bindSettingsEvents() {
        const saveAuthBtn = document.getElementById('save-auth-settings-btn');
        const saveUIBtn = document.getElementById('save-ui-settings-btn');
        const themeSelect = document.getElementById('theme-select');

        if (saveAuthBtn) {
            saveAuthBtn.addEventListener('click', () => this.saveAuthSettings());
        }

        if (saveUIBtn) {
            saveUIBtn.addEventListener('click', () => this.saveUISettings());
        }

        if (themeSelect) {
            themeSelect.addEventListener('change', (e) => {
                this.setTheme(e.target.value);
            });
        }

        // Auto refresh toggle
        const autoRefreshToggle = document.getElementById('auto-refresh-toggle');
        if (autoRefreshToggle) {
            autoRefreshToggle.addEventListener('change', (e) => {
                this.config.autoRefresh = e.target.checked;
                if (e.target.checked) {
                    this.startAutoRefresh();
                } else {
                    this.stopAutoRefresh();
                }
            });
        }
    }

    bindModalEvents() {
        // Close modal buttons
        document.querySelectorAll('[data-modal-close], .modal-close').forEach(btn => {
            btn.addEventListener('click', () => this.closeModal());
        });

        // Click outside modal to close
        document.querySelectorAll('.modal-overlay').forEach(overlay => {
            overlay.addEventListener('click', () => this.closeModal());
        });

        // Escape key to close modal
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') this.closeModal();
        });
    }

    // Navigation
    showPage(pageName) {
        // Update navigation
        document.querySelectorAll('.nav-link').forEach(link => {
            link.classList.remove('active');
        });
        
        const activeLink = document.querySelector(`[data-page="${pageName}"]`);
        if (activeLink) {
            activeLink.classList.add('active');
        }

        // Show page
        document.querySelectorAll('.page').forEach(page => {
            page.classList.remove('active');
        });

        const targetPage = document.getElementById(`page-${pageName}`);
        if (targetPage) {
            targetPage.classList.add('active');
            this.currentPage = pageName;

            // Load page-specific data
            this.loadPageData(pageName);
        }
    }

    async loadPageData(pageName) {
        switch (pageName) {
            case 'dashboard':
                await this.loadDashboardData();
                break;
            case 's3':
                await this.loadS3Data();
                break;
            case 'ec2':
                await this.loadEC2Data();
                break;
            case 'settings':
                this.loadSettingsData();
                break;
        }
    }

    // Theme Management
    initTheme() {
        const savedTheme = localStorage.getItem('aws-app-theme') || 'auto';
        this.setTheme(savedTheme);
    }

    setTheme(theme) {
        this.config.theme = theme;
        localStorage.setItem('aws-app-theme', theme);

        const html = document.documentElement;
        html.classList.remove('dark');

        if (theme === 'dark') {
            html.classList.add('dark');
        } else if (theme === 'auto') {
            if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
                html.classList.add('dark');
            }
        }

        // Update theme select
        const themeSelect = document.getElementById('theme-select');
        if (themeSelect && themeSelect.value !== theme) {
            themeSelect.value = theme;
        }

        this.onThemeChange(theme);
    }

    toggleTheme() {
        const currentTheme = this.config.theme;
        const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        this.setTheme(newTheme);
    }

    onThemeChange(theme) {
        // Notify theme change listeners
        console.log('Theme changed to:', theme);
    }

    // Authentication Management
    async loadAuthStatus() {
        try {
            const response = await fetch('/api/auth/status');
            if (response.ok) {
                this.authStatus = await response.json();
                this.updateAuthUI();
            } else {
                throw new Error(`HTTP ${response.status}`);
            }
        } catch (error) {
            console.error('Failed to load auth status:', error);
            this.showNotification('Failed to load authentication status', 'error');
        }
    }

    updateAuthUI() {
        const authConfigured = document.getElementById('auth-configured');
        const authNotConfigured = document.getElementById('auth-not-configured');
        const authStatus = document.getElementById('auth-status');
        const resourceCards = document.getElementById('resource-cards');

        if (!this.authStatus) return;

        if (this.authStatus.configured) {
            authConfigured.classList.remove('hidden');
            authNotConfigured.classList.add('hidden');
            resourceCards.classList.remove('hidden');

            // Update auth status badge
            const badge = authConfigured.querySelector('.auth-status-badge');
            if (badge) {
                badge.className = `auth-status-badge ${this.authStatus.active ? 'active' : 'inactive'}`;
                badge.textContent = this.authStatus.active ? 'Active' : 'Inactive';
            }

            // Update auth details
            this.updateElement('.auth-method', this.authStatus.method || 'Unknown');
            this.updateElement('.auth-region', this.authStatus.region || 'Unknown');
            this.updateElement('.auth-account', this.authStatus.identity?.account || 'Unknown');
        } else {
            authConfigured.classList.add('hidden');
            authNotConfigured.classList.remove('hidden');
            resourceCards.classList.add('hidden');
        }

        // Update header auth status
        if (authStatus) {
            const indicator = this.authStatus.active ? 'auth-active' : 
                             this.authStatus.configured ? 'auth-inactive' : 'auth-loading';
            
            const icon = this.authStatus.active ? 'fas fa-shield-alt text-green-500' :
                        this.authStatus.configured ? 'fas fa-shield-alt text-red-500' :
                        'fas fa-spinner fa-spin text-yellow-500';
                        
            const text = this.authStatus.active ? 'Connected' :
                        this.authStatus.configured ? 'Not Connected' : 'Loading...';

            authStatus.innerHTML = `
                <div class="auth-indicator ${indicator}">
                    <i class="${icon}"></i>
                    <span class="text-sm text-gray-600 dark:text-gray-300">${text}</span>
                </div>
            `;
        }
    }

    showSetupModal() {
        const modal = document.getElementById('setup-modal');
        if (modal) {
            modal.classList.add('show');
        }
    }

    closeModal() {
        document.querySelectorAll('.modal').forEach(modal => {
            modal.classList.remove('show');
        });
    }

    toggleSetupFields() {
        const method = document.getElementById('setup-method').value;
        const ssoFields = document.getElementById('sso-fields');
        const profileFields = document.getElementById('profile-fields');

        ssoFields.classList.add('hidden');
        profileFields.classList.add('hidden');

        switch (method) {
            case 'sso':
                ssoFields.classList.remove('hidden');
                break;
            case 'profile':
                profileFields.classList.remove('hidden');
                break;
        }
    }

    async handleSetupForm() {
        const form = document.getElementById('setup-form');
        const formData = new FormData(form);
        
        const setupRequest = {
            method: document.getElementById('setup-method').value,
            region: document.getElementById('setup-region').value
        };

        // Add method-specific fields
        switch (setupRequest.method) {
            case 'sso':
                setupRequest.start_url = document.getElementById('sso-start-url').value;
                break;
            case 'profile':
                setupRequest.profile_name = document.getElementById('profile-name').value;
                break;
        }

        try {
            const response = await fetch('/api/auth/setup', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(setupRequest)
            });

            if (response.ok) {
                this.showNotification('Authentication setup successful', 'success');
                this.closeModal();
                await this.loadAuthStatus();
            } else {
                const error = await response.text();
                throw new Error(error);
            }
        } catch (error) {
            console.error('Setup failed:', error);
            this.showNotification('Setup failed: ' + error.message, 'error');
        }
    }

    async testAuthentication() {
        try {
            const response = await fetch('/api/auth/test', { method: 'POST' });
            if (response.ok) {
                this.showNotification('Authentication test successful', 'success');
                await this.loadAuthStatus();
            } else {
                throw new Error('Authentication test failed');
            }
        } catch (error) {
            console.error('Test failed:', error);
            this.showNotification('Authentication test failed', 'error');
        }
    }

    async refreshAuthentication() {
        try {
            const response = await fetch('/api/auth/refresh', { method: 'POST' });
            if (response.ok) {
                this.showNotification('Credentials refreshed successfully', 'success');
                await this.loadAuthStatus();
            } else {
                throw new Error('Failed to refresh credentials');
            }
        } catch (error) {
            console.error('Refresh failed:', error);
            this.showNotification('Failed to refresh credentials', 'error');
        }
    }

    async clearAuthentication() {
        if (confirm('Are you sure you want to clear the authentication configuration?')) {
            try {
                const response = await fetch('/api/auth/clear', { method: 'POST' });
                if (response.ok) {
                    this.showNotification('Authentication cleared', 'success');
                    await this.loadAuthStatus();
                } else {
                    throw new Error('Failed to clear authentication');
                }
            } catch (error) {
                console.error('Clear failed:', error);
                this.showNotification('Failed to clear authentication', 'error');
            }
        }
    }

    // Data Loading
    async loadDashboardData() {
        if (this.authStatus && this.authStatus.active) {
            await Promise.all([
                this.loadS3Count(),
                this.loadEC2Count()
            ]);
        }
    }

    async loadS3Count() {
        try {
            const response = await fetch('/api/s3/buckets');
            if (response.ok) {
                const buckets = await response.json();
                this.updateResourceCard('s3', buckets.length);
            }
        } catch (error) {
            console.error('Failed to load S3 count:', error);
        }
    }

    async loadEC2Count() {
        try {
            const response = await fetch('/api/ec2/instances');
            if (response.ok) {
                const instances = await response.json();
                this.updateResourceCard('ec2', instances.length);
            }
        } catch (error) {
            console.error('Failed to load EC2 count:', error);
        }
    }

    async loadS3Data() {
        if (!this.authStatus || !this.authStatus.active) return;

        this.resources.s3.loading = true;
        this.updateS3UI();

        try {
            const response = await fetch('/api/s3/buckets');
            if (response.ok) {
                this.resources.s3.buckets = await response.json();
            } else {
                throw new Error('Failed to load S3 buckets');
            }
        } catch (error) {
            console.error('Failed to load S3 data:', error);
            this.showNotification('Failed to load S3 buckets', 'error');
        } finally {
            this.resources.s3.loading = false;
            this.updateS3UI();
        }
    }

    async loadEC2Data() {
        if (!this.authStatus || !this.authStatus.active) return;

        this.resources.ec2.loading = true;
        this.updateEC2UI();

        try {
            const response = await fetch('/api/ec2/instances');
            if (response.ok) {
                this.resources.ec2.instances = await response.json();
            } else {
                throw new Error('Failed to load EC2 instances');
            }
        } catch (error) {
            console.error('Failed to load EC2 data:', error);
            this.showNotification('Failed to load EC2 instances', 'error');
        } finally {
            this.resources.ec2.loading = false;
            this.updateEC2UI();
        }
    }

    loadSettingsData() {
        // Load current settings into UI
        const themeSelect = document.getElementById('theme-select');
        const autoRefreshToggle = document.getElementById('auto-refresh-toggle');
        const notificationsToggle = document.getElementById('notifications-toggle');

        if (themeSelect) themeSelect.value = this.config.theme;
        if (autoRefreshToggle) autoRefreshToggle.checked = this.config.autoRefresh;
        if (notificationsToggle) notificationsToggle.checked = this.config.notifications;
    }

    // UI Updates
    updateResourceCard(type, count) {
        const card = document.querySelector(`.resource-card[data-type="${type}"]`);
        if (card) {
            const countElement = card.querySelector('.resource-count');
            const valueElement = card.querySelector('.resource-value');
            
            if (countElement) countElement.textContent = count;
            if (valueElement) valueElement.textContent = count;
        }
    }

    updateS3UI() {
        const listElement = document.getElementById('s3-buckets-list');
        if (!listElement) return;

        if (this.resources.s3.loading) {
            listElement.innerHTML = `
                <div class="loading-spinner text-center py-8">
                    <i class="fas fa-spinner fa-spin text-3xl text-gray-400"></i>
                    <p class="text-gray-500 mt-2">Loading buckets...</p>
                </div>
            `;
            return;
        }

        if (this.resources.s3.buckets.length === 0) {
            listElement.innerHTML = `
                <div class="text-center py-8">
                    <i class="fas fa-archive text-4xl text-gray-400 mb-4"></i>
                    <p class="text-gray-500">No S3 buckets found</p>
                </div>
            `;
            return;
        }

        const bucketsHtml = this.resources.s3.buckets.map(bucket => `
            <div class="border border-gray-200 dark:border-gray-700 rounded-lg p-4 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors">
                <div class="flex items-center justify-between">
                    <div>
                        <h3 class="font-semibold text-gray-900 dark:text-white">${bucket.Name}</h3>
                        <p class="text-sm text-gray-500 dark:text-gray-400">
                            Created: ${new Date(bucket.CreationDate).toLocaleDateString()}
                        </p>
                    </div>
                    <div class="flex space-x-2">
                        <button class="btn btn-secondary btn-sm">
                            <i class="fas fa-eye mr-1"></i>Browse
                        </button>
                        <button class="btn btn-secondary btn-sm">
                            <i class="fas fa-cog mr-1"></i>Settings
                        </button>
                    </div>
                </div>
            </div>
        `).join('');

        listElement.innerHTML = bucketsHtml;
    }

    updateEC2UI() {
        const listElement = document.getElementById('ec2-instances-list');
        if (!listElement) return;

        if (this.resources.ec2.loading) {
            listElement.innerHTML = `
                <div class="loading-spinner text-center py-8">
                    <i class="fas fa-spinner fa-spin text-3xl text-gray-400"></i>
                    <p class="text-gray-500 mt-2">Loading instances...</p>
                </div>
            `;
            return;
        }

        if (this.resources.ec2.instances.length === 0) {
            listElement.innerHTML = `
                <div class="text-center py-8">
                    <i class="fas fa-server text-4xl text-gray-400 mb-4"></i>
                    <p class="text-gray-500">No EC2 instances found</p>
                </div>
            `;
            return;
        }

        const instancesHtml = this.resources.ec2.instances.map(instance => {
            const statusDot = this.getStatusDot(instance.State.Name);
            return `
                <div class="border border-gray-200 dark:border-gray-700 rounded-lg p-4 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors">
                    <div class="flex items-center justify-between">
                        <div class="flex-1">
                            <div class="flex items-center mb-2">
                                <span class="status-dot ${statusDot}"></span>
                                <h3 class="font-semibold text-gray-900 dark:text-white">${instance.InstanceId}</h3>
                                <span class="ml-2 badge badge-primary">${instance.InstanceType}</span>
                            </div>
                            <p class="text-sm text-gray-500 dark:text-gray-400 mb-1">
                                State: ${instance.State.Name} | AZ: ${instance.Placement.AvailabilityZone}
                            </p>
                            ${instance.PublicIpAddress ? `<p class="text-sm text-gray-500 dark:text-gray-400">IP: ${instance.PublicIpAddress}</p>` : ''}
                        </div>
                        <div class="flex space-x-2">
                            <button class="btn btn-secondary btn-sm">
                                <i class="fas fa-eye mr-1"></i>Details
                            </button>
                            <button class="btn btn-secondary btn-sm">
                                <i class="fas fa-terminal mr-1"></i>Connect
                            </button>
                        </div>
                    </div>
                </div>
            `;
        }).join('');

        listElement.innerHTML = instancesHtml;
    }

    getStatusDot(state) {
        switch (state.toLowerCase()) {
            case 'running': return 'running';
            case 'stopped': return 'stopped';
            case 'stopping': return 'stopping';
            case 'pending': return 'pending';
            default: return 'pending';
        }
    }

    updateElement(selector, text) {
        const element = document.querySelector(selector);
        if (element) element.textContent = text;
    }

    // Settings
    async saveAuthSettings() {
        // Auth settings are handled by individual setup flows
        this.showNotification('Authentication settings saved', 'success');
    }

    async saveUISettings() {
        try {
            const settings = {
                theme: document.getElementById('theme-select').value,
                auto_refresh: document.getElementById('auto-refresh-toggle').checked,
                notifications: document.getElementById('notifications-toggle').checked
            };

            this.config = { ...this.config, ...settings };
            
            // Apply settings
            this.setTheme(settings.theme);
            if (settings.auto_refresh) {
                this.startAutoRefresh();
            } else {
                this.stopAutoRefresh();
            }

            this.showNotification('UI settings saved', 'success');
        } catch (error) {
            console.error('Failed to save settings:', error);
            this.showNotification('Failed to save settings', 'error');
        }
    }

    // Auto Refresh
    startAutoRefresh() {
        if (this.config.autoRefresh) {
            this.stopAutoRefresh();
            this.refreshTimer = setInterval(() => {
                if (this.authStatus && this.authStatus.active) {
                    this.refreshCurrentPage();
                }
            }, this.config.refreshInterval * 1000);
        }
    }

    stopAutoRefresh() {
        if (this.refreshTimer) {
            clearInterval(this.refreshTimer);
            this.refreshTimer = null;
        }
    }

    async refreshAll() {
        await this.loadAuthStatus();
        await this.refreshCurrentPage();
        this.showNotification('Data refreshed', 'success');
    }

    async refreshCurrentPage() {
        await this.loadPageData(this.currentPage);
    }

    // Notifications
    showNotification(message, type = 'info') {
        if (!this.config.notifications) return;

        const container = document.getElementById('toast-container');
        if (!container) return;

        const toast = document.createElement('div');
        toast.className = `toast toast-${type} hide`;
        toast.innerHTML = `
            <div class="toast-header">
                <div class="flex items-center space-x-2">
                    <i class="${this.getNotificationIcon(type)}"></i>
                    <strong class="text-sm font-medium">${this.getNotificationTitle(type)}</strong>
                </div>
                <button class="toast-close" onclick="this.parentElement.parentElement.parentElement.remove()">
                    <i class="fas fa-times text-gray-400"></i>
                </button>
            </div>
            <div class="toast-body">
                <p class="text-sm text-gray-700 dark:text-gray-300">${message}</p>
            </div>
        `;

        container.appendChild(toast);

        // Show toast
        setTimeout(() => toast.classList.remove('hide'), 100);
        setTimeout(() => toast.classList.add('show'), 150);

        // Auto remove
        setTimeout(() => {
            toast.classList.remove('show');
            toast.classList.add('hide');
            setTimeout(() => toast.remove(), 300);
        }, 5000);
    }

    getNotificationIcon(type) {
        switch (type) {
            case 'success': return 'fas fa-check-circle text-green-500';
            case 'error': return 'fas fa-exclamation-circle text-red-500';
            case 'warning': return 'fas fa-exclamation-triangle text-yellow-500';
            default: return 'fas fa-info-circle text-blue-500';
        }
    }

    getNotificationTitle(type) {
        switch (type) {
            case 'success': return 'Success';
            case 'error': return 'Error';
            case 'warning': return 'Warning';
            default: return 'Info';
        }
    }

    // API Helpers
    getAuthStatus() {
        return this.authStatus;
    }

    async refreshResources() {
        await this.refreshCurrentPage();
    }

    async getS3Buckets() {
        const response = await fetch('/api/s3/buckets');
        return response.ok ? await response.json() : [];
    }

    getTheme() {
        return this.config.theme;
    }
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.awsApp = new AWSDesktopApp();
});

// Make app globally available
window.AWSDesktopApp = AWSDesktopApp;