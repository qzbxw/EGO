<script lang="ts">
	import type { PageData } from './$types';
	import { _ } from 'svelte-i18n';
	import ChatView from './ChatView.svelte';
	import { page } from '$app/stores';
	import { sessions } from '$lib/stores/sessions.svelte.ts';
	let { data } = $props<{ data: PageData }>();
	let currentSessionTitle = $derived.by(() => {
		const uuid = $page.params?.sessionID;
		const list = $sessions || [];
		const found = uuid ? list.find((s) => s.uuid === uuid) : null;
		return (found?.title || data.session?.title || $_('chat.new_chat')) + ' - EGO';
	});
</script>

<svelte:head>
	<title>{currentSessionTitle}</title>
</svelte:head>

<ChatView {data} />
