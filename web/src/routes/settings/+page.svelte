<!--
  The Settings screen: fapi's ports, and whether there's a newer fapi. fapi
  reads its settings when it starts, so saved port changes are shown as
  waiting for a restart until then.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import * as api from '$lib/api';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import PortSettingsForm from '$lib/components/PortSettingsForm.svelte';
	import UpdatesCard from '$lib/components/UpdatesCard.svelte';
	import * as Card from '$lib/components/ui/card';

	let settings = $state<api.PortSettings | null>(null);
	let error = $state('');

	onMount(async () => {
		try {
			settings = await api.getPortSettings();
		} catch (caught) {
			error = (caught as Error).message;
		}
	});

	// Where the web UI will be once fapi restarts with a new admin port.
	const newAddress = $derived(
		settings && settings.saved.adminPort !== settings.active.adminPort
			? `http://${location.hostname}:${settings.saved.adminPort}`
			: ''
	);
</script>

<PageHeader
	title="Settings"
	description="fapi's ports and updates. Port changes are saved to its settings file and take effect when fapi restarts."
/>

{#if error}
	<p class="text-destructive text-sm" role="alert">{error}</p>
{/if}

{#if settings}
	{#if settings.restartNeeded}
		<Card.Root class="border-warning/50">
			<Card.Header>
				<Card.Title>Restart fapi to use the new ports</Card.Title>
				<Card.Description>
					Running now: admin port {settings.active.adminPort}, default mock port
					{settings.active.mockPorts.default}, mock ports {settings.active.mockPorts.min} to
					{settings.active.mockPorts.max}.
				</Card.Description>
			</Card.Header>
			<Card.Content class="grid gap-2 text-sm">
				<p>
					If fapi runs as the Linux service, run
					<code class="bg-muted rounded px-1 py-0.5">sudo systemctl restart fapi</code>. Otherwise stop
					it and run <code class="bg-muted rounded px-1 py-0.5">fapi serve</code> again.
				</p>
				{#if newAddress}
					<p>The web UI will then be at <a class="underline" href={newAddress}>{newAddress}</a>.</p>
				{/if}
			</Card.Content>
		</Card.Root>
	{/if}

	<Card.Root>
		<Card.Header>
			<Card.Title>Ports</Card.Title>
			<Card.Description>
				Saved in <code class="bg-muted rounded px-1 py-0.5 break-all">{settings.file}</code>.
			</Card.Description>
		</Card.Header>
		<Card.Content>
			<PortSettingsForm saved={settings.saved} onSaved={(saved) => (settings = saved)} />
		</Card.Content>
	</Card.Root>
{/if}

<UpdatesCard />
