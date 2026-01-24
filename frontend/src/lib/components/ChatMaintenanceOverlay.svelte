<script lang="ts">
	import { maintenanceStore } from '$lib/stores/maintenance-store.svelte.ts';
	import { _ } from 'svelte-i18n';
	import { fade, scale } from 'svelte/transition';

	let inputToken = $state('');
	let tokenError = $state<string | null>(null);
	let isSubmitting = $state(false);

	async function submitToken(e: Event) {
		e.preventDefault();
		tokenError = null;
		const token = inputToken.trim();
		if (!token) return;

		isSubmitting = true;
		const ok = await maintenanceStore.setBypassToken(token);
		isSubmitting = false;

		if (!ok) {
			tokenError = 'INVALID_KEY';
		}
	}
</script>

<div 
	class="relative w-full overflow-hidden rounded-3xl border border-black/10 bg-secondary/40 p-10 backdrop-blur-2xl dark:border-white/5 dark:bg-secondary/20 shadow-2xl font-mono"
	in:scale={{ duration: 600, start: 0.98 }}
>
	<!-- Scanline Effect -->
	<div class="scanlines pointer-events-none absolute inset-0 opacity-[0.02]"></div>

	<div class="relative z-10 flex flex-col items-center text-center">
		<!-- Status Info -->
		<div class="mb-8">
			<h2 
				class="mb-2 text-2xl font-black tracking-tighter text-text-primary sm:text-3xl uppercase flex flex-wrap justify-center gap-x-[0.3em]"
			>
				{#each $_('maintenance.chat_title').split(' ') as word}
					{@const isTarget = word.toLowerCase().includes('maintenance') || word.toLowerCase().includes('техобслуживании')}
					<span class={isTarget ? 'logo-malfunction text-accent' : ''}>
						{word}
					</span>
				{/each}
			</h2>
			<p class="text-text-secondary text-[10px] font-bold tracking-[0.3em] uppercase opacity-60">
				{$_('maintenance.chat_description')}
			</p>
		</div>

		<!-- Bypass Form -->
		<form
			class="flex w-full max-w-xs flex-col gap-3"
			onsubmit={submitToken}
			autocomplete="off"
		>
			<div class="relative">
				<input
					type="password"
					class="auth-input !bg-secondary/40 !border-black/5 dark:!border-white/5 !pl-0 !pr-0 text-center text-xs tracking-widest placeholder:opacity-30"
					placeholder="ACCESS_KEY"
					bind:value={inputToken}
					spellcheck="false"
					autocapitalize="off"
				/>
			</div>

			<button
				type="submit"
				class="btn-gradient px-6 py-3 text-xs font-bold tracking-widest uppercase disabled:opacity-30"
				disabled={isSubmitting || !inputToken}
			>
				{isSubmitting ? 'VERIFYING...' : 'UNLOCK_CHAT'}
			</button>

			{#if tokenError}
				<p class="text-center text-[10px] font-bold text-red-500 uppercase tracking-widest animate-pulse" in:fade>
					ACCESS_DENIED
				</p>
			{/if}
		</form>
	</div>

	<!-- Decorative Bottom Line -->
	<div class="absolute bottom-0 left-0 h-[2px] w-full bg-gradient-to-r from-transparent via-accent/20 to-transparent"></div>
</div>

<style>
	.scanlines {
		background: linear-gradient(
			to bottom,
			rgba(255, 255, 255, 0) 50%,
			rgba(0, 0, 0, 0.1) 50%
		);
		background-size: 100% 4px;
	}

	.logo-malfunction {
		animation: glitch-malfunction 6s infinite step-end;
		filter: contrast(1.1);
	}

	@keyframes glitch-malfunction {
		0%, 86%, 100% { transform: translate(0); filter: none; opacity: 1; text-shadow: none; }
		87% { transform: translate(-2px, 1px); filter: hue-rotate(90deg) saturate(1.5); clip-path: inset(10% 0 40% 0); color: #ff003c; opacity: 0.9; text-shadow: 2px 0 #00fff2; }
		88% { transform: translate(2px, -1px); filter: hue-rotate(-90deg) contrast(2); clip-path: inset(40% 0 10% 0); color: #ff003c; opacity: 0.8; text-shadow: -2px 0 #00fff2; }
		89% { transform: translate(0); filter: none; clip-path: none; opacity: 1; text-shadow: none; }
		90% { transform: translate(-3px, 2px); filter: hue-rotate(45deg); clip-path: inset(5% 0 60% 0); color: #ff003c; opacity: 0.7; }
		91% { transform: translate(0); filter: none; clip-path: none; opacity: 1; }
	}
</style>
