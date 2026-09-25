<!--
  A logged request or response: a heading with copy buttons, a line of
  details, then the body. The body can be copied as it is or, when it's
  JSON, with blank values; a JSON response can also be copied as a schema.
  Everything is shown as text (Svelte escapes {values}), never as HTML,
  because it came from whoever sent the request or from the real API.
-->
<script lang="ts">
	import { blankValuesFor } from '$lib/blank';
	import { formatBody } from '$lib/format';
	import { jsonSchemaFor } from '$lib/schema';
	import CopyButton from './CopyButton.svelte';
	import BracesIcon from '@lucide/svelte/icons/braces';
	import EraserIcon from '@lucide/svelte/icons/eraser';

	type Props = {
		name: 'request' | 'response'; // for the copy buttons' labels
		heading: string;
		details: string;
		body: string;
		truncated: boolean;
		empty: string; // shown when there's no body, such as "No payload"
	};
	let { name, heading, details, body, truncated, empty }: Props = $props();

	const blanked = $derived(body ? blankValuesFor(body) : null);
	const schema = $derived(name === 'response' && body ? jsonSchemaFor(body) : null);
</script>

<div class="grid content-start gap-1">
	<div class="flex min-h-7 flex-wrap items-center justify-between gap-x-2">
		<p class="text-foreground font-medium">{heading}</p>
		{#if body}
			<span class="flex gap-1">
				<CopyButton
					text={formatBody(body)}
					label="Copy the {name} body"
					class="bg-hero-soft text-hero-soft-foreground hover:bg-hero/20 hover:text-hero-soft-foreground"
				/>
				{#if blanked}
					<CopyButton
						text={blanked}
						label="Copy the {name} with blank values"
						icon={EraserIcon}
						class="bg-chart-2/15 text-chart-2 hover:bg-chart-2/25 hover:text-chart-2"
					/>
				{/if}
				{#if schema}
					<CopyButton
						text={schema}
						label="Copy the response's structure as a JSON Schema"
						icon={BracesIcon}
						class="bg-chart-1/15 text-chart-1 hover:bg-chart-1/25 hover:text-chart-1"
					/>
				{/if}
			</span>
		{/if}
	</div>
	<p class="text-muted-foreground">{details}</p>
	{#if body}
		<pre
			class="bg-muted max-h-80 overflow-auto rounded-md p-2.5 font-mono break-all whitespace-pre-wrap">{formatBody(
				body
			)}</pre>
		{#if truncated}
			<p class="text-muted-foreground">The body was cut at fapi's size limit.</p>
		{/if}
	{:else}
		<p class="text-muted-foreground">{empty}</p>
	{/if}
</div>
