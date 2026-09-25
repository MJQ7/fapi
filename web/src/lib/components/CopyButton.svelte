<!--
  A small icon button that copies text to the clipboard, and shows a tick for
  a moment once it has. Its label is the tooltip and what screen readers say.
-->
<script lang="ts">
	import type { Component } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { cn } from '$lib/utils';
	import CheckIcon from '@lucide/svelte/icons/check';
	import CopyIcon from '@lucide/svelte/icons/copy';

	type Props = {
		text: string;
		label: string; // what's copied, such as "Copy the request body"
		icon?: Component; // the copy icon unless given
		class?: string; // colours, to tell buttons side by side apart
	};
	let { text, label, icon: Icon = CopyIcon, class: className }: Props = $props();

	let copied = $state(false);
	let timer: ReturnType<typeof setTimeout> | undefined;

	async function copy() {
		try {
			await navigator.clipboard.writeText(text);
		} catch {
			copyWithSelection();
		}
		copied = true;
		clearTimeout(timer);
		timer = setTimeout(() => (copied = false), 1500);
	}

	// The clipboard API only works on secure pages; http://127.0.0.1 is one,
	// but fapi opened by a network address isn't. The older way works there.
	function copyWithSelection() {
		const area = document.createElement('textarea');
		area.value = text;
		area.style.position = 'fixed';
		area.style.opacity = '0';
		document.body.append(area);
		area.select();
		document.execCommand('copy');
		area.remove();
	}
</script>

<Button
	variant="ghost"
	size="icon-xs"
	class={cn('size-7', className)}
	aria-label={copied ? 'Copied' : label}
	title={label}
	onclick={copy}
>
	{#if copied}<CheckIcon class="size-3.5" />{:else}<Icon class="size-3.5" />{/if}
</Button>
