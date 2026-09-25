<!--
  The Proxies screen: which real API port each fapi port forwards to.
-->
<script lang="ts">
	import * as api from '$lib/api';
	import { app } from '$lib/app-state.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ProxiesTable from '$lib/components/ProxiesTable.svelte';
	import ProxyForm from '$lib/components/ProxyForm.svelte';
	import * as Card from '$lib/components/ui/card';

	const defaultHost = $derived(app.settings?.passThrough.host ?? 'localhost');

	function added() {
		app.showErrors(app.loadMocks());
	}

	function remove(port: number) {
		app.showErrors(api.setUpstream(port, null).then(app.loadMocks));
	}
</script>

<PageHeader
	title="Proxies"
	description="Requests to a fapi port that no endpoint overrides are forwarded to the real API."
/>

{#if app.features?.passThrough}
	<Card.Root>
		<Card.Header>
			<Card.Title>Add a proxy</Card.Title>
			<Card.Description>Adding a fapi port that already has a proxy replaces it.</Card.Description>
		</Card.Header>
		<Card.Content>
			<ProxyForm
				defaultPort={app.settings?.mockPorts.default ?? 3001}
				{defaultHost}
				onAdded={added}
			/>
		</Card.Content>
	</Card.Root>

	<Card.Root>
		<Card.Header>
			<Card.Title>Saved proxies</Card.Title>
		</Card.Header>
		<Card.Content>
			<ProxiesTable
				proxies={app.proxies ?? []}
				onSetEnabled={app.setUpstreamEnabled}
				onDelete={remove}
			/>
		</Card.Content>
	</Card.Root>
{:else}
	<p class="text-muted-foreground">Proxies are turned off in the fapi config (passThrough).</p>
{/if}
