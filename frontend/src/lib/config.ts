import type { ThemeName, ChatMode } from '$lib/types';
export type { ThemeName, ChatMode };
export function getAppLogo(theme: ThemeName): string {
  return theme === 'light' ? '/logolight.png' : '/logodark.png';
}
export function getModeLogo(mode: ChatMode, theme: ThemeName): string {
  const suffix = theme === 'light' ? 'light' : 'dark';
  if (mode === 'deeper') return `/deepermodelogo${suffix}.png`;
  if (mode === 'research') return `/researchmodelogo${suffix}.png`;
  return `/agentmodelogo${suffix}.png`;
}