<script lang="ts">
	import { auth } from '$stores/auth';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	let username = '';
	let password = '';
	let loading = false;

	onMount(() => {
		// Redirect if already logged in
		const unsubscribe = auth.subscribe((state) => {
			if (state.user) {
				goto('/');
			}
		});

		return unsubscribe;
	});

	async function handleLogin() {
		if (!username || !password) {
			return;
		}

		loading = true;
		const success = await auth.login(username, password);
		loading = false;

		if (success) {
			goto('/');
		}
	}

	function handleKeyPress(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			handleLogin();
		}
	}
</script>

<svelte:head>
	<title>Login - StableRisk</title>
</svelte:head>

<div class="hero min-h-screen bg-base-200">
	<div class="hero-content flex-col lg:flex-row-reverse">
		<div class="text-center lg:text-left">
			<h1 class="text-5xl font-bold flex items-center gap-2">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-12 w-12"
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
			</h1>
			<p class="py-6">
				Real-time USDT transaction monitoring and anomaly detection system. Secure access to
				outlier analysis and statistical insights.
			</p>
		</div>
		<div class="card flex-shrink-0 w-full max-w-sm shadow-2xl bg-base-100">
			<form class="card-body" on:submit|preventDefault={handleLogin}>
				<h2 class="text-2xl font-bold text-center mb-4">Sign In</h2>

				{#if $auth.error}
					<div class="alert alert-error">
						<svg
							xmlns="http://www.w3.org/2000/svg"
							class="stroke-current shrink-0 h-6 w-6"
							fill="none"
							viewBox="0 0 24 24"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<span>{$auth.error}</span>
					</div>
				{/if}

				<div class="form-control">
					<label class="label" for="username">
						<span class="label-text">Username</span>
					</label>
					<input
						id="username"
						name="username"
						type="text"
						placeholder="username"
						class="input input-bordered"
						bind:value={username}
						on:keypress={handleKeyPress}
						disabled={loading}
						required
					/>
				</div>

				<div class="form-control">
					<label class="label" for="password">
						<span class="label-text">Password</span>
					</label>
					<input
						id="password"
						name="password"
						type="password"
						placeholder="password"
						class="input input-bordered"
						bind:value={password}
						on:keypress={handleKeyPress}
						disabled={loading}
						required
					/>
				</div>

				<div class="form-control mt-6">
					<button class="btn btn-primary" type="submit" disabled={loading || !username || !password}>
						{#if loading}
							<span class="loading loading-spinner"></span>
							Signing in...
						{:else}
							Sign In
						{/if}
					</button>
				</div>

				<div class="divider">Default Credentials</div>

				<div class="text-sm space-y-1 opacity-70">
					<p><strong>Admin:</strong> admin / changeme123</p>
					<p><strong>Analyst:</strong> analyst / changeme123</p>
					<p><strong>Viewer:</strong> viewer / changeme123</p>
				</div>
			</form>
		</div>
	</div>
</div>
