<script lang="ts">
	import { slide, fade } from 'svelte/transition';
	import { quartOut, cubicOut, quintOut } from 'svelte/easing';
	import {
		Brain,
		CheckCircle2,
		Circle,
		ChevronDown,
		Search,
		Code2,
		Zap,
		Database,
		PlayCircle,
		ClipboardList,
		Wrench,
		Lightbulb,
		Target,
		RotateCw,
		XCircle,
		SkipForward,
		AlertCircle
	} from 'lucide-svelte';
	import type { ThoughtStep, PlanStep } from '$lib/types';
	import { chatStore } from '$lib/stores/chat.svelte';
	import MessageBody from '$lib/components/MessageBody.svelte';
	import { _ } from 'svelte-i18n';
	import type { SessionPlan } from '$lib/types';

	let {
		thoughts = [],
		currentHeader = '',
		isThinking = false
	}: {
		thoughts: ThoughtStep[];
		currentHeader?: string;
		isThinking?: boolean;
	} = $props();

	// Normalize thoughts from history if they have missing fields
	let normalizedThoughts = $derived(
		thoughts.map((t) => {
			const step = { ...t };
			const s = step as unknown as Record<string, unknown>;
			// If it's a tool output/error from DB history
			if (!step.type && (s.tool_name || s.output || s.error)) {
				step.type = 'tool';
				step.toolName = (s.tool_name as string) || (s.toolName as string) || 'Unknown Tool';
				step.header = step.header || `${step.toolName} result`;
				step.status = s.error ? 'failed' : 'completed';
				step.content = step.content || (s.output as string) || (s.error as string);
			} else if (!step.type) {
				// Default to thought if type is missing
				step.type = 'thought';
				step.header = step.header || (s.thoughts_header as string) || 'Thinking...';
				step.content = step.content || (s.thoughts as string) || (s.text as string);
			}

			// Ensure new agentic fields are preserved
			step.confidence_score = step.confidence_score ?? (s.confidence_score as number);
			step.self_critique = step.self_critique || (s.self_critique as string);
			step.plan_status = step.plan_status || (s.plan_status as string);

			return step;
		})
	);

	let activePlan = $derived(chatStore.activePlan);

	let isExpanded = $state(false);
	let expandedStepIndex = $state<number | null>(null);

	$effect(() => {
		const hasRunningTools = normalizedThoughts.some((t) => t.status === 'running');
		if (
			(isThinking && normalizedThoughts.length > 0 && normalizedThoughts.length <= 2) ||
			hasRunningTools ||
			(isThinking && activePlan)
		) {
			isExpanded = true;
		}
	});

	function toggleMain() {
		isExpanded = !isExpanded;
	}

	function toggleStep(index: number, event: MouseEvent) {
		event.stopPropagation();
		expandedStepIndex = expandedStepIndex === index ? null : index;
	}

	function getIcon(step: ThoughtStep) {
		if (step.type === 'tool') {
			const name = step.toolName?.toLowerCase() || '';
			if (name.includes('search')) return Search;
			if (name.includes('code') || name.includes('exec')) return Code2;
			if (name.includes('calc')) return Zap;
			if (name.includes('wiki')) return Database;
			if (name.includes('tube') || name.includes('video')) return PlayCircle;
			if (name.includes('memory')) return Brain;
			if (name.includes('plan')) return ClipboardList;
			return Wrench;
		}

		const h = step.header.toLowerCase();
		if (h.includes('analyz') || h.includes('think')) return Brain;
		if (h.includes('idea') || h.includes('creative')) return Lightbulb;
		if (h.includes('search') || h.includes('find') || h.includes('fetch')) return Target;
		return Brain;
	}

	function getPlanStepIcon(status: string) {
		switch (status) {
			case 'completed':
				return CheckCircle2;
			case 'in_progress':
				return RotateCw;
			case 'failed':
				return XCircle;
			case 'skipped':
				return SkipForward;
			default:
				return Circle;
		}
	}

	function getPlanStepColor(status: string) {
		switch (status) {
			case 'completed':
				return 'text-green-400/70';
			case 'in_progress':
				return 'text-accent animate-spin-slow';
			case 'failed':
				return 'text-red-400/70';
			case 'skipped':
				return 'text-text-secondary/40';
			default:
				return 'text-text-secondary/30';
		}
	}

	function tryParsePlan(content: unknown) {
		if (!content) return null;

		// If it's already an object (sometimes happens with auto-parsing in the pipeline)
		if (typeof content === 'object' && content !== null) {
			const c = content as Record<string, unknown>;
			if (Array.isArray(c.steps)) {
				return content as SessionPlan;
			}
			return null;
		}

		if (typeof content !== 'string') return null;

		try {
			const parsed = JSON.parse(content);
			// Check if it looks like a plan (has steps array)
			if (parsed && typeof parsed === 'object' && Array.isArray(parsed.steps)) {
				return parsed;
			}
			return null;
		} catch {
			return null;
		}
	}

	let displayCurrentHeader = $derived(
		isThinking &&
			currentHeader &&
			(thoughts.length === 0 || thoughts[thoughts.length - 1].header !== currentHeader)
	);
