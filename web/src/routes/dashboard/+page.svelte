<!--
  The Dashboard screen. On the left, a chart for each proxy of the requests
  its port received, proxied and sent over the last few minutes; on the
  right, those requests as a live list. On narrow screens the list goes
  under the charts.

  The charts come from fapi's traffic counts (GET /api/traffic), which are
  kept whether or not the request log is on; the list is the request log.
-->
<script lang="ts">
	import * as api from '$lib/api';
	import { app } from '$lib/app-state.svelte';
	import ChoiceButtons from '$lib/components/ChoiceButtons.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import TrafficChart from '$lib/components/TrafficChart.svelte';
	import TrafficList from '$lib/components/TrafficList.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import { totalCounts, trafficSeries } from '$lib/traffic';

	/** How often the charts are brought up to date. */
	const refreshMilliseconds = 2000;

	const periods: { value: api.TrafficMinutes; label: string }[] = [
		{ value: 5, label: '5 min' },
		{ value: 15, label: '15 min' },
		{ value: 30, label: '30 min' },
		{ value: 60, label: '1 hour' }
	];

	let minutes = $state<api.TrafficMinutes>(15);
	let report = $state<api.TrafficReport | null>(null);

	// Load the counts for the chosen period now and every 2 seconds after,
	// starting again when the period changes.
	$effect(() => {
		const period = minutes;
		let stopped = false;
		const load = () =>
			app.showErrors(
				api.getTraffic(period).then((loaded) => {
					if (!stopped) {
						report = loaded;
					}
				})
			);
		load();
		const timer = setInterval(load, refreshMilliseconds);
		return () => {
			stopped = true;
			clearInterval(timer);
		};
	});

	type Chart = {
		port: number;
		title: string;
		description: string;
		off: boolean;
		counts: api.TrafficCounts[];
	};

	// A chart for each proxy, then one for each other port that received
	// requests in the period (a port with endpoints but no proxy), so no
	// traffic goes unshown.
	const charts = $derived.by<Chart[]>(() => {
		if (!report) {
			return [];
		}
		const ports = report.ports;
		const steps = Object.values(ports)[0]?.length ?? 60;
		const countsFor = (port: number) =>
			ports[port] ?? Array.from({ length: steps }, () => ({ received: 0, proxied: 0, sent: 0 }));

		const proxies = app.proxies ?? [];
		const charts: Chart[] = proxies.map((proxy) => ({
			port: proxy.port,
			title: `Port ${proxy.port}`,
			description: `Proxy to ${proxy.realAPI}`,
			off: !proxy.enabled,
			counts: countsFor(proxy.port)
		}));
		const others = Object.keys(ports)
			.map(Number)
			.filter((port) => !proxies.some((proxy) => proxy.port === port))
			.sort((a, b) => a - b);
		for (const port of others) {
			charts.push({
				port,
				title: `Port ${port}`,
				description: app.proxies ? 'No proxy: fapi answers every request itself' : 'Endpoints only',
				off: false,
				counts: countsFor(port)
			});
		}
		return charts;
	});

	const periodLabel = $derived(periods.find((period) => period.value === minutes)?.label ?? '');
</script>

<PageHeader
	title="Dashboard"
	description="The requests each port received, and whether fapi proxied them or sent the response itself."
/>

<div class="grid items-start gap-(--section-gap) xl:grid-cols-2">
	<section class="grid gap-4" aria-labelledby="charts-heading">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h2 id="charts-heading" class="text-lg font-semibold tracking-tight">Requests over time</h2>
			<ChoiceButtons
				label="Period"
				options={periods}
				value={minutes}
				onChange={(value) => (minutes = value)}
			/>
		</div>

		<ul class="text-muted-foreground grid gap-1 text-sm">
			{#each trafficSeries as series (series.kind)}
				<li class="flex items-baseline gap-2">
					<span class="size-2.5 shrink-0 self-center rounded-full {series.swatch}" aria-hidden="true"></span>
					<span><strong class="text-foreground font-medium">{series.label}</strong>: {series.description}</span>
				</li>
			{/each}
		</ul>

		{#if report}
			{#each charts as chart (chart.port)}
				{@const total = totalCounts(chart.counts)}
				<Card.Root size="sm">
					<!-- The totals go under the title when there isn't room beside it. -->
					<Card.Header class="flex flex-wrap items-start justify-between gap-x-6 gap-y-3">
						<div class="grid min-w-0 gap-1">
							<Card.Title class="flex items-center gap-2">
								{chart.title}
								{#if chart.off}<Badge variant="outline">Off</Badge>{/if}
							</Card.Title>
							<Card.Description class="wrap-anywhere">{chart.description}</Card.Description>
						</div>
						<dl class="flex gap-4">
							{#each trafficSeries as series (series.kind)}
								<div>
									<dt class="text-muted-foreground text-xs">{series.label}</dt>
									<dd class="text-base font-semibold tabular-nums">{total[series.kind]}</dd>
								</div>
							{/each}
						</dl>
					</Card.Header>
					<Card.Content>
						<TrafficChart
							counts={chart.counts}
							start={report.start}
							stepSeconds={report.stepSeconds}
							label="Port {chart.port} in the last {periodLabel}"
						/>
					</Card.Content>
				</Card.Root>
			{:else}
				<Card.Root>
					<Card.Content class="text-muted-foreground grid justify-items-start gap-3">
						{#if app.proxies}
							<p>No proxies yet, and no requests in the last {periodLabel}.</p>
							<Button size="sm" variant="outline" href="/proxies">Add a proxy</Button>
						{:else}
							<p>No requests in the last {periodLabel}.</p>
						{/if}
					</Card.Content>
				</Card.Root>
			{/each}
			<p class="text-muted-foreground text-xs">
				Each point is {report.stepSeconds} seconds of requests. The counts start again when fapi restarts.
			</p>
		{/if}
	</section>

	<section class="xl:sticky xl:top-(--page-padding-y)" aria-labelledby="traffic-heading">
		<Card.Root class="gap-3 pb-0">
			<Card.Header>
				<Card.Title id="traffic-heading" class="text-lg">Traffic</Card.Title>
				<Card.Description>Newest first, as it arrives. Open a request to see its payload and response.</Card.Description>
				{#if app.features?.requestLog}
					<Card.Action>
						<Button variant="outline" size="sm" onclick={app.clearRequests}>Clear</Button>
					</Card.Action>
				{/if}
			</Card.Header>
			{#if app.features?.requestLog}
				<TrafficList requests={app.requests} mocks={app.data.mocks} proxies={app.proxies} />
			{:else}
				<Card.Content class="text-muted-foreground pb-(--card-spacing)">
					The request log is turned off in the fapi config (requestLog), so there are no requests to
					list. The charts still count them.
				</Card.Content>
			{/if}
		</Card.Root>
	</section>
</div>
