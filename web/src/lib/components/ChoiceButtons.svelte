<!--
  A row of buttons that choose one of a few options, such as the dashboard's
  period. The chosen one is marked with aria-pressed.
-->
<script lang="ts" generics="T extends string | number">
	import { cn } from '$lib/utils';

	type Props = {
		options: { value: T; label: string }[];
		value: T;
		label: string; // what the buttons choose, for screen readers
		onChange: (value: T) => void;
	};
	let { options, value, label, onChange }: Props = $props();
</script>

<div role="group" aria-label={label} class="bg-muted inline-flex flex-wrap gap-0.5 rounded-lg p-0.5">
	{#each options as option (option.value)}
		{@const chosen = option.value === value}
		<button
			type="button"
			aria-pressed={chosen}
			class={cn(
				'rounded-md px-2.5 py-1 text-sm font-medium whitespace-nowrap transition-colors',
				'focus-visible:ring-ring/50 outline-none focus-visible:ring-3',
				chosen
					? 'bg-background text-foreground shadow-xs'
					: 'text-muted-foreground hover:text-foreground'
			)}
			onclick={() => onChange(option.value)}
		>
			{option.label}
		</button>
	{/each}
</div>
