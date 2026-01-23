import { describe, it, expect, beforeEach, vi } from 'vitest';
import apiClient from '$api/client';
import type { OutlierListRequest, OutlierListResponse } from '$api/types';

/**
 * Integration Test: Outlier Filtering and Pagination Workflow
 *
 * Tests the integration between outlier filters, pagination, and API calls:
 * - Filter parameter construction
 * - Pagination state management
 * - API request integration
 * - Response handling
 */

describe('Integration: Outlier Workflow', () => {
	const mockOutliers = [
		{
			id: '1',
			detected_at: '2024-01-01T10:00:00Z',
			type: 'zscore',
			severity: 'high',
			address: '0x1234567890abcdef',
			amount: '1000',
			details: {},
			acknowledged: false
		},
		{
			id: '2',
			detected_at: '2024-01-01T11:00:00Z',
			type: 'iqr',
			severity: 'critical',
			address: '0xabcdef1234567890',
			amount: '5000',
			details: {},
			acknowledged: false
		}
	];

	beforeEach(() => {
		vi.clearAllMocks();
		global.fetch = vi.fn();
	});

	describe('Filter Integration', () => {
		it('should pass type filter to API', async () => {
			// Mock API response
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					outliers: mockOutliers.filter((o) => o.type === 'zscore'),
					total: 1,
					page: 1,
					limit: 20,
					total_pages: 1
				})
			} as Response);

			// Call API with type filter
			const result = await apiClient.listOutliers({
				type: 'zscore'
			});

			// Verify API was called with correct parameters
			const calls = vi.mocked(global.fetch).mock.calls;
			const [url] = calls[0];
			expect(url).toContain('type=zscore');

			// Verify result
			expect(result.outliers.length).toBe(1);
			expect(result.outliers[0].type).toBe('zscore');
		});

		it('should pass severity filter to API', async () => {
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					outliers: mockOutliers.filter((o) => o.severity === 'critical'),
					total: 1,
					page: 1,
					limit: 20,
					total_pages: 1
				})
			} as Response);

			await apiClient.listOutliers({
				severity: 'critical'
			});

			const calls = vi.mocked(global.fetch).mock.calls;
			const [url] = calls[0];
			expect(url).toContain('severity=critical');
		});

		it('should pass multiple filters to API', async () => {
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					outliers: [],
					total: 0,
					page: 1,
					limit: 20,
					total_pages: 0
				})
			} as Response);

			await apiClient.listOutliers({
				type: 'zscore',
				severity: 'high',
				acknowledged: false
			});

			const calls = vi.mocked(global.fetch).mock.calls;
			const [url] = calls[0];
			expect(url).toContain('type=zscore');
			expect(url).toContain('severity=high');
			expect(url).toContain('acknowledged=false');
		});
	});

	describe('Pagination Integration', () => {
		it('should pass pagination parameters to API', async () => {
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					outliers: mockOutliers,
					total: 50,
					page: 2,
					limit: 10,
					total_pages: 5
				})
			} as Response);

			const result = await apiClient.listOutliers({
				page: 2,
				limit: 10
			});

			// Verify pagination params in URL
			const calls = vi.mocked(global.fetch).mock.calls;
			const [url] = calls[0];
			expect(url).toContain('page=2');
			expect(url).toContain('limit=10');

			// Verify response includes pagination metadata
			expect(result.page).toBe(2);
			expect(result.total_pages).toBe(5);
			expect(result.total).toBe(50);
		});

		it('should handle first page', async () => {
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					outliers: mockOutliers,
					total: 100,
					page: 1,
					limit: 20,
					total_pages: 5
				})
			} as Response);

			const result = await apiClient.listOutliers({
				page: 1,
				limit: 20
			});

			expect(result.page).toBe(1);
			expect(result.total_pages).toBe(5);
		});

		it('should handle last page', async () => {
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					outliers: [mockOutliers[0]], // Only one item on last page
					total: 41,
					page: 3,
					limit: 20,
					total_pages: 3
				})
			} as Response);

			const result = await apiClient.listOutliers({
				page: 3,
				limit: 20
			});

			expect(result.page).toBe(3);
			expect(result.page).toBe(result.total_pages); // On last page
		});
	});

	describe('Combined Filters and Pagination', () => {
		it('should combine filters with pagination', async () => {
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					outliers: mockOutliers,
					total: 25,
					page: 2,
					limit: 10,
					total_pages: 3
				})
			} as Response);

			await apiClient.listOutliers({
				type: 'zscore',
				severity: 'high',
				page: 2,
				limit: 10
			});

			const calls = vi.mocked(global.fetch).mock.calls;
			const [url] = calls[0];

			// Verify both filters and pagination are in URL
			expect(url).toContain('type=zscore');
			expect(url).toContain('severity=high');
			expect(url).toContain('page=2');
			expect(url).toContain('limit=10');
		});

		it('should reset to page 1 when applying new filters', async () => {
			// Simulating UI behavior: when filter changes, reset to page 1
			const filterParams: OutlierListRequest = {
				type: 'iqr',
				page: 1, // Reset to first page
				limit: 20
			};

			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					outliers: mockOutliers,
					total: 30,
					page: 1,
					limit: 20,
					total_pages: 2
				})
			} as Response);

			const result = await apiClient.listOutliers(filterParams);

			// Verify we're on page 1 with new filter
			expect(result.page).toBe(1);
			const calls = vi.mocked(global.fetch).mock.calls;
			const [url] = calls[0];
			expect(url).toContain('page=1');
			expect(url).toContain('type=iqr');
		});
	});

	describe('Address Filter Integration', () => {
		it('should pass address filter to API', async () => {
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					outliers: [mockOutliers[0]],
					total: 1,
					page: 1,
					limit: 20,
					total_pages: 1
				})
			} as Response);

			await apiClient.listOutliers({
				address: '0x1234567890abcdef'
			});

			const calls = vi.mocked(global.fetch).mock.calls;
			const [url] = calls[0];
			expect(url).toContain('address=0x1234567890abcdef');
		});
	});

	describe('Date Range Filter Integration', () => {
		it('should pass date range filters to API', async () => {
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					outliers: mockOutliers,
					total: 2,
					page: 1,
					limit: 20,
					total_pages: 1
				})
			} as Response);

			await apiClient.listOutliers({
				from: '2024-01-01',
				to: '2024-01-31'
			});

			const calls = vi.mocked(global.fetch).mock.calls;
			const [url] = calls[0];
			expect(url).toContain('from=2024-01-01');
			expect(url).toContain('to=2024-01-31');
		});
	});

	describe('Empty Results Integration', () => {
		it('should handle empty results from filters', async () => {
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					outliers: [],
					total: 0,
					page: 1,
					limit: 20,
					total_pages: 0
				})
			} as Response);

			const result = await apiClient.listOutliers({
				type: 'nonexistent'
			});

			expect(result.outliers).toEqual([]);
			expect(result.total).toBe(0);
			expect(result.total_pages).toBe(0);
		});
	});

	describe('Error Handling Integration', () => {
		it('should handle API errors during filtering', async () => {
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: false,
				status: 400,
				json: async () => ({ error: 'Invalid filter parameters' })
			} as Response);

			await expect(
				apiClient.listOutliers({
					type: 'invalid-type'
				})
			).rejects.toThrow();
		});

		it('should handle network errors', async () => {
			vi.mocked(global.fetch).mockRejectedValueOnce(new Error('Network error'));

			await expect(
				apiClient.listOutliers({
					page: 1,
					limit: 20
				})
			).rejects.toThrow('Network error');
		});
	});
});
