<!--
  The Override an endpoint screen: the form for adding an endpoint. After
  adding one, it goes to the Endpoints screen to show it.
-->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { app } from '$lib/app-state.svelte';
	import AddEndpointForm from '$lib/components/AddEndpointForm.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import * as Card from '$lib/components/ui/card';

	async function added() {
		await app.showErrors(app.loadMocks());
		await goto('/');
	}
</script>

<PageHeader
	title="Override an endpoint"
	description="Requests to this port's endpoint get your response instead of reaching the real API."
/>

<Card.Root>
	<Card.Content>
		<AddEndpointForm
			payloads={app.payloads}
			proxies={app.proxies}
			defaultPort={app.settings?.mockPorts.default ?? 3001}
			onAdded={added}
		/>
	</Card.Content>
</Card.Root>
