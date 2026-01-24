import { api } from '$lib/api';
type UserSettings = { llm_provider: string; llm_model?: string; api_key?: string };
let userSettings = $state<UserSettings | null>(null);
let isLoading = $state(false);
const CACHE_KEY = 'user-settings-cache';
function readCache(): UserSettings | null {
	try {
		const raw = localStorage.getItem(CACHE_KEY);
		return raw ? (JSON.parse(raw) as UserSettings) : null;
	} catch {
		return null;
	}
}
function writeCache(value: UserSettings | null) {
	try {
		if (value) localStorage.setItem(CACHE_KEY, JSON.stringify(value));
		else localStorage.removeItem(CACHE_KEY);
	} catch {}
}
export const userSettingsStore = {
	get settings() {
		return userSettings;
	},
	get isLoading() {
		return isLoading;
	}
};
export async function loadUserSettings(force = false) {
	if (isLoading && !force) return;
	isLoading = true;
	try {
		const { auth } = await import('$lib/stores/auth.svelte.ts');
		if (!auth.accessToken) {
			const cached = readCache();
			userSettings = cached;
			console.log('[USER SETTINGS] Используем кэш (нет токена):', cached);
			return;
		}
		const settings = await api.getUserSettings();
		console.log('[USER SETTINGS] Получены настройки с сервера:', settings);
		const cached = readCache();
		console.log('[USER SETTINGS] Кэш:', cached);
		const merged: UserSettings = {
			...settings,
			api_key: settings.llm_provider !== 'ego' ? cached?.api_key : ''
		};
		userSettings = merged;
		writeCache(merged);
		console.log('[USER SETTINGS] Загружены настройки:', merged);
	} catch (e) {
		console.error('[USER SETTINGS] Ошибка загрузки настроек, используем кеш:', e);
		const cached = readCache();
		userSettings = cached;
	} finally {
		isLoading = false;
	}
}
export function clearUserSettings() {
	userSettings = null;
	writeCache(null);
}
export function updateUserSettings(settings: UserSettings) {
	console.log('[USER SETTINGS] Обновляем настройки:', settings);
	const settingsToCache = settings.llm_provider === 'ego' ? { ...settings, api_key: '' } : settings;
	userSettings = settings;
	writeCache(settingsToCache);
	console.log('[USER SETTINGS] Настройки обновлены и сохранены в кэш:', settingsToCache);
}
