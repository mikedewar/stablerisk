<script lang="ts">
	import { onMount } from 'svelte';
	import { outlierMessages } from '$stores/websocket';
	import { auth } from '$stores/auth';
	import apiClient from '$api/client';
	import type { Outlier, OutlierType, Severity } from '$api/types';

	let outliers: Outlier[] = [];
	let total = 0;
	let page = 1;
	let limit = 20;
	let totalPages = 0;
	let loading = true;
	let error: string | null = null;

	// Filters
	let typeFilter: OutlierType | '' = '';
	let severityFilter: Severity | '' = '';
	let acknowledgedFilter: 'all' | 'acknowledged' | 'unacknowledged' = 'all';

	// Selected outlier for details modal
	let selectedOutlier: Outlier | null = null;
	let acknowledgeNotes = '';
	let acknowledging = false;

	$: canAcknowledge = $auth.user?.role === 'admin' || $auth.user?.role === 'analyst';

	onMount(() => {
		loadOutliers();
	});

	async function loadOutliers() {
		loading = true;
		error = null;

		try {
			const params: any = { page, limit };

			if (typeFilter) params.type = typeFilter;
			if (severityFilter) params.severity = severityFilter;
			if (acknowledgedFilter !== 'all') {
				params.acknowledged = acknowledgedFilter === 'acknowledged';
			}

			const data = await apiClient.listOutliers(params);
			outliers = data.outliers;
			total = data.total;
			totalPages = data.total_pages;
			loading = false;
		} catch (e: any) {
			error = e.message || 'Failed to load outliers';
			loading = false;
		}
	}

	// Listen for real-time outlier updates
	$: if ($outlierMessages) {
		// Add new outlier to the top if on first page
		if (page === 1) {
			outliers = [$outlierMessages, ...outliers];
			total += 1;
		}
	}

	function handleFilterChange() {
		page = 1;
		loadOutliers();
	}

	function nextPage() {
		if (page < totalPages) {
			page++;
			loadOutliers();
		}
	}

	function prevPage() {
		if (page > 1) {
			page--;
			loadOutliers();
		}
	}

	function openDetails(outlier: Outlier) {
		selectedOutlier = outlier;
		acknowledgeNotes = outlier.notes || '';
	}

	function closeDetails() {
		selectedOutlier = null;
		acknowledgeNotes = '';
	}

	async function handleAcknowledge() {
		if (!selectedOutlier || !canAcknowledge) return;

		acknowledging = true;
		try {
			await apiClient.acknowledgeOutlier(selectedOutlier.id, { notes: acknowledgeNotes });

			// Update local state
			outliers = outliers.map((o) =>
				o.id === selectedOutlier.id
					? {
							...o,
							acknowledged: true,
							acknowledged_by: $auth.user?.username,
							acknowledged_at: new Date().toISOString(),
							notes: acknowledgeNotes
						}
					: o
			);

			closeDetails();
		} catch (e: any) {
			alert(`Failed to acknowledge outlier: ${e.message}`);
		} finally {
			acknowledging = false;
		}
	}

	function getSeverityClass(severity: string): string {
		return `severity-${severity}`;
	}

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleString();
	}

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text);
	}
</script>

