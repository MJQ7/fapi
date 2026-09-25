<!--
  A line chart of one port's traffic over a period: the requests it received,
  proxied and sent, in each step. Received is also shaded, since its line is
  hidden under another when all the requests were proxied or all were sent.
  Pointing at the chart shows the counts for that step.
-->
<script lang="ts">
	import type { TrafficCounts } from '$lib/api';
	import { trafficSeries, totalCounts, type TrafficKind } from '$lib/traffic';

	type Props = {
		counts: TrafficCounts[]; // one per step, oldest first
		start: string; // when the first step began
		stepSeconds: number;
		label: string; // what the chart shows, for screen readers
	};
	let { counts, start, stepSeconds, label }: Props = $props();

	const height = 176;
	const padding = { top: 8, right: 12, bottom: 24, left: 36 };
	const plotHeight = height - padding.top - padding.bottom;

	let width = $state(0);
	const plotWidth = $derived(Math.max(width - padding.left - padding.right, 0));

	// The top of the y axis: a round number at least as high as the busiest
	// step, and even, so the middle line is a whole number too.
	const top = $derived(roundUp(Math.max(0, ...counts.map((step) => step.received))));

	function roundUp(value: number): number {
		if (value <= 4) {
			return 4;
		}
		const magnitude = 10 ** Math.floor(Math.log10(value));
		const multiple = [1, 2, 4, 6, 8, 10].find((each) => each * magnitude >= value) ?? 10;
		return multiple * magnitude;
	}

	function x(index: number): number {
		return padding.left + (counts.length > 1 ? (index / (counts.length - 1)) * plotWidth : 0);
	}

	function y(value: number): number {
		return padding.top + plotHeight - (value / top) * plotHeight;
	}

	function line(kind: TrafficKind): string {
		return counts.map((step, index) => `${x(index)},${y(step[kind])}`).join(' ');
	}

	const receivedArea = $derived(
		`${x(0)},${y(0)} ${line('received')} ${x(counts.length - 1)},${y(0)}`
	);

	function stepTime(index: number): Date {
		return new Date(new Date(start).getTime() + index * stepSeconds * 1000);
	}

	function formatTime(time: Date, seconds: boolean): string {
		return time.toLocaleTimeString(undefined, {
			hour: '2-digit',
			minute: '2-digit',
			second: seconds ? '2-digit' : undefined
		});
	}

	// The axis shows seconds only for 5-second steps, where its times are 2½
	// minutes apart; the step under the pointer shows them unless steps are
	// whole minutes.
	const axisSeconds = $derived(stepSeconds < 15);

	// Times under the first, middle and last steps.
	const timeTicks = $derived(
		counts.length > 1
			? [
					{ index: 0, anchor: 'start', text: formatTime(stepTime(0), axisSeconds) },
					{
						index: Math.floor((counts.length - 1) / 2),
						anchor: 'middle',
						text: formatTime(stepTime(Math.floor((counts.length - 1) / 2)), axisSeconds)
					},
					{ index: counts.length - 1, anchor: 'end', text: 'Now' }
				]
			: []
	);

	let hovered = $state<number | null>(null);

	function pointAt(event: PointerEvent) {
		const bounds = (event.currentTarget as SVGElement).getBoundingClientRect();
		const fraction = (event.clientX - bounds.left - padding.left) / plotWidth;
		hovered = Math.min(Math.max(Math.round(fraction * (counts.length - 1)), 0), counts.length - 1);
	}

	const summary = $derived.by(() => {
		const total = totalCounts(counts);
		return `${label}: ${total.received} received, ${total.proxied} proxied, ${total.sent} sent`;
	});
</script>

<div class="relative" bind:clientWidth={width}>
	{#if width > 0 && counts.length > 0}
		<svg
			{width}
			{height}
			role="img"
			aria-label={summary}
			class="block touch-pan-y select-none"
			onpointermove={pointAt}
			onpointerdown={pointAt}
			onpointerleave={() => (hovered = null)}
		>
			{#each [0, top / 2, top] as value (value)}
				<line
					x1={padding.left}
					x2={padding.left + plotWidth}
					y1={y(value)}
					y2={y(value)}
					class="stroke-border"
					stroke-dasharray={value === 0 ? undefined : '3 3'}
				/>
				<text
					x={padding.left - 8}
					y={y(value)}
					text-anchor="end"
					dominant-baseline="middle"
					class="fill-muted-foreground text-[11px] tabular-nums">{value}</text
				>
			{/each}

			{#each timeTicks as tick (tick.index)}
				<text
					x={x(tick.index)}
					y={height - 6}
					text-anchor={tick.anchor}
					class="fill-muted-foreground text-[11px] tabular-nums">{tick.text}</text
				>
			{/each}

			<polygon points={receivedArea} class="fill-chart-1 opacity-10" />
			{#each trafficSeries as series (series.kind)}
				<polyline
					points={line(series.kind)}
					fill="none"
					stroke-width="2"
					stroke-linejoin="round"
					stroke-linecap="round"
					class={series.stroke}
				/>
			{/each}

			{#if hovered !== null}
				<line
					x1={x(hovered)}
					x2={x(hovered)}
					y1={padding.top}
					y2={padding.top + plotHeight}
					class="stroke-muted-foreground/50"
				/>
				{#each trafficSeries as series (series.kind)}
					<circle
						cx={x(hovered)}
						cy={y(counts[hovered][series.kind])}
						r="3.5"
						class="{series.fill} stroke-card"
						stroke-width="1.5"
					/>
				{/each}
			{/if}
		</svg>

		{#if hovered !== null}
			{@const onRight = x(hovered) > width / 2}
			<div
				class="bg-popover text-popover-foreground pointer-events-none absolute top-1 z-10 grid min-w-36 gap-1 rounded-md border px-2.5 py-2 text-xs shadow-md"
				style:left="{x(hovered)}px"
				style:transform="translateX({onRight ? 'calc(-100% - 10px)' : '10px'})"
				aria-hidden="true"
			>
				<p class="font-medium tabular-nums">
					{formatTime(stepTime(hovered), stepSeconds < 60)} – {formatTime(
						stepTime(hovered + 1),
						stepSeconds < 60
					)}
				</p>
				{#each trafficSeries as series (series.kind)}
					<p class="flex items-center gap-2">
						<span class="size-2 rounded-full {series.swatch}"></span>
						<span class="text-muted-foreground flex-1">{series.label}</span>
						<span class="font-medium tabular-nums">{counts[hovered][series.kind]}</span>
					</p>
				{/each}
			</div>
		{/if}
	{:else}
		<div style:height="{height}px"></div>
	{/if}
</div>
