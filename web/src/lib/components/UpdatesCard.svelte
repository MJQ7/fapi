<!--
  The Updates section of the Settings screen: whether GitHub has a newer
  release of fapi, and how to install it. fapi only asks GitHub when this
  is shown (and remembers the answer for an hour), or when "Check now" is
  pressed.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import * as api from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import CircleCheckIcon from '@lucide/svelte/icons/circle-check';
	import ExternalLinkIcon from '@lucide/svelte/icons/external-link';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';

	let check = $state<api.UpdateCheck | null>(null);
	let checking = $state(false);
	// An error from fapi itself, as opposed to a failed check (check.error).
	let error = $state('');

	async function run(force: boolean) {
		checking = true;
		error = '';
		try {
			check = await api.checkForUpdates(force);
		} catch (caught) {
			error = (caught as Error).message;
		} finally {
			checking = false;
		}
	}

	onMount(() => {
		run(false);
	});

	function formatDate(time: string): string {
		return new Date(time).toLocaleDateString(undefined, { dateStyle: 'medium' });
	}

	function formatDateTime(time: string): string {
		return new Date(time).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
	}
</script>

<Card.Root class={check?.updateAvailable ? 'border-warning/50' : ''}>
	<Card.Header>
		<Card.Title>Updates</Card.Title>
		<Card.Description>
			fapi checks GitHub for a newer release when you open this screen.
		</Card.Description>
	</Card.Header>
	<Card.Content class="grid gap-(--form-gap) text-sm">
		{#if error || check?.error}
			<p class="text-warning flex items-start gap-2" role="alert">
				<TriangleAlertIcon class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
				<span>{error || check?.error}</span>
			</p>
		{:else if !check}
			<p class="text-muted-foreground">Checking GitHub…</p>
		{:else if check.updateAvailable}
			<div class="grid gap-2">
				<p class="text-base font-semibold">fapi {check.latest} is available</p>
				<p class="text-muted-foreground">
					You have {check.current}.
					{#if check.publishedAt}Released {formatDate(check.publishedAt)}.{/if}
				</p>
			</div>
			<div>
				<Button href={check.latestUrl} target="_blank" rel="noopener noreferrer">
					View the release <ExternalLinkIcon />
				</Button>
			</div>
			<div class="grid gap-2">
				<p class="font-medium">To update, download the new version from the release page:</p>
				<ul class="text-muted-foreground grid list-disc gap-1 pl-5">
					<li>
						<strong class="text-foreground">Debian, Ubuntu:</strong> run
						<code class="bg-muted rounded px-1 py-0.5 break-all"
							>sudo apt install ./fapi_{check.latest}_linux_amd64.deb</code
						>
						(use the file for your edition). The service restarts by itself.
					</li>
					<li>
						<strong class="text-foreground">Fedora, RHEL:</strong> run
						<code class="bg-muted rounded px-1 py-0.5 break-all"
							>sudo dnf install ./fapi_{check.latest}_linux_amd64.rpm</code
						>
						(use the file for your edition). The service restarts by itself.
					</li>
					<li>
						<strong class="text-foreground">Windows:</strong> stop fapi, unzip the new .exe over the
						old one and start it again.
					</li>
					<li>
						<strong class="text-foreground">Docker:</strong> run the image tagged {check.latest}.
					</li>
				</ul>
				<p class="text-muted-foreground">
					Your endpoints, proxies, payloads and settings are kept.
				</p>
			</div>
		{:else if check.developmentBuild}
			<p>
				This is a development build ({check.current}), so fapi can't tell whether it's older than
				the latest release{check.latest ? `, ${check.latest}` : ''}.
			</p>
		{:else if !check.latest}
			<p class="text-muted-foreground">There are no releases on GitHub yet.</p>
		{:else}
			<p class="flex items-center gap-2">
				<CircleCheckIcon class="text-primary size-4 shrink-0" aria-hidden="true" />
				You have the latest version, {check.current}.
			</p>
		{/if}

		<div class="flex flex-wrap items-center gap-3">
			<Button variant="outline" size="sm" disabled={checking} onclick={() => run(true)}>
				{checking ? 'Checking…' : 'Check now'}
			</Button>
			{#if check}
				<span class="text-muted-foreground">Last checked {formatDateTime(check.checkedAt)}</span>
			{/if}
		</div>
	</Card.Content>
</Card.Root>
