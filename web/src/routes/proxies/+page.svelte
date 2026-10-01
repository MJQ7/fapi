<!--
  The Proxies screen: which real API port each fapi port forwards to. A port
  can have several proxies, but only one is on at a time.
-->
<script lang="ts">
	import { app } from '$lib/app-state.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ProxiesTable from '$lib/components/ProxiesTable.svelte';
	import ProxyForm from '$lib/components/ProxyForm.svelte';
	import * as Card from '$lib/components/ui/card';

	const defaultHost = $derived(app.settings?.passThrough.host ?? 'localhost');

	function added() {
		app.showErrors(app.loadMocks());
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
			<Card.Description>
				A fapi port can have several proxies, but only one is on at a time. A new proxy is turned
				on, and any other on its port off.
			</Card.Description>
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
			<Card.Description>Turning a proxy on turns off the others on its port.</Card.Description>
		</Card.Header>
		<Card.Content>
			<ProxiesTable
				proxies={app.proxies ?? []}
				onSetEnabled={app.setProxyEnabled}
				onDelete={app.deleteProxy}
			/>
		</Card.Content>
	</Card.Root>
{:else}
	<p class="text-muted-foreground">Proxies are turned off in the fapi config (passThrough).</p>
{/if}
