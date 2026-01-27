<script lang="ts">
	import ChatView from '../[sessionID]/ChatView.svelte';
	import type { PageData } from './$types';
	import { _ } from 'svelte-i18n';

	let { data: pageData } = $props<{ data: PageData }>();

	const data = $derived({
		session: null,
		messages: [],
		...pageData
	});
</script>

<svelte:head>
	<title>{$_('chat.new_chat')} - EGO</title>
</svelte:head>

<ChatView {data} />
