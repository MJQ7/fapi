<!--
  The Updates section of the Settings screen: how fapi was installed,
  whether GitHub has a newer release, and a button to install it. fapi only
  asks GitHub when this is shown (and remembers the answer for an hour), or
  when "Check now" is pressed.

  fapi itself downloads the update, on the machine it runs on (so a fapi in
  WSL saves it in WSL, not in Windows), checks it, and installs it where it
  can: the .deb and .rpm through their root helper, and the Windows .exe by
  replacing itself. It then restarts, and this page reloads once the new
  version answers. Where it can't install, it downloads the update and
  shows the command that installs it. In Docker, it only says to update the
  container.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import * as api from '$lib/api';
	import CopyButton from './CopyButton.svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Card from '$lib/components/ui/card';
	import CircleCheckIcon from '@lucide/svelte/icons/circle-check';
	import DownloadIcon from '@lucide/svelte/icons/download';
	import ExternalLinkIcon from '@lucide/svelte/icons/external-link';
	import LoaderCircleIcon from '@lucide/svelte/icons/loader-circle';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';

	/** How long to wait for fapi to come back as the new version. */
	const restartTimeoutMilliseconds = 5 * 60 * 1000;

	let check = $state<api.UpdateCheck | null>(null);
	let checking = $state(false);
	// An error from fapi itself, as opposed to a failed check (check.error).
	let error = $state('');

	// The update being installed, or how the last one went.
	let job = $state<api.UpdateJob | null>(null);
	// "Download and install" was pressed, and is waiting to be confirmed.
	let confirming = $state(false);
	// fapi stopped answering while restarting as the new version.
	let restarting = $state(false);

	const busy = $derived(
		restarting ||
			job?.state === 'downloading' ||
			job?.state === 'installing' ||
			job?.state === 'restarting'
	);

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

	async function loadJob() {
		try {
			job = await api.getUpdateJob();
		} catch {
			// Shown by run's error, if fapi can't be reached.
		}
	}

	onMount(async () => {
		await Promise.all([run(false), loadJob()]);
		// An update started before this page was opened.
		if (busy && check) {
			follow(check.current);
		}
	});

	async function startUpdate() {
		if (!check) {
			return;
		}
		confirming = false;
		error = '';
		try {
			job = await api.installUpdate(check.latest);
		} catch (caught) {
			error = (caught as Error).message;
			return;
		}
		follow(check.current);
	}

	/**
	 * Follows the update until it's downloaded or has failed, or until fapi
	 * answers as a new version (it restarts after installing), then reloads
	 * the page to load the new version's web UI. fapi doesn't answer while
	 * it restarts, which isn't an error.
	 */
	async function follow(current: string) {
		const deadline = Date.now() + restartTimeoutMilliseconds;
		while (Date.now() < deadline) {
			await new Promise((resolve) => setTimeout(resolve, 700));
			try {
				const status = await api.getStatus();
				if (status.version !== current) {
					location.reload();
					return;
				}
				job = await api.getUpdateJob();
				restarting = false;
			} catch {
				restarting = true;
				continue;
			}
			// idle: fapi restarted as the same version, so the install
			// failed (job.last says why).
			if (job.state === 'failed' || job.state === 'downloaded' || job.state === 'idle') {
				return;
			}
		}
		restarting = false;
		error =
			"fapi hasn't come back as the new version. On Linux, see what happened with: journalctl -u fapi-update -u fapi";
	}

	/** The last install, if it's worth showing: it failed, or installed the running version. */
	const last = $derived(
		job?.state === 'idle' && job.last && (!job.last.ok || job.last.version === check?.current)
			? job.last
			: null
	);

	function describeInstall(check: api.UpdateCheck): string {
		const install = check.install;
		const parts = [`fapi ${check.current}`, `${install.package || 'fapi'} edition`];
		switch (install.type) {
			case 'deb':
				parts.push('Debian package');
				break;
			case 'rpm':
				parts.push('RPM package');
				break;
			case 'docker':
				return [...parts, 'Docker'].join(' · ');
			case 'manual':
				parts.push('installed by hand');
				break;
		}
		const system = install.os === 'windows' ? 'Windows' : install.os === 'linux' ? 'Linux' : install.os;
		parts.push(`${system} ${install.arch}${install.wsl ? ' (WSL)' : ''}`);
		return parts.join(' · ');
	}

	function megabytes(bytes: number): string {
		return (bytes / (1024 * 1024)).toFixed(1);
	}

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
		{#if check}
			<p class="text-muted-foreground">{describeInstall(check)}</p>
		{/if}

		{#if last?.ok}
			<p class="text-muted-foreground">Updated to {last.version} on {formatDateTime(last.time)}.</p>
		{:else if last}
			<p class="text-warning flex items-start gap-2" role="alert">
				<TriangleAlertIcon class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
				<span>The update to {last.version} failed: {last.error}</span>
			</p>
		{/if}

		{#if error || check?.error}
			<p class="text-warning flex items-start gap-2" role="alert">
				<TriangleAlertIcon class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
				<span>{error || check?.error}</span>
			</p>
		{/if}

		{#if !check}
			{#if !error}
				<p class="text-muted-foreground">Checking GitHub…</p>
			{/if}
		{:else if check.updateAvailable && check.install.type === 'docker'}
			<div class="grid gap-2">
				<p class="text-base font-semibold">fapi {check.latest} is available</p>
				<p class="text-muted-foreground">
					fapi runs in a container. To get a new version, update the container.
				</p>
			</div>
		{:else if check.updateAvailable}
			{@const install = check.install}
			<div class="grid gap-2">
				<p class="text-base font-semibold">fapi {check.latest} is available</p>
				<p class="text-muted-foreground">
					You have {check.current}.
					{#if check.publishedAt}Released {formatDate(check.publishedAt)}.{/if}
				</p>
			</div>

			{#if busy}
				<div class="grid gap-2" role="status">
					<p class="flex items-center gap-2 font-medium">
						<LoaderCircleIcon class="size-4 shrink-0 animate-spin" aria-hidden="true" />
						{#if restarting || job?.state === 'restarting'}
							Restarting fapi as {check.latest}…
						{:else if job?.state === 'installing'}
							Installing fapi {check.latest}…
						{:else}
							Downloading {check.asset}…
						{/if}
					</p>
					{#if job?.state === 'downloading' && job.total > 0}
						<div class="bg-muted h-2 overflow-hidden rounded-full">
							<div
								class="bg-primary h-full transition-[width]"
								style:width="{(100 * job.downloaded) / job.total}%"
							></div>
						</div>
						<p class="text-muted-foreground tabular-nums">
							{megabytes(job.downloaded)} of {megabytes(job.total)} MB
						</p>
					{/if}
				</div>
			{:else if job?.state === 'downloaded' && job.file}
				<div class="grid gap-2">
					<p>
						Downloaded to
						<code class="bg-muted rounded px-1 py-0.5 break-all">{job.file}</code>{install.wsl
							? ', in WSL'
							: ''}.
					</p>
					{#if job.command}
						<p>Install it with:</p>
						<div class="flex items-start gap-2">
							<code class="bg-muted min-w-0 flex-1 rounded px-2 py-1.5 break-all">{job.command}</code>
							<CopyButton text={job.command} label="Copy the command" />
						</div>
						<p class="text-muted-foreground">
							The package restarts the fapi service by itself. Your endpoints, proxies, payloads
							and settings are kept.
						</p>
					{:else}
						<p class="text-muted-foreground">
							Stop fapi, unzip fapi.exe from it over the old one, and start fapi again. Your
							endpoints, proxies, payloads and settings are kept.
						</p>
					{/if}
				</div>
			{:else}
				{#if job?.state === 'failed'}
					<p class="text-warning flex items-start gap-2" role="alert">
						<TriangleAlertIcon class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
						<span>{job.error}</span>
					</p>
				{/if}

				{#if confirming}
					<div class="bg-muted/50 grid gap-3 rounded-md border p-3">
						<p>
							fapi will download {check.asset}, install it and restart. Requests arriving while it
							restarts get no answer.
							{#if install.type === 'windows'}
								If fapi is running in a console window, the window closes and fapi carries on in
								the background.
							{/if}
						</p>
						<div class="flex flex-wrap gap-2">
							<Button onclick={startUpdate}>Install and restart</Button>
							<Button variant="outline" onclick={() => (confirming = false)}>Cancel</Button>
						</div>
					</div>
				{:else}
					<div class="flex flex-wrap gap-2">
						{#if install.canInstall}
							<Button onclick={() => (confirming = true)}>
								<DownloadIcon /> Download and install
							</Button>
						{:else if install.canDownload}
							<Button onclick={startUpdate}><DownloadIcon /> Download</Button>
						{/if}
						<Button variant="outline" href={check.latestUrl} target="_blank" rel="noopener noreferrer">
							View the release <ExternalLinkIcon />
						</Button>
					</div>
				{/if}

				{#if install.reason}
					<p class="text-muted-foreground">{install.reason}</p>
				{/if}
			{/if}
		{:else if check.developmentBuild}
			<p>
				This is a development build ({check.current}), so fapi can't tell whether it's older than
				the latest release{check.latest ? `, ${check.latest}` : ''}.
			</p>
		{:else if !check.latest}
			{#if !check.error}
				<p class="text-muted-foreground">There are no releases on GitHub yet.</p>
			{/if}
		{:else}
			<p class="flex items-center gap-2">
				<CircleCheckIcon class="text-primary size-4 shrink-0" aria-hidden="true" />
				You have the latest version, {check.current}.
			</p>
		{/if}

		{#if !busy}
			<div class="flex flex-wrap items-center gap-3">
				<Button variant="outline" size="sm" disabled={checking} onclick={() => run(true)}>
					{checking ? 'Checking…' : 'Check now'}
				</Button>
				{#if check}
					<span class="text-muted-foreground">Last checked {formatDateTime(check.checkedAt)}</span>
				{/if}
			</div>
		{/if}
	</Card.Content>
</Card.Root>