</script>

{#if normalizedThoughts.length > 0 || displayCurrentHeader || activePlan}
	<div class="reasoning-container mb-6 flex w-full max-w-2xl flex-col">
		<!-- Modern Glass Toggle Button -->
		<button
			onclick={toggleMain}
			class="group relative flex w-fit items-center gap-3 overflow-hidden rounded-2xl border border-white/10 bg-secondary/40 px-4 py-2 shadow-lg backdrop-blur-xl transition-all duration-300 hover:bg-secondary/60 active:scale-95"
		>
			<!-- Smooth Ambient Background Glow when thinking -->
			{#if isThinking}
				<div
					class="animate-shimmer absolute inset-0 bg-gradient-to-r from-accent/0 via-accent/5 to-accent/0"
				></div>
			{/if}

			<div
				class="relative flex h-6 w-6 items-center justify-center rounded-full bg-accent/10 text-accent transition-transform duration-300 {isThinking
					? 'scale-100'
					: ''}"
			>
				{#if isThinking}
					<div class="animate-pulse-ring absolute inset-0 rounded-full bg-accent opacity-20"></div>
					<Brain size={14} class="animate-pulse-scale relative z-10" />
				{:else}
					<CheckCircle2 size={14} class="text-green-400/80" />
				{/if}
			</div>

			<span
				class="relative text-[11px] font-black uppercase tracking-[0.15em] text-text-secondary transition-colors group-hover:text-text-primary {isThinking
					? 'animate-pulse-text'
					: ''}"
			>
				{#if isThinking}
					{currentHeader || $_('chat.thinking')}
				{:else}
					{$_('chat.thought_process') || 'Reasoning History'}
				{/if}
			</span>

			<div
				class="relative ml-1 text-text-secondary/40 transition-colors duration-300 group-hover:text-text-primary/60"
				class:rotate-180={isExpanded}
			>
				<ChevronDown size={14} />
			</div>
		</button>

		<!-- Detailed Timeline -->
		{#if isExpanded}
			<div
				transition:slide={{ duration: 400, easing: quartOut }}
				class="ml-5 mt-4 flex flex-col gap-0 border-l-[1.5px] border-white/5 py-2 pl-6"
			>
				<!-- Active Execution Plan -->
				{#if activePlan}
					{@const completedSteps = activePlan.steps.filter(
						(s: PlanStep) => s.status === 'completed'
					).length}
					{@const totalSteps = activePlan.steps.length}
					{@const progressPercent = totalSteps > 0 ? (completedSteps / totalSteps) * 100 : 0}
					<div
						in:fade={{ duration: 300 }}
						class="mb-6 overflow-hidden rounded-xl border border-accent/15 bg-gradient-to-br from-accent/[0.03] to-accent/[0.08] p-5 backdrop-blur-sm"
					>
						<!-- Plan Header with Progress -->
						<div class="mb-4 flex items-start justify-between">
							<div class="flex items-center gap-3">
								<div
									class="flex h-8 w-8 items-center justify-center rounded-lg border border-accent/20 bg-accent/15 text-accent"
								>
									<ClipboardList size={16} strokeWidth={2.5} />
								</div>
								<div class="flex flex-col gap-0.5">
									<span class="text-[13px] font-bold tracking-tight text-text-primary/90">
										{activePlan.title || 'Execution Plan'}
									</span>
									<span class="text-[10px] font-medium tracking-wide text-text-secondary/50">
										{completedSteps} / {totalSteps} steps completed
									</span>
								</div>
							</div>
							<span class="font-mono text-[11px] font-semibold tabular-nums text-accent/70">
								{progressPercent.toFixed(0)}%
							</span>
						</div>

						<!-- Progress Bar -->
						<div class="mb-4 h-1.5 w-full overflow-hidden rounded-full bg-white/5">
							<div
								class="h-full bg-gradient-to-r from-accent/70 to-accent transition-all duration-700 ease-out"
								style="width: {progressPercent}%"
							></div>
						</div>

						<!-- Steps List -->
						<div class="flex flex-col gap-2">
							{#each activePlan.steps as step, idx (step.id)}
								{@const StepIcon = getPlanStepIcon(step.status)}
								{@const isActive = step.status === 'in_progress'}
								{@const isCompleted = step.status === 'completed'}
								{@const isFailed = step.status === 'failed'}

								<div
									class="group/plan-step flex items-start gap-3 rounded-lg px-2 py-1.5 transition-colors {isActive
										? 'bg-accent/5'
										: 'hover:bg-white/[0.02]'}"
								>
									<!-- Step Number & Icon -->
									<div class="flex shrink-0 items-center gap-2">
										<span
											class="w-4 text-right font-mono text-[10px] font-bold text-text-secondary/40"
										>
											{idx + 1}
										</span>
										<div class="mt-0.5 {getPlanStepColor(step.status)} transition-all duration-300">
											<StepIcon size={14} strokeWidth={2} />
										</div>
									</div>

									<!-- Step Description -->
									<div class="flex min-w-0 flex-1 flex-col gap-1">
										<span
											class="text-[12px] leading-snug transition-colors duration-300 {isCompleted
												? 'text-text-secondary/40 line-through'
												: isFailed
													? 'text-red-400/60'
													: isActive
														? 'font-medium text-text-primary'
														: 'text-text-secondary/70'}"
										>
											{step.description}
										</span>

										<!-- Status Badge -->
										{#if step.status !== 'pending'}
											<span
												class="w-fit rounded px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider {isCompleted
													? 'bg-green-400/10 text-green-400/70'
													: isFailed
														? 'bg-red-400/10 text-red-400/70'
														: isActive
															? 'bg-accent/10 text-accent/80'
															: 'bg-white/5 text-text-secondary/50'}"
											>
												{step.status.replace('_', ' ')}
											</span>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				{#each normalizedThoughts as step, i (step.id || i)}
					{@const Icon = getIcon(step)}
					{@const isRunning = step.status === 'running'}
					<div
						in:fade={{ duration: 300, delay: 0, easing: cubicOut }}
						class="group/step relative mb-4 flex flex-col last:mb-0"
					>
						<!-- Timeline Dot -->
						<div class="absolute -left-[31px] top-1.5 flex h-2.5 w-2.5 items-center justify-center">
							<div
								class="h-2 w-2 rounded-full border border-white/20 bg-primary transition-all duration-300 {isRunning
									? 'scale-110 bg-accent shadow-[0_0_8px_rgba(var(--color-accent-rgb),0.5)]'
									: 'group-hover/step:border-accent/50'}"
							></div>
						</div>

						<button onclick={(e) => toggleStep(i, e)} class="flex w-full flex-col text-left">
							<div class="flex items-center gap-3">
								<div
									class="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl transition-all duration-300 {isRunning
										? 'bg-accent/10 text-accent'
										: 'bg-white/5 text-text-secondary/60 group-hover/step:bg-accent/10 group-hover/step:text-accent'}"
								>
									{#if step.status === 'failed'}
										<AlertCircle size={14} class="text-red-400" />
									{:else if step.type === 'tool' && step.status === 'completed'}
										<CheckCircle2 size={14} class="text-green-400/70" />
									{:else}
										<Icon size={14} class={isRunning ? 'animate-pulse-scale' : ''} />
									{/if}
								</div>
								<div class="flex min-w-0 flex-col">
									<span
										class="text-[13px] font-semibold transition-colors duration-300 {isRunning
											? 'text-accent'
											: 'text-text-primary/80 group-hover/step:text-text-primary'}"
									>
										{step.header}
									</span>
									{#if step.type === 'tool' && step.status}
										<span
											class="text-[9px] uppercase tracking-widest text-text-secondary/50 transition-colors group-hover/step:text-accent/60"
										>
											{step.toolName} • {step.status}
											{step.progress ? `• ${step.progress}` : ''}
										</span>
									{/if}
								</div>
								<div
									class="ml-auto text-text-secondary/10 transition-colors duration-300 group-hover/step:text-text-secondary/40"
									class:rotate-180={expandedStepIndex === i}
								>
									<ChevronDown size={14} />
								</div>
							</div>
						</button>

						{#if expandedStepIndex === i && (step.content || step.reasoning)}
							<div
								transition:slide={{ duration: 300, easing: quintOut }}
								class="mt-3 overflow-hidden"
							>
								<div
									class="thought-content prose-tight rounded-2xl border border-white/5 bg-white/[0.02] p-4 text-[13px] leading-relaxed text-text-secondary/80 shadow-inner backdrop-blur-sm"
								>
									{#if step.type === 'tool'}
										<div
											class="mb-2 text-[10px] font-black uppercase tracking-wider text-accent/80"
										>
											Output
										</div>
									{/if}

									{#if step.toolName === 'manage_plan' && step.content && step.content.includes('LOCAL_TOOL_SIGNAL')}
										<!-- Plan initialization in progress -->
										{@const jsonPart = step.content.split('LOCAL_TOOL_SIGNAL:manage_plan:')[1]}
										<div
											class="flex flex-col gap-2.5 rounded-lg border border-accent/10 bg-accent/5 p-3"
										>
											<div class="flex items-center gap-2.5 text-text-secondary/70">
												<RotateCw
													size={12}
													class="animate-spin-slow text-accent/70"
													strokeWidth={2.5}
												/>
												<span class="text-[10px] font-bold uppercase tracking-wider"
													>Processing plan request</span
												>
											</div>

											{#if jsonPart}
												<div
													class="truncate font-mono text-[9px] text-text-secondary/50 opacity-25"
												>
													{jsonPart.slice(0, 80)}...
												</div>
											{/if}
										</div>
									{:else if step.toolName === 'manage_plan' && step.content}
										{@const parsedPlan = tryParsePlan(step.content)}
										{#if parsedPlan}
											{@const planCompleted = parsedPlan.steps.filter(
												(s: PlanStep) => s.status === 'completed'
											).length}
											{@const planTotal = parsedPlan.steps.length}

											<div
												class="flex flex-col gap-3 rounded-lg border border-accent/10 bg-accent/5 p-3"
											>
												<div class="flex items-center justify-between">
													<div class="flex items-center gap-2">
														<ClipboardList size={13} class="text-accent" strokeWidth={2.5} />
														<span class="text-[11px] font-bold tracking-tight text-accent/90">
															{parsedPlan.title || 'Plan Update'}
														</span>
													</div>
													<span class="font-mono text-[9px] text-text-secondary/50">
														{planCompleted}/{planTotal}
													</span>
												</div>
												<div class="flex flex-col gap-1.5">
													{#each parsedPlan.steps as pStep, pidx (pStep.id || pidx)}
														{@const PStepIcon = getPlanStepIcon(pStep.status)}
														{@const pIsCompleted = pStep.status === 'completed'}
														{@const pIsFailed = pStep.status === 'failed'}

														<div class="flex items-start gap-2">
															<span
																class="mt-0.5 w-3 text-right font-mono text-[9px] font-bold text-text-secondary/30"
															>
																{pidx + 1}
															</span>
															<div class="mt-0.5 {getPlanStepColor(pStep.status)}">
																<PStepIcon size={11} strokeWidth={2} />
															</div>
															<div class="flex flex-1 flex-col gap-0.5">
																<span
																	class="text-[11px] leading-tight {pIsCompleted
																		? 'text-text-secondary/40 line-through'
																		: pIsFailed
																			? 'text-red-400/60'
																			: 'text-text-secondary/80'}"
																>
																	{pStep.description}
																</span>
																{#if pStep.status !== 'pending'}
																	<span
																		class="w-fit rounded px-1 py-0.5 text-[8px] font-semibold uppercase tracking-wider {pIsCompleted
																			? 'bg-green-400/10 text-green-400/60'
																			: pIsFailed
																				? 'bg-red-400/10 text-red-400/60'
																				: 'bg-white/5 text-text-secondary/40'}"
																	>
																		{pStep.status.replace('_', ' ')}
																	</span>
																{/if}
															</div>
														</div>
													{/each}
												</div>
											</div>
										{:else}
											{@const isError =
												step.content &&
												(step.content.includes('invalid') ||
													step.content.includes('Error') ||
													step.content.includes('failed') ||
													step.content.includes('cannot'))}
											<div
												class="rounded-lg border p-3 {isError
													? 'border-red-500/15 bg-red-500/5'
													: 'border-accent/10 bg-accent/5'}"
											>
												<div class="mb-1.5 flex items-center gap-2">
													{#if isError}
														<AlertCircle size={12} class="text-red-400/80" strokeWidth={2.5} />
														<span
															class="text-[10px] font-bold uppercase tracking-wider text-red-400/80"
															>Plan Error</span
														>
													{:else}
														<CheckCircle2 size={12} class="text-green-400/70" strokeWidth={2.5} />
														<span
															class="text-[10px] font-bold uppercase tracking-wider text-green-400/70"
															>Plan Action</span
														>
													{/if}
												</div>
												<div
													class="text-[11px] leading-relaxed {isError
														? 'font-mono text-red-300/70'
														: 'text-text-secondary/80'}"
												>
													{step.content}
												</div>
											</div>
										{/if}
									{:else if step.content}
										<MessageBody text={step.content} />
									{/if}

									{#if step.reasoning}
										<div class="mt-3 rounded-lg border border-accent/10 bg-accent/5 p-3">
											<div
												class="mb-1.5 text-[9px] font-bold uppercase tracking-wider text-accent/70"
											>
												Reasoning
											</div>
											<div class="text-[11px] leading-relaxed text-text-secondary/75">
												{step.reasoning}
											</div>
										</div>
									{/if}

									<!-- Analysis Metadata -->
									{#if step.self_critique || step.confidence_score != null || (step.plan_status && step.plan_status !== 'in_progress')}
										<div class="mt-3 flex flex-col gap-2.5 border-t border-white/5 pt-3">
											{#if step.self_critique}
												<div class="rounded-lg border border-orange-500/10 bg-orange-500/5 p-2.5">
													<div class="mb-1.5 flex items-center gap-1.5">
														<AlertCircle size={10} class="text-orange-400/80" strokeWidth={2.5} />
														<div
															class="text-[9px] font-bold uppercase tracking-wider text-orange-400/80"
														>
															Analysis
														</div>
													</div>
													<div class="text-[11px] leading-relaxed text-text-secondary/75">
														{step.self_critique}
													</div>
												</div>
											{/if}

											<div class="flex flex-wrap items-center gap-2.5">
												{#if step.confidence_score != null}
													<div
														class="flex items-center gap-2.5 rounded-md border border-white/5 bg-white/[0.02] px-2.5 py-1.5"
													>
														<div
															class="text-[9px] font-semibold uppercase tracking-wider text-text-secondary/50"
														>
															Confidence
														</div>
														<div class="font-mono text-[10px] font-bold tabular-nums text-accent">
															{(step.confidence_score * 100).toFixed(0)}%
														</div>
														<div class="h-1 w-16 overflow-hidden rounded-full bg-white/5">
															<div
																class="h-full bg-gradient-to-r from-accent/60 to-accent transition-all duration-700"
																style="width: {step.confidence_score * 100}%"
															></div>
														</div>
													</div>
												{/if}

												{#if step.plan_status && step.plan_status !== 'in_progress'}
													<div
														class="flex items-center gap-2 rounded-md border border-white/5 bg-white/[0.02] px-2.5 py-1.5"
													>
														<div
															class="text-[9px] font-semibold uppercase tracking-wider text-text-secondary/50"
														>
															Phase
														</div>
														<div class="text-[10px] font-semibold capitalize text-text-primary/80">
															{step.plan_status.replace('_', ' ')}
														</div>
													</div>
												{/if}
											</div>
										</div>
									{/if}
								</div>
							</div>
						{/if}
					</div>
				{/each}

				{#if displayCurrentHeader}
					<div in:fade={{ duration: 200 }} class="flex items-center gap-3 py-1">
						<div class="absolute -left-[31px] top-2.5 flex h-2.5 w-2.5 items-center justify-center">
							<div class="animate-pulse-ring h-2 w-2 rounded-full bg-accent opacity-40"></div>
						</div>
						<div
							class="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-accent/5 text-accent/70"
						>
							<Brain size={14} class="animate-pulse-scale" />
						</div>
						<div class="flex flex-col">
							<span class="animate-pulse-text truncate text-[13px] font-semibold text-accent/80">
								{currentHeader}
							</span>
							<span class="text-[9px] uppercase tracking-widest text-accent/40">Thinking step</span>
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}

<style>
	.reasoning-container {
		/* perspective: 1000px; removed for cleaner text rendering */
		transform-style: preserve-3d;
		backface-visibility: hidden;
	}

	/* Tighten up Markdown inside thought bubbles */
	:global(.thought-content.prose-tight) {
		max-width: none;
		user-select: text !important;
		-webkit-user-select: text !important;
		font-variant-numeric: tabular-nums;
	}
	:global(.thought-content.prose-tight p) {
		margin-top: 0.5rem !important;
		margin-bottom: 0.5rem !important;
	}
	:global(.thought-content.prose-tight p:first-child) {
		margin-top: 0 !important;
	}
	:global(.thought-content.prose-tight p:last-child) {
		margin-bottom: 0 !important;
	}
	:global(.thought-content.prose-tight ul, .thought-content.prose-tight ol) {
		margin-top: 0.5rem !important;
		margin-bottom: 0.5rem !important;
		padding-left: 1.25rem !important;
	}
	:global(.thought-content.prose-tight li) {
		margin-top: 0.25rem !important;
		margin-bottom: 0.25rem !important;
	}
	:global(.thought-content.prose-tight pre) {
		margin: 0.5rem 0 !important;
		padding: 0.75rem !important;
		background: rgba(0, 0, 0, 0.2) !important;
	}

	/* CLEANER ANIMATIONS */
	@keyframes -global-shimmer {
		0% {
			opacity: 0.05;
			transform: translateX(-10%);
		}
		50% {
			opacity: 0.15;
			transform: translateX(10%);
		}
		100% {
			opacity: 0.05;
			transform: translateX(-10%);
		}
	}

	@keyframes -global-pulse-scale {
		0%,
		100% {
			transform: scale(1);
			opacity: 0.9;
		}
		50% {
			transform: scale(1.05);
			opacity: 1;
		}
	}

	@keyframes -global-pulse-ring {
		0% {
			transform: scale(0.8);
			opacity: 0.5;
		}
		100% {
			transform: scale(2);
			opacity: 0;
		}
	}

	@keyframes -global-pulse-text {
		0%,
		100% {
			opacity: 0.7;
		}
		50% {
			opacity: 1;
		}
	}

	@keyframes -global-spin-slow {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}

	:global(.animate-shimmer) {
		animation: shimmer 4s ease-in-out infinite;
	}

	:global(.animate-pulse-scale) {
		animation: pulse-scale 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	:global(.animate-pulse-ring) {
		animation: pulse-ring 2s cubic-bezier(0, 0, 0.2, 1) infinite;
	}

	:global(.animate-pulse-text) {
		animation: pulse-text 2s ease-in-out infinite;
	}

	:global(.animate-spin-slow) {
		animation: spin-slow 3s linear infinite;
	}
</style>
