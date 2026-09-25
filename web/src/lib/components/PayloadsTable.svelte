<!--
  The saved payloads, with a preview of each body.
-->
<script lang="ts">
	import type { Payload } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Table from '$lib/components/ui/table';

	type Props = {
		payloads: Payload[];
		/** The payload being changed, whose row is highlighted. */
		editingId: string | null;
		onEdit: (payload: Payload) => void;
		onDelete: (id: string) => void;
	};
	let { payloads, editingId, onEdit, onDelete }: Props = $props();
</script>

<Table.Root class="table-fixed">
	<Table.Header>
		<Table.Row>
			<Table.Head class="w-1/4">Name</Table.Head>
			<Table.Head class="w-20">Status</Table.Head>
			<Table.Head>Body</Table.Head>
			<Table.Head class="w-40"><span class="sr-only">Actions</span></Table.Head>
		</Table.Row>
	</Table.Header>
	<Table.Body>
		{#each payloads as payload (payload.id)}
			<Table.Row data-state={payload.id === editingId ? 'selected' : undefined}>
				<Table.Cell class="font-medium break-words whitespace-normal">{payload.name}</Table.Cell>
				<Table.Cell>{payload.status}</Table.Cell>
				<Table.Cell class="text-muted-foreground truncate font-mono text-xs">
					{payload.body === undefined ? 'No body' : JSON.stringify(payload.body)}
				</Table.Cell>
				<Table.Cell class="text-right">
					<Button variant="outline" size="sm" onclick={() => onEdit(payload)}>Edit</Button>
					<Button variant="outline" size="sm" onclick={() => onDelete(payload.id)}>Delete</Button>
				</Table.Cell>
			</Table.Row>
		{:else}
			<Table.Row>
				<Table.Cell colspan={4} class="text-muted-foreground">No saved payloads yet</Table.Cell>
			</Table.Row>
		{/each}
	</Table.Body>
</Table.Root>
