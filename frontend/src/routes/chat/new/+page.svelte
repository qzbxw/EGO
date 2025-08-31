<script lang="ts">
	import ChatView from '../[sessionID]/ChatView.svelte';
	import type { PageData } from './$types';
	const data: PageData = {
		session: null,
		messages: []
	};
</script>
<ChatView {data} />