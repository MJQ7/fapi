<!--
  The form for saving a payload, or changing one when `payload` is given.
  Like the endpoint form, it checks the body is JSON and leaves the rest to
  fapi.
-->
<script lang="ts">
	import * as api from '$lib/api';
	import { bodyText, parseBodyText, sampleBody } from '$lib/statuses';
	import StatusSelect from './StatusSelect.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Textarea } from '$lib/components/ui/textarea';

	type Props = {
		/** The payload being changed, or undefined to save a new one. */
		payload?: api.Payload;
		onSaved: () => void;
		onCancel?: () => void;
	};
	let { payload, onSaved, onCancel }: Props = $props();

	// The form starts from the payload being changed; the parent recreates the
	// form (with {#key}) to change another one.
	// svelte-ignore state_referenced_locally
	let name = $state(payload?.name ?? '');
	// svelte-ignore state_referenced_locally
	let status = $state(payload?.status ?? 400);
	// svelte-ignore state_referenced_locally
	let body = $state(payload ? bodyText(payload.body) : sampleBody(400));
	let error = $state('');
	let saving = $state(false);

	// Choosing a status fills in a sample body for it, unless the body was
	// already written.
	function chooseStatus(code: number) {
		if (body.trim() === '' || body === sampleBody(status)) {
			body = sampleBody(code);
		}
		status = code;
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
			const saved = { name, status, body: parsedBody };
			if (payload) {
				await api.updatePayload(payload.id, saved);
			} else {
				await api.addPayload(saved);
				name = '';
				status = 400;
				body = sampleBody(400);
			}
			onSaved();
		} catch (caught) {
			error = (caught as Error).message;
		} finally {
			saving = false;
		}
	}
</script>

<form class="grid gap-(--form-gap)" onsubmit={submit}>
	<div class="grid gap-2">
		<Label for="payload-name">Name</Label>
		<Input
			id="payload-name"
			class="sm:w-80"
			placeholder="Session expired"
			maxlength={80}
			required
			bind:value={name}
		/>
	</div>

	<div class="grid gap-2">
		<Label for="payload-status">Response status</Label>
		<StatusSelect id="payload-status" {status} onChange={chooseStatus} />
	</div>

	<div class="grid gap-2">
		<Label for="payload-body">Response body (JSON, optional)</Label>
		<Textarea id="payload-body" class="min-h-36 font-mono text-sm" bind:value={body} />
	</div>

	{#if error}
		<p class="text-destructive text-sm" role="alert">{error}</p>
	{/if}

	<div class="flex gap-2">
		<Button type="submit" disabled={saving}>{payload ? 'Save changes' : 'Save payload'}</Button>
		{#if onCancel}
			<Button variant="outline" onclick={onCancel}>Cancel</Button>
		{/if}
	</div>
</form>
