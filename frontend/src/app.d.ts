import type { KatexOptions } from 'katex';
declare global {
	namespace App {}
	interface Window {
		google: {
			accounts: {
				id: {
					initialize(config: unknown): void;
					renderButton(
						parent: HTMLElement,
						options: unknown,
						callback: (response: unknown) => void
					): void;
					prompt(callback: (notification: unknown) => void): void;
				};
			};
		};
		katex: (tex: string, options?: KatexOptions) => string;
		renderMathInElement: (element: HTMLElement, options?: KatexOptions) => void;
		__egoOnlineHandler?: () => void;
		__egoOfflineHandler?: () => void;
		__egoMemoryEnabled?: boolean;
		__egoMaintenanceStore?: {
			isMaintenanceActive: boolean;
		};
		__egoImageUrlCache?: Map<string, string>;
		__egoImageInflight?: Map<string, Promise<string>>;
	}
}
declare module 'marked' {
	export interface MarkedOptions {
		highlight?: (code: string, lang: string) => string;
		langPrefix?: string;
	}
}
declare module '$service-worker' {
	export const build: string[];
	export const files: string[];
	export const version: string;
}
export {};
