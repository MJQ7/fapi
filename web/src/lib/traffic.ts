// The dashboard's three kinds of traffic, shared by its charts and its list,
// so each has the same name and colour everywhere.

import type { LoggedRequest, TrafficCounts } from '$lib/api';

export type TrafficKind = keyof TrafficCounts;

type TrafficSeries = {
	kind: TrafficKind;
	label: string;
	description: string;
	/** Tailwind classes for the kind's colour, written out in full so Tailwind finds them. */
	stroke: string;
	fill: string;
	swatch: string;
};

export const trafficSeries: TrafficSeries[] = [
	{
		kind: 'received',
		label: 'Received',
		description: 'every request to the port',
		stroke: 'stroke-chart-1',
		fill: 'fill-chart-1',
		swatch: 'bg-chart-1'
	},
	{
		kind: 'proxied',
		label: 'Proxied',
		description: 'forwarded to the real API',
		stroke: 'stroke-chart-2',
		fill: 'fill-chart-2',
		swatch: 'bg-chart-2'
	},
	{
		kind: 'sent',
		label: 'Sent',
		description: 'answered by fapi itself: an endpoint or a CORS preflight',
		stroke: 'stroke-chart-3',
		fill: 'fill-chart-3',
		swatch: 'bg-chart-3'
	}
];

/** Whether fapi forwarded a logged request to the real API or sent the response itself. */
export function requestKind(request: LoggedRequest): 'proxied' | 'sent' {
	return request.outcome === 'proxied' ? 'proxied' : 'sent';
}

/** Adds up a port's counts over a report's steps. */
export function totalCounts(counts: TrafficCounts[]): TrafficCounts {
	const total = { received: 0, proxied: 0, sent: 0 };
	for (const step of counts) {
		total.received += step.received;
		total.proxied += step.proxied;
		total.sent += step.sent;
	}
	return total;
}
