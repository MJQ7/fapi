<!--
  The title of a screen, with a short description and, optionally, buttons
  on the right. It also sets the browser tab's title.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		title: string;
		description: string;
		actions?: Snippet;
	};
	let { title, description, actions }: Props = $props();
</script>

<svelte:head>
	<title>{title} · fapi</title>
</svelte:head>

<header class="flex flex-wrap items-end justify-between gap-x-8 gap-y-4">
	<div class="grid gap-1.5">
		<h1 class="text-2xl font-semibold tracking-tight">{title}</h1>
		<p class="text-muted-foreground">{description}</p>
	</div>
	{#if actions}
		<div class="flex gap-2">{@render actions()}</div>
	{/if}
</header>
