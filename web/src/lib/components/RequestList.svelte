<!--
  The requests that one endpoint answered, newest first. Everything is shown
  as text (Svelte escapes {values}), never as HTML, because it came from
  whoever sent the request.
-->
<script lang="ts">
	import type { LoggedRequest } from '$lib/api';
	import { formatBody, formatTime } from '$lib/format';

	type Props = {
		requests: LoggedRequest[];
	};
	let { requests }: Props = $props();
</script>

<ul class="divide-y">
	{#each requests as request (request.id)}
		<li class="grid gap-1 px-3 py-2 text-sm">
			<div class="text-muted-foreground flex flex-wrap gap-x-2">
				<span>{formatTime(request.time)}</span>
				<span class="text-foreground font-medium break-all">
					{request.method}
					{request.path}{request.search}
				</span>
				<span>→ {request.status}</span>
				{#if request.contentType}
					<span>· {request.contentType}</span>
				{/if}
				<span>· from port {request.fromPort}</span>
			</div>

			{#if request.body}
				<pre
					class="bg-muted max-h-56 overflow-auto rounded-md p-2 font-mono text-xs break-all whitespace-pre-wrap">{formatBody(
						request.body
					)}</pre>
				{#if request.bodyTruncated}
					<p class="text-muted-foreground text-xs">The body was cut at fapi's size limit.</p>
				{/if}
			{:else}
				<p class="text-muted-foreground">No payload</p>
			{/if}
		</li>
	{/each}
</ul>
