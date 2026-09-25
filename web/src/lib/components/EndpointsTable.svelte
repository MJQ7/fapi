<!--
  The endpoints, with how many requests each one answered (the Traffic
  column). Clicking the count shows those requests under the row. Each
  endpoint has an on/off switch; an endpoint that's off is kept, but its
  requests go to the proxy instead.
  An endpoint on a port without a proxy that's on is marked with a warning,
  since other requests to its port get a 404.
-->
<script lang="ts">
	import { SvelteSet } from 'svelte/reactivity';
	import type { LoggedRequest, Mock, Proxy } from '$lib/api';
	import { countRequests } from '$lib/format';
	import RequestList from './RequestList.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Switch } from '$lib/components/ui/switch';
	import * as Table from '$lib/components/ui/table';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';

	type Props = {
		mocks: Mock[];
		/** The request log, or null when it's turned off in fapi's settings. */
		requests: LoggedRequest[] | null;
		/** The saved proxies, or null when proxies are turned off in fapi's settings. */
		proxies: Proxy[] | null;
		onSetEnabled: (id: string, enabled: boolean) => void;
		onDelete: (id: string) => void;
	};
	let { mocks, requests, proxies, onSetEnabled, onDelete }: Props = $props();

	/**
	 * Why other requests to port get a 404, or '' when a proxy that's on
	 * forwards them (or proxies are turned off, so there's nothing to warn about).
	 */
	function proxyWarning(port: number): string {
		if (proxies === null) {
			return '';
		}
		const proxy = proxies.find((saved) => saved.port === port);
		if (!proxy) {
			return `No proxy found for port ${port}.`;
		}
		if (!proxy.enabled) {
			return `The proxy for port ${port} is off.`;
		}
		return '';
	}

	// The endpoints whose requests are shown. A SvelteSet updates the page
	// when it changes, which a plain Set doesn't.
	const expanded = new SvelteSet<string>();

	function requestsFor(mock: Mock): LoggedRequest[] {
		return (requests ?? []).filter((request) => request.mockId === mock.id);
	}

	function toggle(id: string) {
		if (expanded.has(id)) {
			expanded.delete(id);
		} else {
			expanded.add(id);
		}
	}

	// The warning and switch columns come first; the requests column is only
	// there with the log.
	const columnCount = $derived(requests === null ? 7 : 8);
</script>

<Tooltip.Provider>
	<Table.Root>
		<Table.Header>
			<Table.Row>
				<Table.Head class="w-8"><span class="sr-only">Proxy</span></Table.Head>
				<Table.Head class="w-12"></Table.Head>
				<Table.Head>Port</Table.Head>
				<Table.Head>Method</Table.Head>
				<Table.Head>Path</Table.Head>
				<Table.Head>Status</Table.Head>
				{#if requests !== null}
					<Table.Head>Traffic</Table.Head>
				{/if}
				<Table.Head><span class="sr-only">Actions</span></Table.Head>
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each mocks as mock (mock.id)}
				{@const received = requestsFor(mock)}
				{@const open = expanded.has(mock.id) && received.length > 0}
				{@const warning = proxyWarning(mock.port)}
				<Table.Row class={mock.disabled ? 'text-muted-foreground' : ''}>
					<Table.Cell class="pr-0">
						{#if warning}
							<Tooltip.Root>
								<Tooltip.Trigger class="flex text-warning" aria-label={warning}>
									<TriangleAlertIcon class="size-4" aria-hidden="true" />
								</Tooltip.Trigger>
								<Tooltip.Content>{warning}</Tooltip.Content>
							</Tooltip.Root>
						{/if}
					</Table.Cell>
					<Table.Cell>
						<Switch
							checked={!mock.disabled}
							onCheckedChange={(checked) => onSetEnabled(mock.id, checked)}
							aria-label="{mock.method} {mock.path} on port {mock.port} on"
						/>
					</Table.Cell>
					<Table.Cell>{mock.port}</Table.Cell>
					<Table.Cell>{mock.method}</Table.Cell>
					<Table.Cell class="font-mono break-all whitespace-normal">{mock.path}</Table.Cell>
					<Table.Cell>{mock.status}</Table.Cell>
					{#if requests !== null}
						<Table.Cell>
							{#if received.length === 0}
								<span class="text-muted-foreground">None yet</span>
							{:else}
								<Button
									variant="link"
									class="h-auto p-0"
									aria-expanded={open}
									onclick={() => toggle(mock.id)}
								>
									{countRequests(received.length)}
									{open ? '▾' : '▸'}
								</Button>
							{/if}
						</Table.Cell>
					{/if}
					<Table.Cell class="text-right">
						<Button variant="outline" size="sm" onclick={() => onDelete(mock.id)}>Delete</Button>
					</Table.Cell>
				</Table.Row>
				{#if open}
					<Table.Row class="hover:bg-transparent">
						<Table.Cell colspan={columnCount} class="bg-muted/40 p-0 whitespace-normal">
							<RequestList requests={received} />
						</Table.Cell>
					</Table.Row>
				{/if}
			{:else}
				<Table.Row>
					<Table.Cell></Table.Cell>
					<Table.Cell colspan={columnCount - 1} class="text-muted-foreground">No endpoints yet</Table.Cell>
				</Table.Row>
			{/each}
		</Table.Body>
	</Table.Root>
</Tooltip.Provider>
