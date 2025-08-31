import type { LayoutServerLoad } from './$types';
import { redirect, isRedirect } from '@sveltejs/kit';
export const load: LayoutServerLoad = async ({ url, fetch, cookies }) => {
    const data = { initialLocale: 'ru' as const };
    if (url.pathname === '/maintenance') {
        return data;
    }
    try {
        const cookieToken = cookies.get('maintenance_bypass_token') || null;
        const urlToken = url.searchParams.get('bypass_token') || url.searchParams.get('token') || null;
        const tokenToSend = cookieToken || urlToken;
        const headers: Record<string, string> = { 'Content-Type': 'application/json' };
        if (tokenToSend) headers['X-Bypass-Token'] = tokenToSend;
        const resp = await fetch('/api/maintenance/status', { method: 'GET', headers });
        let isMaintenance = false;
        if (resp.ok) {
            try {
                const j = await resp.json();
                isMaintenance = !!j?.maintenance;
            } catch {
            }
        } else if (resp.status === 503 || resp.status === 502) {
            isMaintenance = true;
        }
        const isMaintenancePage = url.pathname === '/maintenance';
        const isPublicLegal = url.pathname === '/terms' || url.pathname === '/privacy';
        if (isMaintenance) {
            const hasBypass = !!tokenToSend;
            if (!hasBypass && !isMaintenancePage && !isPublicLegal) {
                const target = urlToken ? `/maintenance?bypass_token=${encodeURIComponent(urlToken)}` : '/maintenance';
                throw redirect(307, target);
            }
        } else if (isMaintenancePage) {
            throw redirect(307, '/');
        }
    } catch (e: unknown) {
        if (isRedirect(e)) {
            throw e;
        }
    }
    return data;
};