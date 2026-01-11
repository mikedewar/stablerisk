<script lang="ts">
	import { onMount } from 'svelte';
	import apiClient from '$api/client';
	import type { Statistics } from '$api/types';

	let stats: Statistics | null = null;
	let trends: any = null;
	let loading = true;
	let error: string | null = null;
	let selectedDays = 7;

	onMount(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = null;

		try {
			const [statsData, trendsData] = await Promise.all([
				apiClient.getStatistics(),
				apiClient.getTrends(selectedDays)
			]);

			stats = statsData;
			trends = trendsData;
			loading = false;
		} catch (e: any) {
			error = e.message || 'Failed to load statistics';
			loading = false;
		}
	}

	function handleDaysChange() {
		loadData();
	}

	function formatNumber(num: number): string {
		return new Intl.NumberFormat().format(num);
	}

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleDateString();
	}
</script>

<svelte:head>
	<title>Statistics - StableRisk</title>
</svelte:head>

<div class="space-y-6">
	<h1 class="text-3xl font-bold">Statistics</h1>

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
		<!-- Overview Stats -->
		<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
			<div class="stats shadow">
				<div class="stat">
					<div class="stat-title">Total Transactions</div>
					<div class="stat-value">{formatNumber(stats.total_transactions)}</div>
					<div class="stat-desc">Monitored</div>
				</div>
			</div>

			<div class="stats shadow">
				<div class="stat">
					<div class="stat-title">Total Outliers</div>
					<div class="stat-value text-secondary">{formatNumber(stats.total_outliers)}</div>
					<div class="stat-desc">Detected anomalies</div>
				</div>
			</div>

			<div class="stats shadow">
				<div class="stat">
					<div class="stat-title">Detection Rate</div>
					<div class="stat-value text-accent">
						{stats.total_transactions > 0
							? ((stats.total_outliers / stats.total_transactions) * 100).toFixed(2)
							: 0}%
					</div>
					<div class="stat-desc">Outliers per transaction</div>
				</div>
			</div>
		</div>

		<!-- Outliers by Severity -->
		<div class="card bg-base-100 shadow-xl">
			<div class="card-body">
				<h2 class="card-title">Outliers by Severity</h2>
				<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
					<div class="stat bg-info/10 rounded-lg">
						<div class="stat-title">Low</div>
						<div class="stat-value text-info">
							{formatNumber(stats.outliers_by_severity.low || 0)}
						</div>
						<div class="stat-desc">
							{stats.total_outliers > 0
								? (((stats.outliers_by_severity.low || 0) / stats.total_outliers) * 100).toFixed(1)
								: 0}%
						</div>
					</div>

					<div class="stat bg-warning/10 rounded-lg">
						<div class="stat-title">Medium</div>
						<div class="stat-value text-warning">
							{formatNumber(stats.outliers_by_severity.medium || 0)}
						</div>
						<div class="stat-desc">
							{stats.total_outliers > 0
								? (
										((stats.outliers_by_severity.medium || 0) / stats.total_outliers) *
										100
									).toFixed(1)
								: 0}%
						</div>
					</div>

					<div class="stat bg-error/10 rounded-lg">
						<div class="stat-title">High</div>
						<div class="stat-value text-error">
							{formatNumber(stats.outliers_by_severity.high || 0)}
						</div>
						<div class="stat-desc">
							{stats.total_outliers > 0
								? (((stats.outliers_by_severity.high || 0) / stats.total_outliers) * 100).toFixed(1)
								: 0}%
						</div>
					</div>

					<div class="stat bg-error/20 rounded-lg">
						<div class="stat-title">Critical</div>
						<div class="stat-value text-error">
							{formatNumber(stats.outliers_by_severity.critical || 0)}
						</div>
						<div class="stat-desc">
							{stats.total_outliers > 0
								? (
										((stats.outliers_by_severity.critical || 0) / stats.total_outliers) *
										100
									).toFixed(1)
								: 0}%
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Outliers by Type -->
		<div class="card bg-base-100 shadow-xl">
			<div class="card-body">
				<h2 class="card-title">Outliers by Detection Method</h2>
				<div class="overflow-x-auto">
					<table class="table">
						<thead>
							<tr>
								<th>Method</th>
								<th>Count</th>
								<th>Percentage</th>
								<th>Progress</th>
							</tr>
						</thead>
						<tbody>
							{#each Object.entries(stats.outliers_by_type) as [type, count]}
								<tr>
									<td class="font-semibold">{type.replace('_', ' ').toUpperCase()}</td>
									<td>{formatNumber(count)}</td>
									<td>{stats.total_outliers > 0 ? ((count / stats.total_outliers) * 100).toFixed(1) : 0}%</td>
									<td>
										<progress
											class="progress progress-primary w-56"
											value={count}
											max={stats.total_outliers}
										></progress>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		</div>

		<!-- Trends -->
		{#if trends}
			<div class="card bg-base-100 shadow-xl">
				<div class="card-body">
					<div class="flex justify-between items-center mb-4">
						<h2 class="card-title">Outlier Trends</h2>
						<select
							class="select select-bordered select-sm"
							bind:value={selectedDays}
							on:change={handleDaysChange}
						>
							<option value={7}>Last 7 days</option>
							<option value={14}>Last 14 days</option>
							<option value={30}>Last 30 days</option>
							<option value={90}>Last 90 days</option>
						</select>
					</div>

					{#if trends.trends && trends.trends.length > 0}
						<div class="overflow-x-auto">
							<table class="table table-zebra">
								<thead>
									<tr>
										<th>Date</th>
										<th>Low</th>
										<th>Medium</th>
										<th>High</th>
										<th>Critical</th>
										<th>Total</th>
									</tr>
								</thead>
								<tbody>
									{#each trends.trends as trend}
										<tr>
											<td>{formatDate(trend.date)}</td>
											<td><span class="badge badge-info badge-sm">{trend.severity.low || 0}</span></td>
											<td><span class="badge badge-warning badge-sm">{trend.severity.medium || 0}</span></td>
											<td><span class="badge badge-error badge-sm">{trend.severity.high || 0}</span></td>
											<td><span class="badge badge-error badge-sm">{trend.severity.critical || 0}</span></td>
											<td class="font-bold">
												{(trend.severity.low || 0) +
													(trend.severity.medium || 0) +
													(trend.severity.high || 0) +
													(trend.severity.critical || 0)}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{:else}
						<p class="text-center py-8 text-base-content/60">No trend data available</p>
					{/if}
				</div>
			</div>
		{/if}

		<!-- Detection Status -->
		<div class="card bg-base-100 shadow-xl">
			<div class="card-body">
				<h2 class="card-title">Detection Engine Status</h2>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<div>
						<p class="text-sm text-base-content/60">Status</p>
						<p class="text-lg font-semibold flex items-center gap-2">
							{#if stats.detection_running}
								<span class="badge badge-success">Running</span>
							{:else}
								<span class="badge badge-error">Stopped</span>
							{/if}
						</p>
					</div>
					<div>
						<p class="text-sm text-base-content/60">Last Detection Run</p>
						<p class="text-lg font-semibold">{formatDate(stats.last_detection_run)}</p>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
