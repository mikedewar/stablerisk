import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { writable } from 'svelte/store';
import OutliersPage from './+page.svelte';
import type { Outlier } from '$api/types';

// Mock API client
vi.mock('$api/client', () => ({
	default: {
		listOutliers: vi.fn(),
		acknowledgeOutlier: vi.fn()
	}
}));

// Mock stores
vi.mock('$stores/websocket', () => ({
	outlierMessages: writable(null)
}));

vi.mock('$stores/auth', () => ({
	auth: writable({
		user: { id: '1', username: 'admin', role: 'admin' },
		token: 'test-token'
	})
}));

import apiClient from '$api/client';
import { outlierMessages } from '$stores/websocket';
import { auth } from '$stores/auth';

describe('Outliers Page', () => {
	const mockOutliers: Outlier[] = [
		{
			id: '1',
			detected_at: '2024-01-01T10:00:00Z',
			type: 'zscore',
			severity: 'high',
			address: '0x1234567890abcdef1234567890',
			amount: '1000',
			z_score: 3.5,
			details: { test: 'data' },
			acknowledged: false
		},
		{
			id: '2',
			detected_at: '2024-01-01T11:00:00Z',
			type: 'iqr',
			severity: 'critical',
			address: '0xabcdef1234567890abcdef1234',
			amount: '5000',
			details: {},
			acknowledged: true,
			acknowledged_by: 'admin',
			acknowledged_at: '2024-01-01T12:00:00Z',
			notes: 'Reviewed and cleared'
		}
	];

	beforeEach(async () => {
		vi.clearAllMocks();

		const { outlierMessages: om } = await import('$stores/websocket');
		om.set(null);

		vi.mocked(apiClient.listOutliers).mockResolvedValue({
			outliers: mockOutliers,
			total: 2,
			page: 1,
			limit: 20,
			total_pages: 1
		});
	});

	describe('Rendering', () => {
		it('should render page title', () => {
			render(OutliersPage);
			expect(screen.getByText('Outliers')).toBeTruthy();
		});

		it('should display total count badge', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('2 Total')).toBeTruthy();
			});
		});

		it('should render filter controls', () => {
			render(OutliersPage);

			expect(screen.getByLabelText('Type')).toBeTruthy();
			expect(screen.getByLabelText('Severity')).toBeTruthy();
			expect(screen.getByLabelText('Status')).toBeTruthy();
			expect(screen.getByText('Reset Filters')).toBeTruthy();
		});
	});

	describe('Data Loading', () => {
		it('should load outliers on mount', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(apiClient.listOutliers).toHaveBeenCalledWith({
					page: 1,
					limit: 20
				});
			});
		});

		it('should display outliers in table', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
				expect(screen.getByText('iqr')).toBeTruthy();
				expect(screen.getByText('high')).toBeTruthy();
				expect(screen.getByText('critical')).toBeTruthy();
			});
		});

		it('should display outlier addresses', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('0x1234567890...')).toBeTruthy();
				expect(screen.getByText('0xabcdef12345...')).toBeTruthy();
			});
		});

		it('should display amounts', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('1000 USDT')).toBeTruthy();
				expect(screen.getByText('5000 USDT')).toBeTruthy();
			});
		});

		it('should show acknowledged status', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('Acknowledged')).toBeTruthy();
				expect(screen.getByText('Pending')).toBeTruthy();
			});
		});
	});

	describe('Filters', () => {
		it('should filter by type', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			const typeFilter = screen.getByLabelText('Type') as HTMLSelectElement;
			await fireEvent.change(typeFilter, { target: { value: 'zscore' } });

			await waitFor(() => {
				expect(apiClient.listOutliers).toHaveBeenCalledWith({
					page: 1,
					limit: 20,
					type: 'zscore'
				});
			});
		});

		it('should filter by severity', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('high')).toBeTruthy();
			});

			const severityFilter = screen.getByLabelText('Severity') as HTMLSelectElement;
			await fireEvent.change(severityFilter, { target: { value: 'high' } });

			await waitFor(() => {
				expect(apiClient.listOutliers).toHaveBeenCalledWith({
					page: 1,
					limit: 20,
					severity: 'high'
				});
			});
		});

		it('should filter by acknowledged status', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('Acknowledged')).toBeTruthy();
			});

			const statusFilter = screen.getByLabelText('Status') as HTMLSelectElement;
			await fireEvent.change(statusFilter, { target: { value: 'acknowledged' } });

			await waitFor(() => {
				expect(apiClient.listOutliers).toHaveBeenCalledWith({
					page: 1,
					limit: 20,
					acknowledged: true
				});
			});
		});

		it('should filter unacknowledged outliers', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('Pending')).toBeTruthy();
			});

			const statusFilter = screen.getByLabelText('Status') as HTMLSelectElement;
			await fireEvent.change(statusFilter, { target: { value: 'unacknowledged' } });

			await waitFor(() => {
				expect(apiClient.listOutliers).toHaveBeenCalledWith({
					page: 1,
					limit: 20,
					acknowledged: false
				});
			});
		});

		it('should reset filters', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			// Set some filters
			const typeFilter = screen.getByLabelText('Type') as HTMLSelectElement;
			await fireEvent.change(typeFilter, { target: { value: 'zscore' } });

			const resetButton = screen.getByText('Reset Filters');
			await fireEvent.click(resetButton);

			await waitFor(() => {
				expect(typeFilter.value).toBe('');
				expect(apiClient.listOutliers).toHaveBeenLastCalledWith({
					page: 1,
					limit: 20
				});
			});
		});

		it('should reset page to 1 when filter changes', async () => {
			vi.mocked(apiClient.listOutliers).mockResolvedValue({
				outliers: mockOutliers,
				total: 50,
				page: 2,
				limit: 20,
				total_pages: 3
			});

			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			// Go to page 2
			const nextButton = screen.getByText('Next');
			await fireEvent.click(nextButton);

			// Apply filter - should reset to page 1
			const typeFilter = screen.getByLabelText('Type') as HTMLSelectElement;
			await fireEvent.change(typeFilter, { target: { value: 'iqr' } });

			await waitFor(() => {
				const calls = vi.mocked(apiClient.listOutliers).mock.calls;
				const lastCall = calls[calls.length - 1][0];
				expect(lastCall.page).toBe(1);
			});
		});
	});

	describe('Pagination', () => {
		beforeEach(() => {
			vi.mocked(apiClient.listOutliers).mockResolvedValue({
				outliers: mockOutliers,
				total: 50,
				page: 1,
				limit: 20,
				total_pages: 3
			});
		});

		it('should display pagination info', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('Showing 1 to 20 of 50')).toBeTruthy();
				expect(screen.getByText('Page 1 of 3')).toBeTruthy();
			});
		});

		it('should navigate to next page', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('Next')).toBeTruthy();
			});

			const nextButton = screen.getByText('Next');
			await fireEvent.click(nextButton);

			await waitFor(() => {
				expect(apiClient.listOutliers).toHaveBeenCalledWith({
					page: 2,
					limit: 20
				});
			});
		});

		it('should navigate to previous page', async () => {
			vi.mocked(apiClient.listOutliers).mockResolvedValue({
				outliers: mockOutliers,
				total: 50,
				page: 2,
				limit: 20,
				total_pages: 3
			});

			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('Previous')).toBeTruthy();
			});

			const prevButton = screen.getByText('Previous');
			await fireEvent.click(prevButton);

			await waitFor(() => {
				expect(apiClient.listOutliers).toHaveBeenLastCalledWith({
					page: 1,
					limit: 20
				});
			});
		});

		it('should disable Previous button on first page', async () => {
			render(OutliersPage);

			await waitFor(() => {
				const prevButton = screen.getByText('Previous');
				expect(prevButton.closest('button')?.disabled).toBe(true);
			});
		});

		it('should disable Next button on last page', async () => {
			vi.mocked(apiClient.listOutliers).mockResolvedValue({
				outliers: mockOutliers,
				total: 20,
				page: 1,
				limit: 20,
				total_pages: 1
			});

			render(OutliersPage);

			await waitFor(() => {
				const nextButton = screen.getByText('Next');
				expect(nextButton.closest('button')?.disabled).toBe(true);
			});
		});
	});

	describe('Details Modal', () => {
		it('should open modal when Details button clicked', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			const detailsButtons = screen.getAllByText('Details');
			await fireEvent.click(detailsButtons[0]);

			await waitFor(() => {
				expect(screen.getByText('Outlier Details')).toBeTruthy();
			});
		});

		it('should display outlier details in modal', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			const detailsButtons = screen.getAllByText('Details');
			await fireEvent.click(detailsButtons[0]);

			await waitFor(() => {
				expect(screen.getByText('Outlier Details')).toBeTruthy();
				expect(screen.getByText('0x1234567890abcdef1234567890')).toBeTruthy();
				expect(screen.getByText('1000 USDT')).toBeTruthy();
			});
		});

		it('should display z-score if available', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			const detailsButtons = screen.getAllByText('Details');
			await fireEvent.click(detailsButtons[0]);

			await waitFor(() => {
				expect(screen.getByText('Z-Score')).toBeTruthy();
				expect(screen.getByText('3.50')).toBeTruthy();
			});
		});

		it('should close modal when Close button clicked', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			const detailsButtons = screen.getAllByText('Details');
			await fireEvent.click(detailsButtons[0]);

			await waitFor(() => {
				expect(screen.getByText('Outlier Details')).toBeTruthy();
			});

			const closeButton = screen.getByText('Close');
			await fireEvent.click(closeButton);

			await waitFor(() => {
				expect(screen.queryByText('Outlier Details')).toBeFalsy();
			});
		});

		it('should show acknowledgement info for acknowledged outliers', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('iqr')).toBeTruthy();
			});

			const detailsButtons = screen.getAllByText('Details');
			await fireEvent.click(detailsButtons[1]);

			await waitFor(() => {
				expect(screen.getByText('Acknowledged by admin')).toBeTruthy();
				expect(screen.getByText('Reviewed and cleared')).toBeTruthy();
			});
		});

		it('should show acknowledgement form for unacknowledged outliers (admin)', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			const detailsButtons = screen.getAllByText('Details');
			await fireEvent.click(detailsButtons[0]);

			await waitFor(() => {
				expect(screen.getByLabelText('Acknowledgement Notes')).toBeTruthy();
				expect(screen.getByText('Acknowledge Outlier')).toBeTruthy();
			});
		});

		it('should not show acknowledgement form for viewers', async () => {
			auth.set({
				user: { id: '1', username: 'viewer', role: 'viewer' },
				token: 'test-token'
			});

			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			const detailsButtons = screen.getAllByText('Details');
			await fireEvent.click(detailsButtons[0]);

			await waitFor(() => {
				expect(screen.queryByText('Acknowledge Outlier')).toBeFalsy();
			});
		});
	});

	describe('Acknowledgement', () => {
		it('should acknowledge outlier with notes', async () => {
			vi.mocked(apiClient.acknowledgeOutlier).mockResolvedValue(undefined);

			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			const detailsButtons = screen.getAllByText('Details');
			await fireEvent.click(detailsButtons[0]);

			await waitFor(() => {
				expect(screen.getByLabelText('Acknowledgement Notes')).toBeTruthy();
			});

			const notesTextarea = screen.getByLabelText('Acknowledgement Notes') as HTMLTextAreaElement;
			await fireEvent.input(notesTextarea, { target: { value: 'Test notes' } });

			const ackButton = screen.getByText('Acknowledge Outlier');
			await fireEvent.click(ackButton);

			await waitFor(() => {
				expect(apiClient.acknowledgeOutlier).toHaveBeenCalledWith('1', {
					notes: 'Test notes'
				});
			});
		});

		it('should update outlier status after acknowledgement', async () => {
			vi.mocked(apiClient.acknowledgeOutlier).mockResolvedValue(undefined);

			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('Pending')).toBeTruthy();
			});

			const detailsButtons = screen.getAllByText('Details');
			await fireEvent.click(detailsButtons[0]);

			await waitFor(() => {
				expect(screen.getByText('Acknowledge Outlier')).toBeTruthy();
			});

			const ackButton = screen.getByText('Acknowledge Outlier');
			await fireEvent.click(ackButton);

			await waitFor(() => {
				// Modal should close
				expect(screen.queryByText('Outlier Details')).toBeFalsy();
			});
		});

		it('should close modal after successful acknowledgement', async () => {
			vi.mocked(apiClient.acknowledgeOutlier).mockResolvedValue(undefined);

			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			const detailsButtons = screen.getAllByText('Details');
			await fireEvent.click(detailsButtons[0]);

			const ackButton = screen.getByText('Acknowledge Outlier');
			await fireEvent.click(ackButton);

			await waitFor(() => {
				expect(screen.queryByText('Outlier Details')).toBeFalsy();
			});
		});
	});

	describe('Real-time Updates', () => {
		it('should add new outlier to list on page 1', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			const newOutlier: Outlier = {
				id: '3',
				detected_at: '2024-01-01T13:00:00Z',
				type: 'pattern_velocity',
				severity: 'medium',
				address: '0xnewaddress123456',
				amount: '2000',
				details: {},
				acknowledged: false
			};

			outlierMessages.set(newOutlier);

			await waitFor(() => {
				expect(screen.getByText('pattern_velocity')).toBeTruthy();
				expect(screen.getByText('3 Total')).toBeTruthy();
			});
		});

		it('should not add outlier to list if not on page 1', async () => {
			vi.mocked(apiClient.listOutliers).mockResolvedValueOnce({
				outliers: mockOutliers,
				total: 50,
				page: 2,
				limit: 20,
				total_pages: 3
			});

			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('Next')).toBeTruthy();
			});

			const nextButton = screen.getByText('Next');
			await fireEvent.click(nextButton);

			const newOutlier: Outlier = {
				id: '3',
				detected_at: '2024-01-01T13:00:00Z',
				type: 'pattern_velocity',
				severity: 'medium',
				address: '0xnewaddress123456',
				amount: '2000',
				details: {},
				acknowledged: false
			};

			outlierMessages.set(newOutlier);

			// Should not appear since we're on page 2
			expect(screen.queryByText('pattern_velocity')).toBeFalsy();
		});
	});

	describe('Error Handling', () => {
		it('should display error message when loading fails', async () => {
			vi.mocked(apiClient.listOutliers).mockRejectedValue(new Error('API Error'));

			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('API Error')).toBeTruthy();
			});
		});

		it('should show generic error message when error has no message', async () => {
			vi.mocked(apiClient.listOutliers).mockRejectedValue({});

			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('Failed to load outliers')).toBeTruthy();
			});
		});
	});

	describe('Copy to Clipboard', () => {
		beforeEach(() => {
			Object.assign(navigator, {
				clipboard: {
					writeText: vi.fn().mockResolvedValue(undefined)
				}
			});
		});

		it('should copy address to clipboard', async () => {
			render(OutliersPage);

			await waitFor(() => {
				expect(screen.getByText('zscore')).toBeTruthy();
			});

			const detailsButtons = screen.getAllByText('Details');
			await fireEvent.click(detailsButtons[0]);

			await waitFor(() => {
				expect(screen.getByText('0x1234567890abcdef1234567890')).toBeTruthy();
			});

			const copyButtons = screen.getAllByText('Copy');
			await fireEvent.click(copyButtons[0]);

			expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
				'0x1234567890abcdef1234567890'
			);
		});
	});
});
