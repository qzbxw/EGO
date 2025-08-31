<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { loadPreferences } from '$lib/stores/preferences.svelte.ts';
  import { maintenanceStore } from '$lib/stores/maintenance-store.svelte.ts';
  import { initAuthStore, auth, setAccessToken } from '$lib/stores/auth.svelte.ts';
  import { api, ApiError } from '$lib/api';
  let authOk = true; 
  let isMaintenanceMode = false; 
  let target = '/';
  onMount(() => {
    loadPreferences();
    try {
      const params = new URLSearchParams(window.location.search);
      const r = params.get('redirect');
      if (r) target = r;
    } catch {}
    const preload = (async () => {
      const maxWaitTime = 5000; 
      const startTime = Date.now();
      while (maintenanceStore.isChecking && (Date.now() - startTime) < maxWaitTime) {
        await new Promise(resolve => setTimeout(resolve, 100));
      }
      if (maintenanceStore.isChecking) {
        console.warn('🔧 Maintenance store initialization timeout, proceeding anyway');
      }
      isMaintenanceMode = maintenanceStore.isMaintenanceActive;
      console.log('🔧 Maintenance check:', { 
        isMaintenanceActive: isMaintenanceMode, 
        hasValidBypass: maintenanceStore.hasValidBypass,
        status: maintenanceStore.status,
        isStillChecking: maintenanceStore.isChecking
      });
      if (isMaintenanceMode) {
        return;
      }
      try {
        initAuthStore();
        if (!auth.accessToken && auth.refreshToken) {
          try {
            const res = await api.post<{ access_token: string }>('/auth/refresh', {
              refresh_token: auth.refreshToken
            });
            if (res?.access_token) {
              setAccessToken(res.access_token);
            }
          } catch {
          }
        }
        const tryRefreshOnce = async () => {
          if (!auth.refreshToken) return false;
          try {
            const res = await api.post<{ access_token: string }>('/auth/refresh', {
              refresh_token: auth.refreshToken
            });
            if (res?.access_token) {
              setAccessToken(res.access_token);
              return true;
            }
          } catch {
          }
          return false;
        };
        if (!auth.accessToken) {
          authOk = false;
          return;
        }
        try {
          await api.get('/me');
          authOk = true;
        } catch (e) {
          if (e instanceof ApiError && e.status === 401 && (await tryRefreshOnce())) {
            try {
              await api.get('/me');
              authOk = true;
            } catch {
              authOk = false;
            }
          } else {
            authOk = false;
          }
        }
      } catch {
        authOk = false;
      }
    })();
    const textEl = document.querySelector('.typing-container');
    const caretEl = document.querySelector('.caret');
    if (!textEl || !caretEl) return;
    let timeouts: NodeJS.Timeout[] = [];
    const clearAllTimeouts = () => timeouts.forEach(clearTimeout);
    const type = (targetStr: string, el: Element, callback?: () => void) => {
      let i = 0;
      const typeChar = () => {
        if (i < targetStr.length) {
          el.textContent += targetStr[i];
          i++;
          timeouts.push(setTimeout(typeChar, 120));
        } else {
          if (callback) timeouts.push(setTimeout(callback, 800));
        }
      };
      typeChar();
    };
    const untype = (count: number, el: Element, callback?: () => void) => {
      let i = 0;
      const untypeChar = () => {
        if (i < count) {
          el.textContent = el.textContent?.slice(0, -1) ?? '';
          i++;
          timeouts.push(setTimeout(untypeChar, 100));
        } else {
          if (callback) timeouts.push(setTimeout(callback, 300));
        }
      };
      untypeChar();
    };
    const clearAll = (el: Element, callback?: () => void) => {
      const currentText = el.textContent || '';
      let i = 0;
      const clearChar = () => {
        if (i < currentText.length) {
          el.textContent = currentText.slice(0, currentText.length - i - 1);
          i++;
          timeouts.push(setTimeout(clearChar, 80));
        } else {
          if (callback) timeouts.push(setTimeout(callback, 300));
        }
      };
      clearChar();
    };
    textEl.textContent = ''; 
    type('Authenticating', textEl, async () => {
      await preload;
      if (isMaintenanceMode) {
        console.log('🔧 Maintenance mode detected, starting animation');
        clearAll(textEl, () => {
          type('Maintenance.', textEl, () => {
            console.log('🔧 Maintenance animation finished, redirecting to /maintenance');
            if (caretEl) (caretEl as HTMLElement).style.display = 'none';
            timeouts.push(setTimeout(() => {
              console.log('🔧 Executing redirect to /maintenance');
              goto('/maintenance', { replaceState: true });
            }, 1500));
          });
        });
      } else if (authOk) {
        console.log('🔧 Auth successful, starting "ed." animation, target:', target);
        untype(3, textEl, () => {
          type('ed.', textEl, () => {
            console.log('🔧 "Authenticated." animation finished, redirecting to:', target);
            if (caretEl) (caretEl as HTMLElement).style.display = 'none';
            timeouts.push(setTimeout(() => {
              console.log('🔧 Executing redirect to:', target);
              goto(target + (target.includes('?') ? '&' : '?') + 'from_auth=1', { replaceState: true });
            }, 900));
          });
        });
      } else {
        console.log('🔧 Auth failed — redirecting to /login immediately');
        if (caretEl) (caretEl as HTMLElement).style.display = 'none';
        goto('/login', { replaceState: true });
      }
    });
    return () => {
      clearAllTimeouts();
    };
  });
</script>
<svelte:head>
  <title>EGO Auth</title>
  <meta name="robots" content="noindex" />
</svelte:head>
<div class="splash">
  <div class="logo">EGO Auth</div>
  <div class="text">
    <span class="typing-container"></span><span class="caret"></span>
  </div>
</div>
<style>
  .splash {
    position: fixed;
    inset: 0;
    background: rgb(var(--color-primary-rgb));
    color: rgb(var(--color-text-primary-rgb));
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 14px;
    font-family: system-ui, -apple-system, Segoe UI, Roboto, Inter, Arial, sans-serif;
    text-align: left;
    padding: 0 20px;
  }
  .logo {
    font-weight: 600;
    opacity: 0.8;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .text {
    font-size: clamp(24px, 3.5vw, 36px);
    font-weight: 500;
    white-space: nowrap;
    line-height: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    width: 100%;
    max-width: 600px;
    margin: 0 auto;
  }
  .typing-container {
  }
  .caret {
    width: 2px;
    height: 1.1em;
    background-color: rgb(var(--color-text-primary-rgb));
    margin-left: 0.15em;
    animation: blink 1s infinite;
  }
  @keyframes blink {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0;
    }
  }
</style>