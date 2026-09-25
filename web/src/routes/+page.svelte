<!--
  The Endpoints screen: every overridden endpoint, with the requests each
  one answered.
-->
<script lang="ts">
	import { app } from '$lib/app-state.svelte';
	import EndpointsTable from '$lib/components/EndpointsTable.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import PlusIcon from '@lucide/svelte/icons/plus';
</script>

<PageHeader title="Endpoints" description="Requests to these endpoints get the response you set.">
	{#snippet actions()}
		{#if app.features?.requestLog}
			<Button variant="outline" size="sm" onclick={app.clearRequests}>Clear requests</Button>
		{/if}
		<Button size="sm" href="/override"><PlusIcon />Override an endpoint</Button>
	{/snippet}
</PageHeader>

<Card.Root>
	<Card.Content>
		<EndpointsTable
			mocks={app.data.mocks}
			requests={app.features?.requestLog ? app.requests : null}
			proxies={app.proxies}
			onSetEnabled={app.setMockEnabled}
			onDelete={app.deleteMock}
		/>
	</Card.Content>
</Card.Root>
