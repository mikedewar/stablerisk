<script lang="ts">
	import '../app.css';
	import { auth } from '$stores/auth';
	import { websocket } from '$stores/websocket';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';

	let menuOpen = false;

	onMount(() => {
		// Redirect to login if not authenticated (except on login page)
		const unsubscribe = auth.subscribe((state) => {
			if (!state.user && !$page.url.pathname.startsWith('/login')) {
				goto('/login');
			}
		});

		return unsubscribe;
	});

	function handleLogout() {
		auth.logout();
		websocket.disconnect();
		goto('/login');
	}
</script>

<div class="min-h-screen bg-base-200">
	{#if $auth.user}
		<!-- Navigation -->
		<div class="navbar bg-base-100 shadow-lg">
			<div class="flex-1">
				<a href="/" class="btn btn-ghost text-xl">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-6 w-6 mr-2"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
						/>
					</svg>
					StableRisk
				</a>
			</div>

			<!-- Desktop Navigation -->
			<div class="flex-none hidden lg:flex">
				<ul class="menu menu-horizontal px-1">
					<li>
						<a href="/" class:active={$page.url.pathname === '/'}>Dashboard</a>
					</li>
					<li>
						<a href="/outliers" class:active={$page.url.pathname === '/outliers'}>Outliers</a>
					</li>
					<li>
						<a href="/statistics" class:active={$page.url.pathname === '/statistics'}
							>Statistics</a
						>
					</li>
				</ul>
			</div>

			<!-- Connection Status -->
			<div class="flex-none">
				<div class="tooltip tooltip-left" data-tip={$websocket.connected ? 'Connected' : 'Disconnected'}>
					<div class="indicator">
						{#if $websocket.reconnecting}
							<span class="indicator-item badge badge-warning badge-xs"></span>
						{:else if $websocket.connected}
							<span class="indicator-item badge badge-success badge-xs"></span>
						{:else}
							<span class="indicator-item badge badge-error badge-xs"></span>
						{/if}
						<svg
							xmlns="http://www.w3.org/2000/svg"
							class="h-5 w-5 mx-2"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M13 10V3L4 14h7v7l9-11h-7z"
							/>
						</svg>
					</div>
				</div>
			</div>

			<!-- User Menu -->
			<div class="flex-none">
				<div class="dropdown dropdown-end">
					<div tabindex="0" role="button" class="btn btn-ghost btn-circle avatar placeholder">
						<div class="bg-neutral text-neutral-content rounded-full w-10">
							<span class="text-xl">{$auth.user.username[0].toUpperCase()}</span>
						</div>
					</div>
					<ul
						tabindex="0"
						class="menu menu-sm dropdown-content mt-3 z-[1] p-2 shadow bg-base-100 rounded-box w-52"
					>
						<li class="menu-title">
							<span>{$auth.user.username}</span>
							<span class="badge badge-sm">{$auth.user.role}</span>
						</li>
						<li><a href="/profile">Profile</a></li>
						<li><button on:click={handleLogout}>Logout</button></li>
					</ul>
				</div>
			</div>

			<!-- Mobile Menu -->
			<div class="flex-none lg:hidden">
				<button class="btn btn-square btn-ghost" on:click={() => (menuOpen = !menuOpen)}>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						class="inline-block w-5 h-5 stroke-current"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M4 6h16M4 12h16M4 18h16"
						></path>
					</svg>
				</button>
			</div>
		</div>

		<!-- Mobile Menu Dropdown -->
		{#if menuOpen}
			<div class="lg:hidden bg-base-100 shadow-lg">
				<ul class="menu p-4">
					<li><a href="/" on:click={() => (menuOpen = false)}>Dashboard</a></li>
					<li><a href="/outliers" on:click={() => (menuOpen = false)}>Outliers</a></li>
					<li><a href="/statistics" on:click={() => (menuOpen = false)}>Statistics</a></li>
				</ul>
			</div>
		{/if}
	{/if}

	<!-- Main Content -->
	<main class="container mx-auto p-4">
		<slot />
	</main>
</div>

<style>
	.active {
		@apply bg-primary text-primary-content;
	}
</style>
