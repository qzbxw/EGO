import { browser } from '$app/environment';
import {
	checkMaintenanceStatus,
	getBypassToken,
	storeBypassToken,
	getStoredBypassToken,
	validateBypassToken,
	clearBypassToken,
	type MaintenanceStatus
} from '$lib/api/maintenance';
class MaintenanceStore {
	private _status = $state<MaintenanceStatus>({ maintenance: false, isChatOnly: false, message: '' });
	private _isChecking = $state(false);
	private _bypassToken = $state<string | null>(null);
	private checkInterval: number | null = null;
	constructor() {
		if (browser) {
			this.initialize();
		}
	}
	get status() {
		return this._status;
	}
	get isChecking() {
		return this._isChecking;
	}
	get bypassToken() {
		return this._bypassToken;
	}
	get isMaintenanceActive() {
		// Only trigger full maintenance redirect if NOT chatOnly
		return this._status.maintenance && !this._status.isChatOnly && !this._bypassToken;
	}
	get isChatMaintenanceActive() {
		// Chat maintenance is active if either full maintenance OR specific chat maintenance is on
		return this._status.maintenance && !this._bypassToken;
	}
	get hasValidBypass() {
		return !!this._bypassToken;
	}
	private async initialize() {
		const urlToken = getBypassToken();
		const storedToken = getStoredBypassToken();
		console.log('🔧 Maintenance init:', {
			urlToken: urlToken ? `${urlToken.substring(0, 20)}...` : null,
			storedToken: storedToken ? `${storedToken.substring(0, 20)}...` : null,
			url: window.location.href
		});
		if (urlToken) {
			console.log('🔧 Testing URL token...');
			const isValid = await validateBypassToken(urlToken);
			console.log('🔧 URL token valid:', isValid);
			if (isValid) {
				this._bypassToken = urlToken;
				storeBypassToken(urlToken);
				console.log('🔧 URL token stored successfully');
			} else {
				clearBypassToken();
				console.log('🔧 URL token invalid, cleared');
			}
			const url = new URL(window.location.href);
			url.searchParams.delete('bypass_token');
			url.searchParams.delete('token');
			window.history.replaceState({}, '', url.toString());
		} else if (storedToken) {
			console.log('🔧 Testing stored token...');
			const isValid = await validateBypassToken(storedToken);
			console.log('🔧 Stored token valid:', isValid);
			if (isValid) {
				this._bypassToken = storedToken;
			} else {
				this.clearBypass();
				console.log('🔧 Stored token invalid, cleared');
			}
		}
		await this.checkStatus();
		this.startPeriodicCheck();
	}
	async checkStatus() {
		// Prevent concurrent or duplicate checks
		if (this._isChecking) {
			console.log('🔧 Maintenance check already in progress, skipping duplicate request');
			return;
		}
		this._isChecking = true;
		try {
			const status = await checkMaintenanceStatus();
			this._status = status;
			if (!status.maintenance && this._bypassToken) {
				this.clearBypass();
			}
			if (status.maintenance && this._bypassToken) {
				const isValid = await validateBypassToken(this._bypassToken);
				if (!isValid) {
					this.clearBypass();
				}
			}
		} catch (error) {
			console.error('Failed to check maintenance status:', error);
		} finally {
			this._isChecking = false;
		}
	}
	private startPeriodicCheck() {
		if (this.checkInterval) {
			clearInterval(this.checkInterval);
		}
		this.checkInterval = setInterval(async () => {
			await this.checkStatus();
		}, 30000);
	}
	async setBypassToken(token: string) {
		const isValid = await validateBypassToken(token);
		if (isValid) {
			this._bypassToken = token;
			storeBypassToken(token);
			return true;
		} else {
			return false;
		}
	}
	clearBypass() {
		this._bypassToken = null;
		if (browser) {
			clearBypassToken();
		}
	}
	destroy() {
		if (this.checkInterval) {
			clearInterval(this.checkInterval);
			this.checkInterval = null;
		}
	}
}
export const maintenanceStore = new MaintenanceStore();
