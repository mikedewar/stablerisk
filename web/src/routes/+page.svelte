<script lang="ts">
	import { onMount } from 'svelte';
	import { outlierMessages } from '$stores/websocket';
	import apiClient from '$api/client';
	import type { Statistics, Outlier } from '$api/types';

	let stats: Statistics | null = null;
	let recentOutliers: Outlier[] = [];
	let loading = true;
	let error: string | null = null;

	onMount(async () => {
		try {
			// Load initial data
			const [statsData, outliersData] = await Promise.all([
				apiClient.getStatistics(),
				apiClient.listOutliers({ page: 1, limit: 5 })
			]);

			stats = statsData;
			recentOutliers = outliersData.outliers;
			loading = false;
		} catch (e: any) {
			error = e.message || 'Failed to load data';
			loading = false;
		}
	});

	// Listen for real-time outlier updates
	$: if ($outlierMessages) {
		// Add new outlier to the top of the list
		recentOutliers = [$outlierMessages, ...recentOutliers.slice(0, 4)];

		// Update statistics
		if (stats) {
			stats.total_outliers += 1;
			stats.outliers_by_severity[$outlierMessages.severity] =
				(stats.outliers_by_severity[$outlierMessages.severity] || 0) + 1;
			stats.outliers_by_type[$outlierMessages.type] =
				(stats.outliers_by_type[$outlierMessages.type] || 0) + 1;
		}
	}

	function getSeverityClass(severity: string): string {
		return `severity-${severity}`;
	}

	function formatNumber(num: number): string {
		return new Intl.NumberFormat().format(num);
	}

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleString();
	}
</script>

<svelte:head>
	<title>Dashboard - StableRisk</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex justify-between items-center">
		<h1 class="text-3xl font-bold">Dashboard</h1>
		{#if stats}
			<div class="badge badge-lg" class:badge-success={stats.detection_running} class:badge-error={!stats.detection_running}>
				{stats.detection_running ? 'Detection Active' : 'Detection Stopped'}
			</div>
		{/if}
	</div>

	{#if loading}
		<div class="flex justify-center py-20">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if error}
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
			<span>{error}</span>
		</div>
	{:else if stats}
		<!-- Statistics Cards -->
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
			<!-- Total Transactions -->
			<div class="stats shadow">
				<div class="stat">
					<div class="stat-figure text-primary">
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							class="inline-block w-8 h-8 stroke-current"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M7 12l3-3 3 3 4-4M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z"
							></path>
						</svg>
					</div>
					<div class="stat-title">Total Transactions</div>
					<div class="stat-value text-primary">{formatNumber(stats.total_transactions)}</div>
					<div class="stat-desc">Monitored on Tron blockchain</div>
				</div>
			</div>

			<!-- Total Outliers -->
			<div class="stats shadow">
				<div class="stat">
					<div class="stat-figure text-secondary">
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							class="inline-block w-8 h-8 stroke-current"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
							></path>
						</svg>
					</div>
					<div class="stat-title">Total Outliers</div>
					<div class="stat-value text-secondary">{formatNumber(stats.total_outliers)}</div>
					<div class="stat-desc">Anomalies detected</div>
				</div>
			</div>

			<!-- Critical Outliers -->
			<div class="stats shadow">
				<div class="stat">
					<div class="stat-figure text-error">
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							class="inline-block w-8 h-8 stroke-current"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							></path>
						</svg>
					</div>
					<div class="stat-title">Critical</div>
					<div class="stat-value text-error">
						{formatNumber(stats.outliers_by_severity.critical || 0)}
					</div>
					<div class="stat-desc">Require immediate attention</div>
				</div>
			</div>

			<!-- High Outliers -->
			<div class="stats shadow">
				<div class="stat">
					<div class="stat-figure text-warning">
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							class="inline-block w-8 h-8 stroke-current"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							></path>
						</svg>
					</div>
					<div class="stat-title">High Severity</div>
					<div class="stat-value text-warning">
						{formatNumber(stats.outliers_by_severity.high || 0)}
					</div>
					<div class="stat-desc">Need investigation</div>
				</div>
			</div>
		</div>

		<!-- Outliers by Type -->
		<div class="card bg-base-100 shadow-xl">
			<div class="card-body">
				<h2 class="card-title">Outliers by Detection Method</h2>
				<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
					{#each Object.entries(stats.outliers_by_type) as [type, count]}
						<div class="stat bg-base-200 rounded-lg p-4">
							<div class="stat-title text-xs">{type.replace('_', ' ').toUpperCase()}</div>
							<div class="stat-value text-2xl">{formatNumber(count)}</div>
						</div>
					{/each}
				</div>
			</div>
		</div>

		<!-- Recent Outliers -->
		<div class="card bg-base-100 shadow-xl">
			<div class="card-body">
				<div class="flex justify-between items-center">
					<h2 class="card-title">Recent Outliers</h2>
					<a href="/outliers" class="btn btn-sm btn-primary">View All</a>
				</div>

				{#if recentOutliers.length === 0}
					<p class="text-center py-8 text-base-content/60">No outliers detected yet</p>
				{:else}
					<div class="overflow-x-auto">
						<table class="table">
							<thead>
								<tr>
									<th>Detected</th>
									<th>Type</th>
									<th>Severity</th>
									<th>Address</th>
									<th>Amount</th>
								</tr>
							</thead>
							<tbody>
								{#each recentOutliers as outlier}
									<tr class="hover">
										<td>{formatDate(outlier.detected_at)}</td>
										<td><span class="outlier-type badge">{outlier.type}</span></td>
										<td><span class={getSeverityClass(outlier.severity)}>{outlier.severity}</span></td>
										<td class="font-mono text-xs">{outlier.address.slice(0, 10)}...</td>
										<td>{outlier.amount ? `${outlier.amount} USDT` : 'N/A'}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</div>
		</div>

		<!-- Detection Status -->
		<div class="card bg-base-100 shadow-xl">
			<div class="card-body">
				<h2 class="card-title">Detection Status</h2>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<p class="text-sm text-base-content/60">Last Detection Run</p>
						<p class="text-lg font-semibold">{formatDate(stats.last_detection_run)}</p>
					</div>
					<div>
						<p class="text-sm text-base-content/60">Status</p>
						<p class="text-lg font-semibold">
							{stats.detection_running ? 'Active' : 'Stopped'}
						</p>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
