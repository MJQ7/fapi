<!--
  A menu of response statuses. A status that isn't in the list (one saved
  through the API) is added to the menu, so it still shows.
-->
<script lang="ts">
	import { statuses } from '$lib/statuses';
	import { NativeSelect, NativeSelectOption } from '$lib/components/ui/native-select';

	type Props = {
		id: string;
		status: number;
		onChange: (status: number) => void;
	};
	let { id, status, onChange }: Props = $props();

	const known = $derived(statuses.some((option) => option.code === status));
</script>

<NativeSelect
	{id}
	class="w-full sm:w-80"
	value={String(status)}
	onchange={(event) => onChange(Number(event.currentTarget.value))}
>
	{#if !known}
		<NativeSelectOption value={String(status)}>{status}</NativeSelectOption>
	{/if}
	{#each statuses as option (option.code)}
		<NativeSelectOption value={String(option.code)}>
			{option.code}
			{option.name}
		</NativeSelectOption>
	{/each}
</NativeSelect>
