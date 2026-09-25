<!--
  The on/off switch for all endpoints. When off, fapi ignores them and passes
  every request to the real API (or answers 404).
-->
<script lang="ts">
	import { Switch } from '$lib/components/ui/switch';
	import { Label } from '$lib/components/ui/label';

	type Props = {
		enabled: boolean;
		onChange: (enabled: boolean) => void;
	};
	let { enabled, onChange }: Props = $props();
</script>

<div class="flex items-start gap-3">
	<Switch id="enabled" class="mt-0.5" checked={enabled} onCheckedChange={onChange} />
	<Label for="enabled" class="grid gap-0.5 leading-snug">
		Proxy service
		<span class="text-muted-foreground text-xs font-normal">
			When off, every request goes to the real API
		</span>
	</Label>
</div>
