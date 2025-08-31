<script lang="ts">
	import type { PageData } from './$types';
	import { _ } from 'svelte-i18n';
	import ChatView from './ChatView.svelte';
	import { page } from '$app/stores';
	import { sessions } from '$lib/stores/sessions.svelte.ts';
	let { data } = $props<{ data: PageData }>();
	let currentSessionTitle: string | null = null;
	$effect(() => {
		const uuid = $page.params?.sessionID;
		const list = $sessions;
		const found = Array.isArray(list) && uuid ? list.find((s) => s.uuid === uuid) : null;
		currentSessionTitle = found?.title ?? null;
	});
</script>
<svelte:head>
	<title>{currentSessionTitle || data.session?.title || $_('chat.new_chat')} - EGO</title>
</svelte:head>
{#key data.session?.uuid ?? 'new'}
	<ChatView {data} />
{/key}