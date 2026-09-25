<!--
  The form for adding a proxy: a fapi port and the real API port its
  requests are forwarded to. The real API is on passThrough.host (usually
  this machine) unless "another host" is ticked and a host is entered.
  Adding a port that already has a proxy changes where it forwards to.
-->
<script lang="ts">
	import * as api from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { Checkbox } from '$lib/components/ui/checkbox';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import ArrowRightIcon from '@lucide/svelte/icons/arrow-right';

	type Props = {
		/** Suggested fapi port: mockPorts.default in fapi's settings. */
		defaultPort: number;
		/** Where the real API is when no host is entered: passThrough.host in fapi's settings. */
		defaultHost: string;
		onAdded: () => void;
	};
	let { defaultPort, defaultHost, onAdded }: Props = $props();

	let port = $state<number | null>(null);
	let otherHost = $state(false);
	let host = $state('');
	let upstreamPort = $state<number | null>(null);
	let error = $state('');
	let saving = $state(false);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		if (port === null || upstreamPort === null) {
			error = 'Enter both ports';
			return;
		}
		if (otherHost && host.trim() === '') {
			error = 'Enter the real API host, or untick "The real API is on another host"';
			return;
		}

		saving = true;
		try {
			await api.setUpstream(port, upstreamPort, otherHost ? host : '');
			port = null;
			upstreamPort = null;
			otherHost = false;
			host = '';
			onAdded();
		} catch (caught) {
			error = (caught as Error).message;
		} finally {
			saving = false;
		}
	}
</script>

<form class="grid gap-(--form-gap)" onsubmit={submit}>
	<div class="flex items-center gap-2">
		<Checkbox id="proxy-other-host" bind:checked={otherHost} />
		<Label for="proxy-other-host">The real API is on another host</Label>
	</div>

	<div class="flex flex-wrap items-end gap-4">
		<div class="grid gap-2">
			<Label for="proxy-port">fapi port</Label>
			<Input
				id="proxy-port"
				type="number"
				class="w-32"
				placeholder={String(defaultPort)}
				required
				bind:value={port}
			/>
		</div>
		<ArrowRightIcon class="text-muted-foreground mb-2.5 size-4" aria-hidden="true" />
		{#if otherHost}
			<div class="grid gap-2">
				<Label for="proxy-host">Real API host</Label>
				<Input
					id="proxy-host"
					class="w-64"
					placeholder="https://api.example.com"
					autocomplete="off"
					spellcheck={false}
					required
					bind:value={host}
				/>
			</div>
		{/if}
		<div class="grid gap-2">
			<Label for="proxy-upstream">Real API port</Label>
			<Input
				id="proxy-upstream"
				type="number"
				class="w-32"
				placeholder={otherHost ? '443' : '3000'}
				required
				bind:value={upstreamPort}
			/>
		</div>
		<Button type="submit" disabled={saving}>Add proxy</Button>
	</div>

	<p class="text-muted-foreground text-sm">
		{#if otherHost}
			A host name or IP address. Start it with https:// if the API uses HTTPS; otherwise HTTP is
			used.
		{:else}
			Requests are forwarded to {defaultHost} over HTTP.
		{/if}
	</p>

	{#if error}
		<p class="text-destructive text-sm" role="alert">{error}</p>
	{/if}
</form>
