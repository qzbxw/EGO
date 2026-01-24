<script lang="ts">
    import { slide, fade } from 'svelte/transition';
    import { cubicOut, quartOut, quintOut } from 'svelte/easing';
    import { 
        ChevronDown, ChevronRight, Brain, Lightbulb, 
        Zap, Target, CheckCircle2, AlertCircle,
        Wrench, Search, Code2, Database, PlayCircle,
        ClipboardList, Circle, RotateCw, XCircle, SkipForward
    } from '@lucide/svelte';
    import type { ThoughtStep } from '$lib/stores/chat.svelte';
    import { chatStore } from '$lib/stores/chat.svelte';
    import MessageBody from '$lib/components/MessageBody.svelte';
    import { _ } from 'svelte-i18n';

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
    let normalizedThoughts = $derived(thoughts.map(t => {
        const step = { ...t };
        // If it's a tool output/error from DB history
        if (!step.type && ((step as any).tool_name || (step as any).output || (step as any).error)) {
            step.type = 'tool';
            step.toolName = (step as any).tool_name || (step as any).toolName || 'Unknown Tool';
            step.header = step.header || `${step.toolName} result`;
            step.status = (step as any).error ? 'failed' : 'completed';
            step.content = step.content || (step as any).output || (step as any).error;
        } else if (!step.type) {
            // Default to thought if type is missing
            step.type = 'thought';
            step.header = step.header || (step as any).thoughts_header || 'Thinking...';
            step.content = step.content || (step as any).thoughts || (step as any).text;
        }
        
        // Ensure new agentic fields are preserved
        step.confidence_score = step.confidence_score ?? (step as any).confidence_score;
        step.self_critique = step.self_critique || (step as any).self_critique;
        step.plan_status = step.plan_status || (step as any).plan_status;

        return step;
    }));

    let activePlan = $derived(chatStore.activePlan);

    let isExpanded = $state(false);
    let expandedStepIndex = $state<number | null>(null);

    $effect(() => {
        const hasRunningTools = normalizedThoughts.some(t => t.status === 'running');
        if ((isThinking && normalizedThoughts.length > 0 && normalizedThoughts.length <= 2) || hasRunningTools || (isThinking && activePlan)) {
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
            case 'completed': return CheckCircle2;
            case 'in_progress': return RotateCw;
            case 'failed': return XCircle;
            case 'skipped': return SkipForward;
            default: return Circle;
        }
    }

    function getPlanStepColor(status: string) {
        switch (status) {
            case 'completed': return 'text-green-400/70';
            case 'in_progress': return 'text-accent animate-spin-slow';
            case 'failed': return 'text-red-400/70';
            case 'skipped': return 'text-text-secondary/40';
            default: return 'text-text-secondary/30';
        }
    }

    function tryParsePlan(content: any) {
        if (!content) return null;
        
        // If it's already an object (sometimes happens with auto-parsing in the pipeline)
        if (typeof content === 'object' && content !== null) {
            if (Array.isArray(content.steps)) {
                return content;
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
        } catch (e) {
            return null;
        }
    }

    let displayCurrentHeader = $derived(
        isThinking && currentHeader && (thoughts.length === 0 || thoughts[thoughts.length - 1].header !== currentHeader)
    );
</script>

{#if normalizedThoughts.length > 0 || displayCurrentHeader || activePlan}
    <div class="reasoning-container flex flex-col mb-6 w-full max-w-2xl">
        <!-- Modern Glass Toggle Button -->
        <button 
            onclick={toggleMain}
            class="group relative flex items-center gap-3 w-fit px-4 py-2 rounded-2xl border border-white/10 bg-secondary/40 hover:bg-secondary/60 transition-all duration-300 shadow-lg backdrop-blur-xl overflow-hidden active:scale-95"
        >
            <!-- Smooth Ambient Background Glow when thinking -->
            {#if isThinking}
                <div class="absolute inset-0 bg-gradient-to-r from-accent/0 via-accent/5 to-accent/0 animate-shimmer"></div>
            {/if}

            <div class="relative flex h-6 w-6 items-center justify-center rounded-full bg-accent/10 text-accent transition-transform duration-300 {isThinking ? 'scale-100' : ''}">
                {#if isThinking}
                    <div class="absolute inset-0 rounded-full bg-accent animate-pulse-ring opacity-20"></div>
                    <Brain size={14} class="relative z-10 animate-pulse-scale" />
                {:else}
                    <CheckCircle2 size={14} class="text-green-400/80" />
                {/if}
            </div>
            
            <span class="relative text-[11px] font-black text-text-secondary group-hover:text-text-primary transition-colors uppercase tracking-[0.15em] {isThinking ? 'animate-pulse-text' : ''}">
                {#if isThinking}
                    {currentHeader || $_('chat.thinking')}
                {:else}
                    {$_('chat.thought_process') || 'Reasoning History'}
                {/if}
            </span>

            <div class="relative text-text-secondary/40 group-hover:text-text-primary/60 transition-colors ml-1 duration-300" class:rotate-180={isExpanded}>
                <ChevronDown size={14} />
            </div>
        </button>

        <!-- Detailed Timeline -->
        {#if isExpanded}
            <div 
                transition:slide={{ duration: 400, easing: quartOut }} 
                class="mt-4 flex flex-col gap-0 border-l-[1.5px] border-white/5 ml-5 pl-6 py-2"
            >
                <!-- Active Mission Plan (Persistent State) -->
                {#if activePlan}
                    <div 
                        in:fade={{ duration: 300 }}
                        class="mb-6 bg-accent/5 border border-accent/10 rounded-2xl p-4 overflow-hidden shadow-inner"
                    >
                        <div class="flex items-center gap-3 mb-4">
                            <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-accent/20 text-accent">
                                <ClipboardList size={14} />
                            </div>
                            <span class="text-[12px] font-black uppercase tracking-widest text-accent/90">
                                {activePlan.title || 'Mission Plan'}
                            </span>
                        </div>

                        <div class="flex flex-col gap-3">
                            {#each activePlan.steps as step (step.id)}
                                {@const StepIcon = getPlanStepIcon(step.status)}
                                <div class="flex items-start gap-3 group/plan-step">
                                    <div class="mt-0.5 {getPlanStepColor(step.status)} transition-colors duration-300">
                                        <StepIcon size={12} />
                                    </div>
                                    <div class="flex flex-col gap-0.5">
                                        <span class="text-[13px] leading-tight transition-colors duration-300 {step.status === 'completed' ? 'text-text-secondary/50 line-through' : 'text-text-primary/80 group-hover/plan-step:text-text-primary'}">
                                            {step.description}
                                        </span>
                                        {#if step.status !== 'pending'}
                                            <span class="text-[9px] uppercase tracking-tighter opacity-40 font-bold">
                                                {step.status}
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
                        class="group/step relative flex flex-col mb-4 last:mb-0"
                    >
                        <!-- Timeline Dot -->
                        <div class="absolute -left-[31px] top-1.5 flex h-2.5 w-2.5 items-center justify-center">
                            <div class="h-2 w-2 rounded-full border border-white/20 bg-primary transition-all duration-300 {isRunning ? 'bg-accent shadow-[0_0_8px_rgba(var(--color-accent-rgb),0.5)] scale-110' : 'group-hover/step:border-accent/50'}"></div>
                        </div>

                        <button 
                            onclick={(e) => toggleStep(i, e)}
                            class="flex flex-col w-full text-left"
                        >
                            <div class="flex items-center gap-3">
                                <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl transition-all duration-300 {isRunning ? 'bg-accent/10 text-accent' : 'bg-white/5 text-text-secondary/60 group-hover/step:text-accent group-hover/step:bg-accent/10'}">
                                    {#if step.status === 'failed'}
                                        <AlertCircle size={14} class="text-red-400" />
                                    {:else if step.type === 'tool' && step.status === 'completed'}
                                        <CheckCircle2 size={14} class="text-green-400/70" />
                                    {:else}
                                        <Icon size={14} class={isRunning ? 'animate-pulse-scale' : ''} />
                                    {/if}
                                </div>
                                <div class="flex flex-col min-w-0">
                                    <span class="text-[13px] font-semibold transition-colors duration-300 {isRunning ? 'text-accent' : 'text-text-primary/80 group-hover/step:text-text-primary'}">
                                        {step.header}
                                    </span>
                                    {#if step.type === 'tool' && step.status}
                                        <span class="text-[9px] uppercase tracking-widest text-text-secondary/50 group-hover/step:text-accent/60 transition-colors">
                                            {step.toolName} • {step.status} {step.progress ? `• ${step.progress}` : ''}
                                        </span>
                                    {/if}
                                </div>
                                <div class="ml-auto text-text-secondary/10 group-hover/step:text-text-secondary/40 transition-colors duration-300" class:rotate-180={expandedStepIndex === i}>
                                    <ChevronDown size={14} />
                                </div>
                            </div>
                        </button>

                        {#if expandedStepIndex === i && (step.content || step.reasoning)}
                            <div transition:slide={{ duration: 300, easing: quintOut }} class="mt-3 overflow-hidden">
                                <div class="thought-content text-[13px] leading-relaxed text-text-secondary/80 bg-white/[0.02] p-4 rounded-2xl border border-white/5 backdrop-blur-sm shadow-inner prose-tight">
                                    {#if step.type === 'tool'}
                                        <div class="text-[10px] font-black text-accent/80 uppercase tracking-wider mb-2">Output</div>
                                    {/if}
                                    
                                    {#if step.toolName === 'manage_plan' && step.content && step.content.includes('LOCAL_TOOL_SIGNAL')}
                                        <!-- If we still have a raw signal (interception failed or in-progress) -->
                                        {@const jsonPart = step.content.split('LOCAL_TOOL_SIGNAL:manage_plan:')[1]}
                                        <div class="flex flex-col gap-3 p-4 rounded-xl bg-accent/5 border border-accent/10">
                                            <div class="flex items-center gap-2 italic text-text-secondary/60">
                                                <RotateCw size={12} class="animate-spin-slow" />
                                                <span class="text-[11px] uppercase tracking-wider font-bold">Initializing Mission Plan...</span>
                                            </div>
                                            
                                            {#if jsonPart}
                                                <div class="text-[10px] opacity-30 truncate">
                                                    {jsonPart.slice(0, 100)}...
                                                </div>
                                            {/if}
                                        </div>
                                    {:else if step.toolName === 'manage_plan' && step.content}
                                        {@const parsedPlan = tryParsePlan(step.content)}
                                        {#if parsedPlan}
                                            <div class="flex flex-col gap-3 p-4 rounded-xl bg-accent/5 border border-accent/10">
                                                <div class="flex items-center gap-2 mb-2">
                                                    <ClipboardList size={14} class="text-accent" />
                                                    <span class="text-[11px] uppercase tracking-wider font-bold text-accent/90">
                                                        {parsedPlan.title || 'Mission Plan Update'}
                                                    </span>
                                                </div>
                                                <div class="flex flex-col gap-2">
                                                    {#each parsedPlan.steps as pStep}
                                                        {@const PStepIcon = getPlanStepIcon(pStep.status)}
                                                        <div class="flex items-start gap-3">
                                                            <div class="mt-0.5 {getPlanStepColor(pStep.status)}">
                                                                <PStepIcon size={12} />
                                                            </div>
                                                            <div class="flex flex-col gap-0.5">
                                                                <span class="text-[12px] leading-tight {pStep.status === 'completed' ? 'text-text-secondary/50 line-through' : 'text-text-secondary/90'}">
                                                                    {pStep.description}
                                                                </span>
                                                                {#if pStep.status !== 'pending'}
                                                                    <span class="text-[9px] uppercase tracking-tighter opacity-40 font-bold">
                                                                        {pStep.status}
                                                                    </span>
                                                                {/if}
                                                            </div>
                                                        </div>
                                                    {/each}
                                                </div>
                                            </div>
                                        {:else}
                                            <div class="p-3 rounded-xl bg-accent/5 border border-accent/10 text-accent/90 font-medium">
                                                <div class="flex items-center gap-2 mb-2">
                                                    <CheckCircle2 size={14} class="text-green-400/70" />
                                                    <span class="text-[11px] uppercase tracking-wider font-bold">System Action Successful</span>
                                                </div>
                                                {step.content}
                                            </div>
                                        {/if}
                                    {:else if step.content}
                                        <MessageBody text={step.content} />
                                    {/if}
                                    
                                    {#if step.reasoning}
                                        <div class="mt-4 p-3 rounded-xl bg-accent/5 border border-accent/10">
                                            <div class="text-[10px] font-black text-accent/80 uppercase tracking-wider mb-2">Strategy & Reasoning</div>
                                            <div class="text-[12px] italic text-text-secondary/70 leading-relaxed">{step.reasoning}</div>
                                        </div>
                                    {/if}

                                    <!-- Agentic Loop Metadata -->
                                    {#if step.self_critique || step.confidence_score != null || (step.plan_status && step.plan_status !== 'in_progress')}
                                        <div class="mt-4 pt-4 border-t border-white/5 flex flex-col gap-3">
                                            {#if step.self_critique}
                                                <div class="p-3 rounded-xl bg-red-500/5 border border-red-500/10">
                                                    <div class="flex items-center gap-2 mb-2">
                                                        <AlertCircle size={10} class="text-red-400" />
                                                        <div class="text-[10px] font-black text-red-400 uppercase tracking-wider">Self Critique</div>
                                                    </div>
                                                    <div class="text-[12px] italic text-text-secondary/80">{step.self_critique}</div>
                                                </div>
                                            {/if}
                                            
                                            <div class="flex flex-wrap items-center gap-4">
                                                {#if step.confidence_score != null}
                                                    <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-black/20 border border-white/5 shadow-sm">
                                                        <div class="text-[10px] font-bold uppercase tracking-wider opacity-50">Confidence</div>
                                                        <div class="text-xs font-mono font-bold text-accent">{(step.confidence_score * 100).toFixed(0)}%</div>
                                                        <!-- Mini visual bar -->
                                                        <div class="w-12 h-1 bg-white/5 rounded-full overflow-hidden hidden sm:block">
                                                            <div class="h-full bg-accent transition-all duration-1000" style="width: {step.confidence_score * 100}%"></div>
                                                        </div>
                                                    </div>
                                                {/if}
                                                
                                                {#if step.plan_status && step.plan_status !== 'in_progress'}
                                                    <div class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-black/20 border border-white/5 shadow-sm">
                                                        <div class="text-[10px] font-bold uppercase tracking-wider opacity-50">Status</div>
                                                        <div class="text-xs font-bold text-text-primary capitalize">{step.plan_status.replace('_', ' ')}</div>
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
                    <div 
                        in:fade={{ duration: 200 }}
                        class="flex items-center gap-3 py-1"
                    >
                        <div class="absolute -left-[31px] top-2.5 flex h-2.5 w-2.5 items-center justify-center">
                            <div class="h-2 w-2 rounded-full bg-accent animate-pulse-ring opacity-40"></div>
                        </div>
                        <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-accent/5 text-accent/70">
                            <Brain size={14} class="animate-pulse-scale" />
                        </div>
                        <div class="flex flex-col">
                            <span class="text-[13px] font-semibold text-accent/80 animate-pulse-text truncate">
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
        0% { opacity: 0.05; transform: translateX(-10%); }
        50% { opacity: 0.15; transform: translateX(10%); }
        100% { opacity: 0.05; transform: translateX(-10%); }
    }

    @keyframes -global-pulse-scale {
        0%, 100% { transform: scale(1); opacity: 0.9; }
        50% { transform: scale(1.05); opacity: 1; }
    }

    @keyframes -global-pulse-ring {
        0% { transform: scale(0.8); opacity: 0.5; }
        100% { transform: scale(2); opacity: 0; }
    }
    
    @keyframes -global-pulse-text {
        0%, 100% { opacity: 0.7; }
        50% { opacity: 1; }
    }

    @keyframes -global-spin-slow {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
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