import { memoryStore } from './memory.svelte.ts';
let theme = $state<'dark' | 'light'>('dark');
let backgroundSpheres = $state<boolean>(true);
function persist() {
  try {
    localStorage.setItem('pref-theme', theme);
    localStorage.setItem('pref-background-spheres', backgroundSpheres ? '1' : '0');
    localStorage.setItem('pref-memory-enabled', memoryStore.memoryEnabled ? '1' : '0');
  } catch {}
}
export const preferencesStore = {
  get theme() { return theme; },
  get backgroundSpheres() { return backgroundSpheres; },
  get memoryEnabled() { return memoryStore.memoryEnabled; },
};
export function loadPreferences() {
  try {
    const t = localStorage.getItem('pref-theme');
    if (t === 'light' || t === 'dark') theme = t;
    const b = localStorage.getItem('pref-background-spheres');
    if (b === '0' || b === '1') backgroundSpheres = b === '1';
    const m = localStorage.getItem('pref-memory-enabled');
    if (m === '0' || m === '1') memoryStore.setMemoryEnabled(m === '1');
    try {
      const root = document.documentElement;
      if (theme === 'light') {
        root.classList.add('theme-light');
      } else {
        root.classList.remove('theme-light');
      }
    } catch {}
  } catch {}
}
export function setTheme(next: 'dark' | 'light') {
  theme = next;
  persist();
  try {
    const root = document.documentElement;
    if (next === 'light') {
      root.classList.add('theme-light');
    } else {
      root.classList.remove('theme-light');
    }
  } catch {}
}
export function setBackgroundSpheres(enabled: boolean) {
  backgroundSpheres = enabled;
  persist();
}
export function setMemoryEnabled(enabled: boolean) {
  memoryStore.setMemoryEnabled(enabled);
  persist();
}