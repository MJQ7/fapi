<!--
  The requests that one endpoint answered, newest first: each one's payload
  and the response the endpoint sent, with buttons to copy them (see
  MessagePart). Everything is shown as text (Svelte escapes {values}), never
  as HTML, because it came from whoever sent the request.
-->
<script lang="ts">
	import type { LoggedRequest } from '$lib/api';
	import { formatTime } from '$lib/format';
	import MessagePart from './MessagePart.svelte';

	type Props = {
		requests: LoggedRequest[];
	};
	let { requests }: Props = $props();
</script>

<ul class="divide-y">
	{#each requests as request (request.id)}
		<li class="grid gap-2 px-3 py-3 text-sm">
			<div class="text-muted-foreground flex flex-wrap gap-x-2">
				<span class="tabular-nums">{formatTime(request.time)}</span>
				<span class="text-foreground font-medium break-all">
					{request.method}
					{request.path}{request.search}
				</span>
				<span>→ {request.status}</span>
			</div>

			<!-- The request and response side by side when there's room. -->
			<div class="@container">
				<div class="grid gap-4 @2xl:grid-cols-2">
					<MessagePart
						name="request"
						heading="Request"
						details="{request.contentType || 'No content type'} · from port {request.fromPort}"
						body={request.body}
						truncated={request.bodyTruncated}
						empty="No payload"
					/>
					{#if request.response}
						<MessagePart
							name="response"
							heading="Response"
							details="{request.response.contentType || 'No content type'} · status {request.status}"
							body={request.response.body}
							truncated={request.response.bodyTruncated}
							empty="No body"
						/>
					{/if}
				</div>
			</div>
		</li>
	{/each}
</ul>
