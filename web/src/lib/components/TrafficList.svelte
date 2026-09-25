<!--
  The dashboard's list of requests, newest first, from the request log: each
  one's port, method, path and status, and whether fapi proxied it or sent
  the response itself. It can be narrowed to proxied or sent requests and to
  one port, and opening a request shows its payload and the response it got
  (the real API's, when proxied), each with a button to copy it. Everything
  is shown as text (Svelte escapes {values}), never as HTML, because it came
  from whoever sent the request or from the real API.
-->
<script lang="ts">
	import type { LoggedRequest, Mock, Proxy } from '$lib/api';
	import { formatBody, formatTime } from '$lib/format';
	import { requestKind, trafficSeries, type TrafficKind } from '$lib/traffic';
	import { cn } from '$lib/utils';
	import ChoiceButtons from './ChoiceButtons.svelte';
	import CopyButton from './CopyButton.svelte';
	import { NativeSelect, NativeSelectOption } from '$lib/components/ui/native-select';

	type Props = {
		requests: LoggedRequest[];
		mocks: Mock[];
		/** The saved proxies, or null when proxies are turned off in fapi's settings. */
		proxies: Proxy[] | null;
	};
	let { requests, mocks, proxies }: Props = $props();

	let kind = $state<TrafficKind>('received');
	let port = $state('all');

	const ports = $derived(
		[...new Set([...requests.map((request) => request.port), ...(proxies ?? []).map((p) => p.port)])].sort(
			(a, b) => a - b
		)
	);

	const onPort = $derived(
		port === 'all' ? requests : requests.filter((request) => request.port === Number(port))
	);

	const shown = $derived(
		kind === 'received' ? onPort : onPort.filter((request) => requestKind(request) === kind)
	);

	const kindOptions = $derived(
		trafficSeries.map((series) => ({
			value: series.kind,
			label: `${series.label} ${
				series.kind === 'received'
					? onPort.length
					: onPort.filter((request) => requestKind(request) === series.kind).length
			}`
		}))
	);

	/** What happened to a request, such as "Proxied to http://localhost:3000". */
	function describe(request: LoggedRequest): string {
		switch (request.outcome) {
			case 'proxied': {
				// The proxy may have changed since; then only the port is known.
				const proxy = proxies?.find((saved) => saved.port === request.port);
				return proxy && proxy.upstreamPort === request.upstream
					? `Proxied to ${proxy.realAPI}`
					: `Proxied to port ${request.upstream}`;
			}
			case 'mocked': {
				const mock = mocks.find((saved) => saved.id === request.mockId);
				return mock ? `Sent by the endpoint ${mock.method} ${mock.path}` : 'Sent by an endpoint';
			}
			case 'unmatched':
				return 'Sent a 404: no endpoint matched, and no proxy was on';
			case 'preflight':
				return 'Sent a CORS preflight answer';
		}
	}

	function statusClass(status: number): string {
		if (status >= 500) {
			return 'text-destructive';
		}
		if (status >= 400) {
			return 'text-warning';
		}
		return 'text-muted-foreground';
	}
</script>

<!-- A request or response: a heading with a copy button, a line of details, then the body. -->
{#snippet part(heading: string, copyLabel: string, details: string, text: string, truncated: boolean, empty: string)}
	<div class="grid content-start gap-1">
		<div class="flex min-h-6 items-center justify-between gap-2">
			<p class="text-foreground font-medium">{heading}</p>
			{#if text}
				<CopyButton text={formatBody(text)} label={copyLabel} />
			{/if}
		</div>
		<p class="text-muted-foreground">{details}</p>
		{#if text}
			<pre
				class="bg-muted max-h-72 overflow-auto rounded-md p-2 font-mono break-all whitespace-pre-wrap">{formatBody(
					text
				)}</pre>
			{#if truncated}
				<p class="text-muted-foreground">The body was cut at fapi's size limit.</p>
			{/if}
		{:else}
			<p class="text-muted-foreground">{empty}</p>
		{/if}
	</div>
{/snippet}

<div class="grid gap-3">
	<div class="flex flex-wrap items-center gap-2 px-(--card-spacing)">
		<ChoiceButtons label="Show" options={kindOptions} value={kind} onChange={(value) => (kind = value)} />
		<NativeSelect size="sm" bind:value={port} aria-label="Port" class="ml-auto">
			<NativeSelectOption value="all">All ports</NativeSelectOption>
			{#each ports as each (each)}
				<NativeSelectOption value={String(each)}>Port {each}</NativeSelectOption>
			{/each}
		</NativeSelect>
	</div>

	<ul class="max-h-[36rem] divide-y overflow-y-auto border-t xl:max-h-[calc(100dvh-17rem)]">
		{#each shown as request (request.id)}
			{@const series = trafficSeries.find((each) => each.kind === requestKind(request))}
			<li>
				<details class="group">
					<summary
						class={cn(
							'grid cursor-pointer list-none gap-0.5 px-(--card-spacing) py-2.5 [&::-webkit-details-marker]:hidden',
							'hover:bg-muted/50 focus-visible:bg-muted/50 outline-none'
						)}
					>
						<span class="flex items-baseline gap-2">
							<span class="size-2 shrink-0 self-center rounded-full {series?.swatch}" aria-hidden="true"></span>
							<span class="font-medium">{request.method}</span>
							<span class="min-w-0 flex-1 font-mono text-xs break-all">{request.path}{request.search}</span>
							<span class="font-medium tabular-nums {statusClass(request.status)}">{request.status}</span>
						</span>
						<span class="text-muted-foreground flex flex-wrap gap-x-2 pl-4 text-xs">
							<span class="tabular-nums">{formatTime(request.time)}</span>
							<span>· port {request.port}</span>
							<span>· {describe(request)}</span>
						</span>
					</summary>

					<!-- The request and response side by side when there's room. -->
					<div class="@container px-(--card-spacing) pb-3 pl-[calc(var(--card-spacing)+1rem)] text-xs">
						<div class="grid gap-3 @xl:grid-cols-2">
							{@render part(
								'Request',
								'Copy the request body',
								`${request.contentType || 'No content type'} · from port ${request.fromPort}`,
								request.body,
								request.bodyTruncated,
								'No payload'
							)}

							{#if request.response?.error}
								<div class="grid content-start gap-1">
									<p class="text-foreground flex min-h-6 items-center font-medium">
										No response from the real API
									</p>
									<p class="text-destructive">{request.response.error}</p>
								</div>
							{:else if request.response}
								{@render part(
									request.outcome === 'proxied' ? 'Response from the real API' : 'Response from fapi',
									'Copy the response body',
									`${request.response.contentType || 'No content type'} · status ${request.status}`,
									request.response.body,
									request.response.bodyTruncated,
									'No body'
								)}
							{/if}
						</div>
					</div>
				</details>
			</li>
		{:else}
			<li class="text-muted-foreground px-(--card-spacing) py-6 text-center">
				{requests.length === 0 ? 'No requests yet' : 'No requests match'}
			</li>
		{/each}
	</ul>
</div>
