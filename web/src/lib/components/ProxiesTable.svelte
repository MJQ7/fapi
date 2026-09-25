<!--
  The saved proxies, one row per fapi port, in port order. Each has an on/off
  switch; a proxy that's off is kept, but its port's unmatched requests get a
  404 instead of being forwarded.
-->
<script lang="ts">
	import type { Proxy } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { Switch } from '$lib/components/ui/switch';
	import * as Table from '$lib/components/ui/table';

	type Props = {
		proxies: Proxy[];
		onSetEnabled: (port: number, enabled: boolean) => void;
		onDelete: (port: number) => void;
	};
	let { proxies, onSetEnabled, onDelete }: Props = $props();
</script>

<Table.Root>
	<Table.Header>
		<Table.Row>
			<Table.Head class="w-12"></Table.Head>
			<Table.Head>fapi port</Table.Head>
			<Table.Head>Real API</Table.Head>
			<Table.Head><span class="sr-only">Actions</span></Table.Head>
		</Table.Row>
	</Table.Header>
	<Table.Body>
		{#each proxies as proxy (proxy.port)}
			<Table.Row class={proxy.enabled ? '' : 'text-muted-foreground'}>
				<Table.Cell>
					<Switch
						checked={proxy.enabled}
						onCheckedChange={(checked) => onSetEnabled(proxy.port, checked)}
						aria-label="Proxy on port {proxy.port} on"
					/>
				</Table.Cell>
				<Table.Cell>{proxy.port}</Table.Cell>
				<Table.Cell class="break-all">{proxy.realAPI}</Table.Cell>
				<Table.Cell class="text-right">
					<Button variant="outline" size="sm" onclick={() => onDelete(proxy.port)}>Delete</Button>
				</Table.Cell>
			</Table.Row>
		{:else}
			<Table.Row>
				<Table.Cell></Table.Cell>
				<Table.Cell colspan={3} class="text-muted-foreground">No proxies yet</Table.Cell>
			</Table.Row>
		{/each}
	</Table.Body>
</Table.Root>
