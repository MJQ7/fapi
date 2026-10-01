<!--
  The saved proxies, in port order. Each has an on/off switch. Only one proxy
  per port is on: turning one on turns the others on its port off (fapi does
  that). A proxy that's off is kept; when none on its port is on, the port's
  unmatched requests get a 404 instead of being forwarded.
-->
<script lang="ts">
	import type { Proxy } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { Switch } from '$lib/components/ui/switch';
	import * as Table from '$lib/components/ui/table';

	type Props = {
		proxies: Proxy[];
		onSetEnabled: (id: string, enabled: boolean) => void;
		onDelete: (id: string) => void;
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
		{#each proxies as proxy (proxy.id)}
			<Table.Row class={proxy.enabled ? '' : 'text-muted-foreground'}>
				<Table.Cell>
					<Switch
						checked={proxy.enabled}
						onCheckedChange={(checked) => onSetEnabled(proxy.id, checked)}
						aria-label="Proxy from port {proxy.port} to {proxy.realAPI} on"
					/>
				</Table.Cell>
				<Table.Cell>{proxy.port}</Table.Cell>
				<Table.Cell class="break-all">{proxy.realAPI}</Table.Cell>
				<Table.Cell class="text-right">
					<Button variant="outline" size="sm" onclick={() => onDelete(proxy.id)}>Delete</Button>
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
