<script lang="ts">
	import ChatView from '../[sessionID]/ChatView.svelte';
	import type { PageData } from './$types';
	const data: PageData = {
		session: null,
		messages: []
	};
	import { _ } from 'svelte-i18n';
</script>

<svelte:head>
	<title>{$_('chat.new_chat')} - EGO</title>
</svelte:head>

<ChatView {data} />
