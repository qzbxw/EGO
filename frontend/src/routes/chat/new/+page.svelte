<script lang="ts">
	import ChatView from '../[sessionID]/ChatView.svelte';
	import type { PageData } from './$types';
	import { _ } from 'svelte-i18n';
	import { onMount } from 'svelte';
	import { chatStore } from '$lib/stores/chat.svelte';
	import { resetStreamStore } from '$lib/stores/stream.svelte.ts';

	let { data: pageData } = $props<{ data: PageData }>();

	const data = $derived({
		session: null,
		messages: [],
		...pageData
	});

	onMount(() => {
		chatStore.clearMessages();
		resetStreamStore();
	});
</script>

<svelte:head>
	<title>{$_('chat.new_chat')} - EGO</title>
</svelte:head>

<ChatView {data} />
