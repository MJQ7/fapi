<!--
  The frame around every screen: a sidebar to move between screens, with the
  switch for all endpoints at its foot. It starts loading the shared data
  (see $lib/app-state.svelte.ts) and shows a screen once the settings arrive.
  On narrow screens the sidebar opens from a menu button instead.
-->
<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import type { Component } from 'svelte';
	import { page } from '$app/state';
	import favicon from '$lib/assets/favicon.svg';
	import { app } from '$lib/app-state.svelte';
	import EnabledSwitch from '$lib/components/EnabledSwitch.svelte';
	import { Button } from '$lib/components/ui/button';
	import { cn } from '$lib/utils';
	import ArrowRightLeftIcon from '@lucide/svelte/icons/arrow-right-left';
	import BracesIcon from '@lucide/svelte/icons/braces';
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
	import ListIcon from '@lucide/svelte/icons/list';
	import MenuIcon from '@lucide/svelte/icons/menu';
	import SettingsIcon from '@lucide/svelte/icons/settings';
	import SplitIcon from '@lucide/svelte/icons/split';
	import XIcon from '@lucide/svelte/icons/x';

	let { children } = $props();

	type Screen = { href: string; label: string; icon: Component; count?: number };

	const screens = $derived<Screen[]>([
		{ href: '/dashboard', label: 'Dashboard', icon: LayoutDashboardIcon },
		{ href: '/', label: 'Endpoints', icon: ListIcon, count: app.data.mocks.length },
		{ href: '/override', label: 'Override an endpoint', icon: SplitIcon },
		{ href: '/payloads', label: 'Payloads', icon: BracesIcon, count: app.payloads.length },
		...(app.features?.passThrough
			? [
					{
						href: '/proxies',
						label: 'Proxies',
						icon: ArrowRightLeftIcon,
						count: Object.keys(app.data.upstreams).length
					}
				]
			: []),
		{ href: '/settings', label: 'Settings', icon: SettingsIcon }
	]);

	// Screens with more to show side by side use more of a wide window.
	const wide = $derived(page.url.pathname === '/dashboard');

	let menuOpen = $state(false);

	onMount(() => {
		app.load();
	});

	$effect(() => app.watchRequests());

	// Close the menu after moving to another screen.
	$effect(() => {
		void page.url.pathname;
		menuOpen = false;
	});
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

{#snippet logo()}
	<span class="flex items-center gap-2.5">
		<span
			class="bg-hero text-hero-foreground grid size-8 place-items-center rounded-lg text-lg leading-none font-bold"
			aria-hidden="true">f</span
		>
		<span class="text-xl font-semibold tracking-tight">fapi</span>
	</span>
{/snippet}

<div class="min-h-dvh md:flex">
	<!-- The top bar, only on narrow screens. -->
	<header class="bg-sidebar flex items-center justify-between border-b px-4 py-2 md:hidden">
		{@render logo()}
		<Button
			variant="ghost"
			size="icon"
			aria-label="Open the menu"
			aria-expanded={menuOpen}
			onclick={() => (menuOpen = true)}
		>
			<MenuIcon />
		</Button>
	</header>

	{#if menuOpen}
		<button
			class="fixed inset-0 z-30 bg-black/40 md:hidden"
			aria-label="Close the menu"
			onclick={() => (menuOpen = false)}
		></button>
	{/if}

	<aside
		class={cn(
			'bg-sidebar text-sidebar-foreground border-sidebar-border fixed inset-y-0 left-0 z-40 flex w-(--sidebar-width) flex-col border-r transition-transform',
			'md:sticky md:top-0 md:h-dvh md:translate-x-0',
			menuOpen ? 'translate-x-0' : '-translate-x-full'
		)}
	>
		<div class="flex items-start justify-between gap-2 px-5 pt-6 pb-6">
			<div class="grid gap-1.5">
				{@render logo()}
				<span class="text-muted-foreground text-sm">An API proxy.</span>
			</div>
			<Button
				variant="ghost"
				size="icon-sm"
				class="md:hidden"
				aria-label="Close the menu"
				onclick={() => (menuOpen = false)}
			>
				<XIcon />
			</Button>
		</div>

		<nav class="grid gap-1 px-3" aria-label="Screens">
			{#each screens as screen (screen.href)}
				{@const current = page.url.pathname === screen.href}
				<a
					href={screen.href}
					aria-current={current ? 'page' : undefined}
					class={cn(
						'group flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium transition-colors',
						'focus-visible:ring-sidebar-ring outline-none focus-visible:ring-2',
						current
							? 'bg-hero-soft text-hero-soft-foreground'
							: 'text-sidebar-foreground/75 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
					)}
				>
					<screen.icon class={cn('size-4 shrink-0', current && 'text-hero')} />
					<span class="flex-1">{screen.label}</span>
					{#if screen.count !== undefined && app.settings}
						<span
							class={cn(
								'text-xs tabular-nums',
								current ? 'text-hero-soft-foreground/75' : 'text-muted-foreground'
							)}>{screen.count}</span
						>
					{/if}
				</a>
			{/each}
		</nav>

		{#if app.settings}
			<div class="border-sidebar-border mt-auto border-t px-5 py-4">
				<EnabledSwitch enabled={app.data.enabled} onChange={app.setEnabled} />
			</div>
		{/if}
	</aside>

	<main class="min-w-0 flex-1">
		<div
			class={cn(
				'mx-auto grid gap-(--section-gap) px-(--page-padding-x) py-(--page-padding-y)',
				wide ? 'max-w-(--wide-content-max-width)' : 'max-w-(--content-max-width)'
			)}
		>
			{#if app.error}
				<p
					class="border-destructive/40 text-destructive rounded-md border px-3 py-2 text-sm"
					role="alert"
				>
					{app.error}
				</p>
			{/if}

			{#if app.settings}
				{@render children()}
			{/if}
		</div>
	</main>
</div>
