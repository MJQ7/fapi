<!--
  The form for fapi's port settings. fapi checks the values and saves them to
  its settings file; its message is shown here if it refuses them.
-->
<script lang="ts">
	import { untrack } from 'svelte';
	import * as api from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';

	type Props = {
		/** The values saved for fapi's next start, which the form starts from. */
		saved: api.Ports;
		onSaved: (settings: api.PortSettings) => void;
	};
	let { saved, onSaved }: Props = $props();

	// untrack: start from the saved values, without resetting what's being
	// typed when they're reloaded.
	let adminPort = $state<number | null>(untrack(() => saved.adminPort));
	let min = $state<number | null>(untrack(() => saved.mockPorts.min));
	let max = $state<number | null>(untrack(() => saved.mockPorts.max));
	let defaultPort = $state<number | null>(untrack(() => saved.mockPorts.default));
	let error = $state('');
	let saving = $state(false);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		if (adminPort === null || min === null || max === null || defaultPort === null) {
			error = 'Fill in every port';
			return;
		}

		saving = true;
		try {
			const result = await api.savePortSettings({
				adminPort,
				mockPorts: { min, max, default: defaultPort }
			});
			onSaved(result);
		} catch (caught) {
			error = (caught as Error).message;
		} finally {
			saving = false;
		}
	}
</script>

<form class="grid gap-(--form-gap)" onsubmit={submit}>
	<div class="grid gap-2">
		<Label for="admin-port">Admin port</Label>
		<Input id="admin-port" type="number" class="w-32" required bind:value={adminPort} />
		<p class="text-muted-foreground text-sm">
			The port of this web UI and of the admin API the terminal UI uses.
		</p>
	</div>

	<div class="grid gap-2">
		<Label for="default-port">Default mock port</Label>
		<Input id="default-port" type="number" class="w-32" required bind:value={defaultPort} />
		<p class="text-muted-foreground text-sm">
			The fapi port the forms suggest when you add an endpoint or a proxy.
		</p>
	</div>

	<div class="grid gap-2">
		<span class="text-sm font-medium">Allowed mock ports</span>
		<div class="flex flex-wrap items-center gap-2">
			<Input
				type="number"
				class="w-32"
				aria-label="Lowest allowed mock port"
				required
				bind:value={min}
			/>
			<span class="text-muted-foreground">to</span>
			<Input
				type="number"
				class="w-32"
				aria-label="Highest allowed mock port"
				required
				bind:value={max}
			/>
		</div>
		<p class="text-muted-foreground text-sm">
			Endpoints and proxies can only use ports in this range. Useful with Docker, where ports must
			be published in advance.
		</p>
	</div>

	{#if error}
		<p class="text-destructive text-sm" role="alert">{error}</p>
	{/if}

	<div>
		<Button type="submit" disabled={saving}>Save</Button>
	</div>
</form>