<svelte:head>
	<title>Outliers - StableRisk</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex justify-between items-center">
		<h1 class="text-3xl font-bold">Outliers</h1>
		<div class="badge badge-lg badge-info">{total} Total</div>
	</div>

	<!-- Filters -->
	<div class="card bg-base-100 shadow-xl">
		<div class="card-body">
			<div class="grid grid-cols-1 md:grid-cols-4 gap-4">
				<!-- Type Filter -->
				<div class="form-control">
					<label class="label" for="type-filter">
						<span class="label-text">Type</span>
					</label>
					<select
						id="type-filter"
						class="select select-bordered"
						bind:value={typeFilter}
						on:change={handleFilterChange}
					>
						<option value="">All Types</option>
						<option value="zscore">Z-Score</option>
						<option value="iqr">IQR</option>
						<option value="pattern_circulation">Circulation</option>
						<option value="pattern_fanout">Fan-out</option>
						<option value="pattern_fanin">Fan-in</option>
						<option value="pattern_dormant">Dormant</option>
						<option value="pattern_velocity">Velocity</option>
					</select>
				</div>

				<!-- Severity Filter -->
				<div class="form-control">
					<label class="label" for="severity-filter">
						<span class="label-text">Severity</span>
					</label>
					<select
						id="severity-filter"
						class="select select-bordered"
						bind:value={severityFilter}
						on:change={handleFilterChange}
					>
						<option value="">All Severities</option>
						<option value="low">Low</option>
						<option value="medium">Medium</option>
						<option value="high">High</option>
						<option value="critical">Critical</option>
					</select>
				</div>

				<!-- Acknowledged Filter -->
				<div class="form-control">
					<label class="label" for="ack-filter">
						<span class="label-text">Status</span>
					</label>
					<select
						id="ack-filter"
						class="select select-bordered"
						bind:value={acknowledgedFilter}
						on:change={handleFilterChange}
					>
						<option value="all">All</option>
						<option value="unacknowledged">Unacknowledged</option>
						<option value="acknowledged">Acknowledged</option>
					</select>
				</div>

				<!-- Reset Filters -->
				<div class="form-control">
					<label class="label">
						<span class="label-text">&nbsp;</span>
					</label>
					<button
						class="btn btn-outline"
						on:click={() => {
							typeFilter = '';
							severityFilter = '';
							acknowledgedFilter = 'all';
							handleFilterChange();
						}}
					>
						Reset Filters
					</button>
				</div>
			</div>
		</div>
	</div>

	<!-- Outliers Table -->
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
	{:else}
		<div class="card bg-base-100 shadow-xl">
			<div class="card-body p-0">
				<div class="overflow-x-auto">
					<table class="table">
						<thead>
							<tr>
								<th>Detected</th>
								<th>Type</th>
								<th>Severity</th>
								<th>Address</th>
								<th>Amount</th>
								<th>Status</th>
								<th>Actions</th>
							</tr>
						</thead>
						<tbody>
							{#each outliers as outlier}
								<tr class="hover">
									<td class="text-xs">{formatDate(outlier.detected_at)}</td>
									<td><span class="outlier-type badge badge-sm">{outlier.type}</span></td>
									<td><span class={`${getSeverityClass(outlier.severity)} badge-sm`}>{outlier.severity}</span></td>
									<td class="font-mono text-xs">{outlier.address.slice(0, 12)}...</td>
									<td class="text-sm">{outlier.amount ? `${outlier.amount} USDT` : 'N/A'}</td>
									<td>
										{#if outlier.acknowledged}
											<span class="badge badge-success badge-sm">Acknowledged</span>
										{:else}
											<span class="badge badge-warning badge-sm">Pending</span>
										{/if}
									</td>
									<td>
										<button class="btn btn-xs btn-primary" on:click={() => openDetails(outlier)}>
											Details
										</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<!-- Pagination -->
				<div class="flex justify-between items-center p-4 border-t">
					<div class="text-sm">
						Showing {(page - 1) * limit + 1} to {Math.min(page * limit, total)} of {total}
					</div>
					<div class="join">
						<button class="join-item btn btn-sm" on:click={prevPage} disabled={page === 1}>
							Previous
						</button>
						<button class="join-item btn btn-sm btn-disabled">
							Page {page} of {totalPages}
						</button>
						<button class="join-item btn btn-sm" on:click={nextPage} disabled={page === totalPages}>
							Next
						</button>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>

<!-- Details Modal -->
{#if selectedOutlier}
	<div class="modal modal-open">
		<div class="modal-box max-w-3xl">
			<h3 class="font-bold text-lg mb-4">Outlier Details</h3>

			<div class="space-y-4">
				<!-- Basic Info -->
				<div class="grid grid-cols-2 gap-4">
					<div>
						<p class="text-sm text-base-content/60">ID</p>
						<p class="font-mono text-xs">{selectedOutlier.id}</p>
					</div>
					<div>
						<p class="text-sm text-base-content/60">Detected</p>
						<p>{formatDate(selectedOutlier.detected_at)}</p>
					</div>
					<div>
						<p class="text-sm text-base-content/60">Type</p>
						<span class="outlier-type badge">{selectedOutlier.type}</span>
					</div>
					<div>
						<p class="text-sm text-base-content/60">Severity</p>
						<span class={getSeverityClass(selectedOutlier.severity)}>{selectedOutlier.severity}</span>
					</div>
				</div>

				<!-- Address -->
				<div>
					<p class="text-sm text-base-content/60">Address</p>
					<div class="flex gap-2">
						<p class="font-mono text-sm flex-1">{selectedOutlier.address}</p>
						<button
							class="btn btn-xs btn-ghost"
							on:click={() => copyToClipboard(selectedOutlier.address)}
						>
							Copy
						</button>
					</div>
				</div>

				<!-- Transaction Hash -->
				{#if selectedOutlier.transaction_hash}
					<div>
						<p class="text-sm text-base-content/60">Transaction Hash</p>
						<div class="flex gap-2">
							<p class="font-mono text-sm flex-1">{selectedOutlier.transaction_hash}</p>
							<button
								class="btn btn-xs btn-ghost"
								on:click={() => copyToClipboard(selectedOutlier.transaction_hash)}
							>
								Copy
							</button>
						</div>
					</div>
				{/if}

				<!-- Amount -->
				{#if selectedOutlier.amount}
					<div>
						<p class="text-sm text-base-content/60">Amount</p>
						<p class="text-lg font-semibold">{selectedOutlier.amount} USDT</p>
					</div>
				{/if}

				<!-- Z-Score -->
				{#if selectedOutlier.z_score}
					<div>
						<p class="text-sm text-base-content/60">Z-Score</p>
						<p class="font-mono">{selectedOutlier.z_score.toFixed(2)}</p>
					</div>
				{/if}

				<!-- Details -->
				<div>
					<p class="text-sm text-base-content/60 mb-2">Details</p>
					<pre class="bg-base-200 p-4 rounded text-xs overflow-x-auto">{JSON.stringify(
							selectedOutlier.details,
							null,
							2
						)}</pre>
				</div>

				<!-- Acknowledgement -->
				{#if selectedOutlier.acknowledged}
					<div class="alert alert-success">
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
								d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<div>
							<p class="font-semibold">Acknowledged by {selectedOutlier.acknowledged_by}</p>
							<p class="text-sm">{formatDate(selectedOutlier.acknowledged_at)}</p>
							{#if selectedOutlier.notes}
								<p class="text-sm mt-2">{selectedOutlier.notes}</p>
							{/if}
						</div>
					</div>
				{:else if canAcknowledge}
					<div class="form-control">
						<label class="label" for="ack-notes">
							<span class="label-text">Acknowledgement Notes</span>
						</label>
						<textarea
							id="ack-notes"
							class="textarea textarea-bordered h-24"
							placeholder="Add notes about this outlier..."
							bind:value={acknowledgeNotes}
						></textarea>
					</div>
					<button
						class="btn btn-success w-full"
						on:click={handleAcknowledge}
						disabled={acknowledging}
					>
						{#if acknowledging}
							<span class="loading loading-spinner"></span>
						{/if}
						Acknowledge Outlier
					</button>
				{/if}
			</div>

			<div class="modal-action">
				<button class="btn" on:click={closeDetails}>Close</button>
			</div>
		</div>
	</div>
{/if}
