<script lang="ts">
	import { slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';
	import { SvelteSet } from 'svelte/reactivity';
	import {
		Brain,
		Code2,
		AlertTriangle,
		Zap,
		CheckCircle,
		Loader2,
		XCircle,
		Users
	} from 'lucide-svelte';
	import MessageBody from '$lib/components/MessageBody.svelte';

	// Agent colors and icons
	const agentConfig = {
		researcher: {
			color: '#3b82f6', // blue
			icon: Brain,
			label: 'Researcher'
		},
		solver: {
			color: '#10b981', // green
			icon: Code2,
			label: 'Solver'
		},
		critic: {
			color: '#ef4444', // red
			icon: AlertTriangle,
			label: 'Critic'
		},
		optimizer: {
			color: '#f59e0b', // amber
			icon: Zap,
			label: 'Optimizer'
		},
		synthesizer: {
			color: '#8b5cf6', // purple
			icon: CheckCircle,
			label: 'Synthesizer'
		}
	};

	interface Agent {
		name: string;
		role: 'researcher' | 'solver' | 'critic' | 'optimizer' | 'synthesizer';
		status: 'running' | 'completed' | 'error';
		message?: string;
		error?: string;
	}

	interface Round {
		number: number;
		title: string;
		agents: Agent[];
		completed: boolean;
	}

	interface SuperEgoDebate {
		active: boolean;
		rounds: Round[];
		completed: boolean;
		summary?: string;
	}

	let { debate }: { debate: SuperEgoDebate } = $props();

	let expandedRounds = new SvelteSet<number>([1]); // Auto-expand first round
	let expandedAgents = new SvelteSet<string>();

	function toggleRound(roundNumber: number) {
		if (expandedRounds.has(roundNumber)) {
			expandedRounds.delete(roundNumber);
		} else {
			expandedRounds.add(roundNumber);
		}
	}

	function toggleAgent(agentId: string) {
		if (expandedAgents.has(agentId)) {
			expandedAgents.delete(agentId);
		} else {
			expandedAgents.add(agentId);
		}
	}

	function getAgentIcon(role: string) {
		return agentConfig[role as keyof typeof agentConfig]?.icon || Users;
	}

	function getAgentColor(role: string) {
		return agentConfig[role as keyof typeof agentConfig]?.color || '#6b7280';
	}
</script>

<div class="superego-debate">
	<div class="debate-header">
		<Users size={20} />
		<h3>super_ego Multi-Agent Debate</h3>
		{#if debate.completed}
			<span class="badge completed">Completed</span>
		{:else if debate.active}
			<span class="badge active">
				<Loader2 size={14} class="spinner" />
				In Progress
			</span>
		{/if}
	</div>

	<div class="rounds-container">
		{#each debate.rounds as round (round.number)}
			<div class="round" class:expanded={expandedRounds.has(round.number)}>
				<button class="round-header" onclick={() => toggleRound(round.number)}>
					<span class="round-number">Round {round.number}</span>
					<span class="round-title">{round.title}</span>
					<span class="round-status">
						{#if round.completed}
							<CheckCircle size={16} />
						{:else}
							<Loader2 size={16} class="spinner" />
						{/if}
					</span>
				</button>

				{#if expandedRounds.has(round.number)}
					<div class="agents-container" transition:slide={{ duration: 300, easing: quintOut }}>
						{#each round.agents as agent (`${round.number}-${agent.name}`)}
							{@const agentId = `${round.number}-${agent.name}`}
							{@const isExpanded = expandedAgents.has(agentId)}
							<div
								class="agent"
								class:running={agent.status === 'running'}
								class:completed={agent.status === 'completed'}
								class:error={agent.status === 'error'}
								style="border-left-color: {getAgentColor(agent.role)}"
							>
								<button class="agent-header" onclick={() => toggleAgent(agentId)}>
									<div class="agent-info">
										<svelte:component
											this={getAgentIcon(agent.role)}
											size={18}
											style="color: {getAgentColor(agent.role)}"
										/>
										<span class="agent-name">{agent.name}</span>
										{#if agent.status === 'running'}
											<Loader2
												size={14}
												class="spinner"
												style="color: {getAgentColor(agent.role)}"
											/>
										{:else if agent.status === 'completed'}
											<CheckCircle size={14} style="color: {getAgentColor(agent.role)}" />
										{:else if agent.status === 'error'}
											<XCircle size={14} style="color: #ef4444" />
										{/if}
									</div>
									<span class="expand-indicator">{isExpanded ? '▼' : '▶'}</span>
								</button>

								{#if isExpanded && (agent.message || agent.error)}
									<div class="agent-content" transition:slide={{ duration: 250, easing: quintOut }}>
										{#if agent.error}
											<div class="error-message">
												<XCircle size={16} />
												<span>{agent.error}</span>
											</div>
										{:else if agent.message}
											<MessageBody text={agent.message} />
										{/if}
									</div>
								{/if}
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{/each}
	</div>

	{#if debate.summary && debate.completed}
		<div class="debate-summary" transition:slide={{ duration: 300, easing: quintOut }}>
			<h4>Final Consensus</h4>
			<p>{debate.summary}</p>
		</div>
	{/if}
</div>

<style>
	.superego-debate {
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: 12px;
		overflow: hidden;
		margin: 1rem 0;
	}

	.debate-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 1rem;
		background: var(--surface-3);
		border-bottom: 1px solid var(--border);
	}

	.debate-header h3 {
		margin: 0;
		font-size: 1rem;
		font-weight: 600;
		flex: 1;
	}

	.badge {
		display: flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.25rem 0.625rem;
		border-radius: 12px;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.badge.active {
		background: rgba(59, 130, 246, 0.1);
		color: #3b82f6;
	}

	.badge.completed {
		background: rgba(16, 185, 129, 0.1);
		color: #10b981;
	}

	.spinner {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}

	.rounds-container {
		display: flex;
		flex-direction: column;
	}

	.round {
		border-bottom: 1px solid var(--border);
	}

	.round:last-child {
		border-bottom: none;
	}

	.round-header {
		width: 100%;
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 0.875rem 1rem;
		background: transparent;
		border: none;
		cursor: pointer;
		transition: background 0.15s;
		text-align: left;
		color: var(--text);
	}

	.round-header:hover {
		background: var(--surface-3);
	}

	.round-number {
		font-weight: 600;
		color: var(--text-2);
		min-width: 4rem;
	}

	.round-title {
		flex: 1;
		font-weight: 500;
	}

	.round-status {
		display: flex;
		align-items: center;
		color: var(--text-2);
	}

	.agents-container {
		padding: 0.5rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		background: var(--surface-1);
	}

	.agent {
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-left-width: 4px;
		border-radius: 8px;
		overflow: hidden;
		transition:
			transform 0.15s,
			box-shadow 0.15s;
	}

	.agent:hover {
		transform: translateX(2px);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
	}

	.agent-header {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem;
		background: transparent;
		border: none;
		cursor: pointer;
		color: var(--text);
	}

	.agent-info {
		display: flex;
		align-items: center;
		gap: 0.625rem;
	}

	.agent-name {
		font-weight: 500;
		font-size: 0.9375rem;
	}

	.expand-indicator {
		color: var(--text-2);
		font-size: 0.75rem;
		transition: transform 0.2s;
	}

	.agent-content {
		padding: 0 0.75rem 0.75rem 0.75rem;
		font-size: 0.9375rem;
		line-height: 1.6;
		color: var(--text-2);
	}

	.error-message {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.75rem;
		background: rgba(239, 68, 68, 0.1);
		border-radius: 6px;
		color: #ef4444;
		font-size: 0.875rem;
	}

	.debate-summary {
		padding: 1rem;
		background: var(--surface-3);
		border-top: 2px solid var(--border);
	}

	.debate-summary h4 {
		margin: 0 0 0.5rem 0;
		font-size: 0.9375rem;
		font-weight: 600;
		color: #8b5cf6;
	}

	.debate-summary p {
		margin: 0;
		font-size: 0.875rem;
		line-height: 1.6;
		color: var(--text-2);
	}
</style>
