import { _ } from 'svelte-i18n';
import { get } from 'svelte/store';
export interface ToolStatusConfig {
	name: string;
	icon?: string;
	color?: string;
}
export const TOOL_CONFIGS: Record<string, ToolStatusConfig> = {
	ego_search: {
		name: 'ego_search',
		icon: '🌐',
		color: 'text-blue-400'
	},
	ego_memory: {
		name: 'ego_memory',
		icon: '🧠',
		color: 'text-purple-400'
	},
	ego_calc: {
		name: 'ego_calc',
		icon: '🔢',
		color: 'text-green-400'
	},
	ego_code_exec: {
		name: 'ego_code_exec',
		icon: '💻',
		color: 'text-yellow-400'
	},
	alter_ego: {
		name: 'alter_ego',
		icon: '🎭',
		color: 'text-pink-400'
	},
	ego_knowledge: {
		name: 'ego_knowledge',
		icon: '📖',
		color: 'text-gray-400'
	},
	super_ego: {
		name: 'super_ego',
		icon: '🤝',
		color: 'text-cyan-400'
	}
};
export function parseToolHeader(header: string): {
	tool: string;
	status: 'using' | 'completed' | 'failed' | 'unknown';
	progress?: string;
} {
	const lowerHeader = header.toLowerCase();

	// Parse progress pattern like "Using EgoSearch... (2/5)" or "EgoSearch completed (3/5)"
	const progressMatch = header.match(/\((\d+\/\d+)\)/);
	const progress = progressMatch ? progressMatch[1] : undefined;

	const usingMatch = header.match(/using\s+(\w+)/i);
	if (usingMatch) {
		return { tool: usingMatch[1], status: 'using', progress };
	}

	const completedMatch = header.match(/(\w+)\s+(completed|выполнен|done)/i);
	if (completedMatch) {
		return { tool: completedMatch[1], status: 'completed', progress };
	}

	const failedMatch = header.match(/(\w+)\s+(failed|ошибка|error)/i);
	if (failedMatch) {
		return { tool: failedMatch[1], status: 'failed', progress };
	}

	for (const toolName of Object.keys(TOOL_CONFIGS)) {
		if (lowerHeader.includes(toolName.toLowerCase())) {
			if (
				lowerHeader.includes('complet') ||
				lowerHeader.includes('done') ||
				lowerHeader.includes('выполнен')
			) {
				return { tool: toolName, status: 'completed', progress };
			}
			if (
				lowerHeader.includes('fail') ||
				lowerHeader.includes('error') ||
				lowerHeader.includes('ошибка')
			) {
				return { tool: toolName, status: 'failed', progress };
			}
			return { tool: toolName, status: 'using', progress };
		}
	}
	return { tool: header, status: 'unknown', progress };
}
export function formatToolHeader(rawHeader: string): string {
	const { tool, status, progress } = parseToolHeader(rawHeader);
	const config = TOOL_CONFIGS[tool];

	try {
		const translator = get(_);
		if (config && status !== 'unknown') {
			const progressSuffix = progress ? ` (${progress})` : '';

			if (status === 'using') {
				const key = `tools.${tool.toLowerCase()}.using`;
				const fallbackKey = 'tools.generic.using';
				let translated = translator(key, { default: '' });
				if (!translated) {
					translated = translator(fallbackKey, {
						default: `Using ${tool}...`,
						values: { tool }
					});
				}
				return `${config.icon || ''} ${translated}${progressSuffix}`;
			} else if (status === 'completed') {
				const key = `tools.${tool.toLowerCase()}.completed`;
				const fallbackKey = 'tools.generic.completed';
				let translated = translator(key, { default: '' });
				if (!translated) {
					translated = translator(fallbackKey, {
						default: `${tool} completed`,
						values: { tool }
					});
				}
				return `${config.icon || ''} ${translated}${progressSuffix}`;
			} else if (status === 'failed') {
				return `${config.icon || ''} ${tool} failed${progressSuffix}`;
			}
		}
		return rawHeader;
	} catch (error) {
		console.warn('Tool header translation failed:', error);
		return rawHeader;
	}
}
export function getToolColor(toolName: string): string {
	const config = TOOL_CONFIGS[toolName];
	return config?.color || 'text-accent';
}
export function getToolIcon(toolName: string): string {
	const config = TOOL_CONFIGS[toolName];
	return config?.icon || '';
}
