<!--
  The form for adding an endpoint. Its port is chosen by picking a proxy, so
  requests the endpoint doesn't match still reach the real API; "No proxy"
  allows any port, with a warning. Choosing a saved payload fills in its
  status and body, which can still be changed. The form checks that the body
  is valid JSON before sending; everything else is checked by fapi, whose
  message is shown here. On wide screens the request and response fields
  sit side by side.
-->
<script lang="ts">
	import { untrack } from 'svelte';
	import * as api from '$lib/api';
	import { bodyText, parseBodyText, sampleBody } from '$lib/statuses';
	import StatusSelect from './StatusSelect.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { NativeSelect, NativeSelectOption } from '$lib/components/ui/native-select';
	import { Textarea } from '$lib/components/ui/textarea';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';

	type Props = {
		payloads: api.Payload[];
		/** The saved proxies, or null when proxies are turned off in fapi's settings. */
		proxies: api.Proxy[] | null;
		/** The port to start with: mockPorts.default in fapi's settings. */
		defaultPort: number;
		onAdded: () => void;
	};
	let { payloads, proxies, defaultPort, onAdded }: Props = $props();

	/** The proxy choice meaning "no proxy: type any port". */
	const noProxy = 'none';

	// The chosen proxy's port, as text because select values are text, or
	// noProxy. null until the user chooses, so the default follows the
	// proxies as they load: the proxy on the default port, or else the first.
	let chosenProxy = $state<string | null>(null);
	const proxyChoice = $derived(
		chosenProxy ??
			String(
				proxies?.find((proxy) => proxy.port === defaultPort)?.port ?? proxies?.[0]?.port ?? noProxy
			)
	);
	const choosingProxy = $derived(proxies !== null && proxies.length > 0);

	// The port typed when there's no proxy to choose. untrack: start from the
	// default, but don't reset the field if the setting changes while the
	// form is open.
	let otherPort = $state(untrack(() => defaultPort));
	const port = $derived(
		choosingProxy && proxyChoice !== noProxy ? Number(proxyChoice) : otherPort
	);
	// The proxy on the chosen port, if it has one. Warn when it has none or
	// it's off, but only when proxies are turned on; otherwise no port has one.
	const portProxy = $derived(proxies?.find((proxy) => proxy.port === port));
	const unproxied = $derived(proxies !== null && !portProxy?.enabled);
	let method = $state<api.Method>('GET');
	let path = $state('');
	let status = $state(400);
	let body = $state(sampleBody(400));
	// The saved payload the status and body came from, or '' once they're
	// changed by hand.
	let payloadId = $state('');
	let error = $state('');
	let saving = $state(false);

	function choosePayload(id: string) {
		payloadId = id;
		const payload = payloads.find((saved) => saved.id === id);
		if (payload) {
			status = payload.status;
			body = bodyText(payload.body);
		}
	}

	// Choosing a status fills in a sample body for it.
	function chooseStatus(code: number) {
		payloadId = '';
		status = code;
		body = sampleBody(code);
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';

		let parsedBody: unknown;
		try {
			parsedBody = parseBodyText(body);
		} catch (caught) {
			error = (caught as Error).message;
			return;
		}

		saving = true;
		try {
			await api.addMock({ port, method, path, status, body: parsedBody });
			onAdded();
		} catch (caught) {
			error = (caught as Error).message;
		} finally {
			saving = false;
		}
	}
</script>

<form class="grid gap-(--form-gap)" onsubmit={submit}>
	<div class="grid gap-(--form-gap) lg:grid-cols-2 lg:gap-x-(--form-column-gap)">
		<fieldset class="grid min-w-0 content-start gap-(--form-gap)">
			<legend class="mb-(--form-gap) text-base font-semibold">Request</legend>
			<div class="grid gap-2">
				<div class="flex flex-wrap items-end gap-4">
					{#if choosingProxy}
						<div class="grid gap-2">
							<Label for="proxy">Proxy</Label>
							<NativeSelect
								id="proxy"
								class="w-full sm:w-96"
								value={proxyChoice}
								onchange={(event) => (chosenProxy = event.currentTarget.value)}
							>
								{#each proxies ?? [] as proxy (proxy.port)}
									<NativeSelectOption value={String(proxy.port)}>
										{proxy.port} → {proxy.realAPI}{proxy.enabled ? '' : ' (off)'}
									</NativeSelectOption>
								{/each}
								<NativeSelectOption value={noProxy}>No proxy: choose a port</NativeSelectOption>
							</NativeSelect>
						</div>
					{/if}
					{#if !choosingProxy || proxyChoice === noProxy}
						<div class="grid gap-2">
							<Label for="port">Port</Label>
							<Input id="port" type="number" class="w-32" required bind:value={otherPort} />
						</div>
					{/if}
				</div>
				{#if unproxied}
					<p class="text-warning flex items-start gap-2 text-sm">
						<TriangleAlertIcon class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
						<span>
							{#if portProxy}
								The proxy for port {port} is off. The endpoint still works, but traffic to this port will
								not be forwarded anywhere.
								<a href="/proxies" class="underline underline-offset-4">Turn it on</a> to forward that
								traffic to the real API.
							{:else}
								No proxy found for port {port}. The endpoint still works, but traffic to this port will
								not be forwarded anywhere.
								<a href="/proxies" class="underline underline-offset-4">Add a proxy</a> to send them to
								the real API.
							{/if}
						</span>
					</p>
				{/if}
			</div>

			<div class="grid gap-4 sm:grid-cols-[9rem_1fr]">
				<div class="grid gap-2">
					<Label for="method">Method</Label>
					<NativeSelect id="method" class="w-full" bind:value={method}>
						{#each api.methods as option (option)}
							<NativeSelectOption value={option}>{option}</NativeSelectOption>
						{/each}
					</NativeSelect>
				</div>
				<div class="grid gap-2">
					<Label for="path">Path</Label>
					<Input id="path" placeholder="/api/users/:id" required bind:value={path} />
				</div>
			</div>
		</fieldset>

		<fieldset class="grid min-w-0 content-start gap-(--form-gap)">
			<legend class="mb-(--form-gap) text-base font-semibold">Response</legend>
			<div class="grid gap-2">
				<Label for="payload">Saved payload</Label>
				{#if payloads.length > 0}
					<NativeSelect
						id="payload"
						class="w-full sm:w-80"
						value={payloadId}
						onchange={(event) => choosePayload(event.currentTarget.value)}
					>
						<NativeSelectOption value="">None: write the response below</NativeSelectOption>
						{#each payloads as payload (payload.id)}
							<NativeSelectOption value={payload.id}>
								{payload.name} ({payload.status})
							</NativeSelectOption>
						{/each}
					</NativeSelect>
				{:else}
					<p class="text-muted-foreground text-sm">
						No saved payloads yet.
						<a href="/payloads" class="text-foreground underline underline-offset-4">Save one</a>
						to reuse a response without typing its JSON each time.
					</p>
				{/if}
			</div>

			<div class="grid gap-2">
				<Label for="status">Response status</Label>
				<StatusSelect id="status" {status} onChange={chooseStatus} />
			</div>

			<div class="grid gap-2">
				<Label for="body">Response body (JSON, optional)</Label>
				<Textarea
					id="body"
					class="min-h-44 font-mono text-sm"
					bind:value={body}
					oninput={() => (payloadId = '')}
				/>
			</div>
		</fieldset>
	</div>

	{#if error}
		<p class="text-destructive text-sm" role="alert">{error}</p>
	{/if}

	<div>
		<Button type="submit" disabled={saving}>Override endpoint</Button>
	</div>
</form>
