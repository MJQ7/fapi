<!--
  The Payloads screen: saved responses to choose from when overriding an
  endpoint. Changing a payload doesn't change endpoints already added from it.
-->
<script lang="ts">
	import * as api from '$lib/api';
	import { app } from '$lib/app-state.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import PayloadForm from '$lib/components/PayloadForm.svelte';
	import PayloadsTable from '$lib/components/PayloadsTable.svelte';
	import * as Card from '$lib/components/ui/card';

	let editing = $state<api.Payload | null>(null);

	function edit(payload: api.Payload) {
		editing = payload;
		window.scrollTo({ top: 0, behavior: 'smooth' });
	}

	function saved() {
		editing = null;
		app.showErrors(app.loadPayloads());
	}

	function remove(id: string) {
		if (editing?.id === id) {
			editing = null;
		}
		app.showErrors(api.deletePayload(id).then(app.loadPayloads));
	}
</script>

<PageHeader
	title="Payloads"
	description="Saved responses you can choose when overriding an endpoint."
/>

<Card.Root>
	<Card.Header>
		<Card.Title>{editing ? `Edit “${editing.name}”` : 'Save a payload'}</Card.Title>
		<Card.Description>
			{editing
				? 'Endpoints already added from this payload keep their response.'
				: 'Give it a name you’ll recognise in the override form.'}
		</Card.Description>
	</Card.Header>
	<Card.Content>
		{#key editing?.id}
			<PayloadForm
				payload={editing ?? undefined}
				onSaved={saved}
				onCancel={editing ? () => (editing = null) : undefined}
			/>
		{/key}
	</Card.Content>
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title>Saved payloads</Card.Title>
	</Card.Header>
	<Card.Content>
		<PayloadsTable
			payloads={app.payloads}
			editingId={editing?.id ?? null}
			onEdit={edit}
			onDelete={remove}
		/>
	</Card.Content>
</Card.Root>
