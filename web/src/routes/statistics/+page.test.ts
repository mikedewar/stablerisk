import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import StatisticsPage from './+page.svelte';
import type { Statistics } from '$api/types';

// Mock API client
vi.mock('$api/client', () => ({
	default: {
		getStatistics: vi.fn(),
		getTrends: vi.fn()
	}
}));

import apiClient from '$api/client';

describe('Statistics Page', () => {
	const mockStatistics: Statistics = {
		total_transactions: 10000,
		total_outliers: 500,
		outliers_by_severity: {
			low: 200,
			medium: 150,
			high: 100,
			critical: 50
		},
		outliers_by_type: {
			zscore: 200,
			iqr: 150,
			pattern_circulation: 50,
			pattern_fanout: 30,
			pattern_fanin: 30,
			pattern_dormant: 20,
			pattern_velocity: 20
		},
		last_detection_run: '2024-01-01T12:00:00Z',
		detection_running: true
	};

	const mockTrends = {
		trends: [
			{
				date: '2024-01-01',
				severity: { low: 10, medium: 5, high: 3, critical: 2 }
			},
			{
				date: '2024-01-02',
				severity: { low: 15, medium: 8, high: 4, critical: 1 }
			}
		]
	};

	beforeEach(() => {
		vi.clearAllMocks();

		vi.mocked(apiClient.getStatistics).mockResolvedValue(mockStatistics);
		vi.mocked(apiClient.getTrends).mockResolvedValue(mockTrends);
	});

	describe('Rendering', () => {
		it('should render page title', () => {
			render(StatisticsPage);
			expect(screen.getByText('Statistics')).toBeTruthy();
		});

		it('should show loading spinner initially', () => {
			const { container } = render(StatisticsPage);
			const spinner = container.querySelector('.loading-spinner');
			expect(spinner).toBeTruthy();
		});
	});

	describe('Overview Statistics', () => {
		it('should display total transactions', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Total Transactions')).toBeTruthy();
				expect(screen.getByText('10,000')).toBeTruthy();
			});
		});

		it('should display total outliers', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Total Outliers')).toBeTruthy();
				expect(screen.getByText('500')).toBeTruthy();
			});
		});

		it('should calculate and display detection rate', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Detection Rate')).toBeTruthy();
				expect(screen.getByText('5.00%')).toBeTruthy();
			});
		});

		it('should show 0% detection rate when no transactions', async () => {
			vi.mocked(apiClient.getStatistics).mockResolvedValue({
				...mockStatistics,
				total_transactions: 0
			});

			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('0%')).toBeTruthy();
			});
		});
	});

	describe('Outliers by Severity', () => {
		it('should display severity breakdown section', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Outliers by Severity')).toBeTruthy();
			});
		});

		it('should display all severity levels', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Low')).toBeTruthy();
				expect(screen.getByText('Medium')).toBeTruthy();
				expect(screen.getByText('High')).toBeTruthy();
				expect(screen.getByText('Critical')).toBeTruthy();
			});
		});

		it('should display counts for each severity', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('200')).toBeTruthy(); // low
				expect(screen.getByText('150')).toBeTruthy(); // medium
				expect(screen.getByText('100')).toBeTruthy(); // high
				expect(screen.getByText('50')).toBeTruthy(); // critical
			});
		});

		it('should display percentages for each severity', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('40.0%')).toBeTruthy(); // low: 200/500
				expect(screen.getByText('30.0%')).toBeTruthy(); // medium: 150/500
				expect(screen.getByText('20.0%')).toBeTruthy(); // high: 100/500
				expect(screen.getByText('10.0%')).toBeTruthy(); // critical: 50/500
			});
		});

		it('should handle zero outliers', async () => {
			vi.mocked(apiClient.getStatistics).mockResolvedValue({
				...mockStatistics,
				total_outliers: 0,
				outliers_by_severity: { low: 0, medium: 0, high: 0, critical: 0 }
			});

			render(StatisticsPage);

			await waitFor(() => {
				const zeroPercents = screen.queryAllByText('0%');
				expect(zeroPercents.length).toBeGreaterThan(0);
			});
		});
	});

	describe('Outliers by Detection Method', () => {
		it('should display detection methods section', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Outliers by Detection Method')).toBeTruthy();
			});
		});

		it('should display table headers', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Method')).toBeTruthy();
				expect(screen.getByText('Count')).toBeTruthy();
				expect(screen.getByText('Percentage')).toBeTruthy();
				expect(screen.getByText('Progress')).toBeTruthy();
			});
		});

		it('should display all detection methods', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('ZSCORE')).toBeTruthy();
				expect(screen.getByText('IQR')).toBeTruthy();
				expect(screen.getByText('PATTERN CIRCULATION')).toBeTruthy();
				expect(screen.getByText('PATTERN FANOUT')).toBeTruthy();
			});
		});

		it('should display counts for each method', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('200')).toBeTruthy(); // zscore
				expect(screen.getByText('150')).toBeTruthy(); // iqr
			});
		});

		it('should calculate percentages for each method', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('40.0%')).toBeTruthy(); // zscore: 200/500
				expect(screen.getByText('30.0%')).toBeTruthy(); // iqr: 150/500
			});
		});

		it('should render progress bars', async () => {
			const { container } = render(StatisticsPage);

			await waitFor(() => {
				const progressBars = container.querySelectorAll('progress');
				expect(progressBars.length).toBeGreaterThan(0);
			});
		});
	});

	describe('Trends', () => {
		it('should display trends section', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Outlier Trends')).toBeTruthy();
			});
		});

		it('should display time period selector', async () => {
			const { container } = render(StatisticsPage);

			await waitFor(() => {
				const select = container.querySelector('select');
				expect(select).toBeTruthy();
				expect(screen.getByText('Last 7 days')).toBeTruthy();
			});
		});

		it('should have all time period options', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Last 7 days')).toBeTruthy();
				expect(screen.getByText('Last 14 days')).toBeTruthy();
				expect(screen.getByText('Last 30 days')).toBeTruthy();
				expect(screen.getByText('Last 90 days')).toBeTruthy();
			});
		});

		it('should load trends with default 7 days', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(apiClient.getTrends).toHaveBeenCalledWith(7);
			});
		});

		it('should reload trends when time period changes', async () => {
			const { container } = render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Outlier Trends')).toBeTruthy();
			});

			const select = container.querySelector('select') as HTMLSelectElement;
			await fireEvent.change(select, { target: { value: '30' } });

			await waitFor(() => {
				expect(apiClient.getTrends).toHaveBeenCalledWith(30);
			});
		});

		it('should display trend data in table', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Date')).toBeTruthy();
				// Check for severity headers
				const headers = screen.getAllByText('Low');
				expect(headers.length).toBeGreaterThan(0);
			});
		});

		it('should display trend dates', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				// Dates will be formatted based on locale, just check table is present
				const table = screen.getAllByRole('table');
				expect(table.length).toBeGreaterThan(0);
			});
		});

		it('should calculate daily totals', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				// First row total: 10+5+3+2 = 20
				expect(screen.getByText('20')).toBeTruthy();
				// Second row total: 15+8+4+1 = 28
				expect(screen.getByText('28')).toBeTruthy();
			});
		});

		it('should show empty state when no trend data', async () => {
			vi.mocked(apiClient.getTrends).mockResolvedValue({ trends: [] });

			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('No trend data available')).toBeTruthy();
			});
		});
	});

	describe('Detection Engine Status', () => {
		it('should display detection status section', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Detection Engine Status')).toBeTruthy();
			});
		});

		it('should show running status when detection is active', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Running')).toBeTruthy();
			});
		});

		it('should show stopped status when detection is inactive', async () => {
			vi.mocked(apiClient.getStatistics).mockResolvedValue({
				...mockStatistics,
				detection_running: false
			});

			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Stopped')).toBeTruthy();
			});
		});

		it('should display last detection run date', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Last Detection Run')).toBeTruthy();
				// Date format varies by locale, just check it's rendered
				const dateElements = screen.queryAllByText(/2024|1\/1|Jan/);
				expect(dateElements.length).toBeGreaterThan(0);
			});
		});
	});

	describe('Error Handling', () => {
		it('should display error message when statistics fail to load', async () => {
			vi.mocked(apiClient.getStatistics).mockRejectedValue(new Error('API Error'));

			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('API Error')).toBeTruthy();
			});
		});

		it('should show generic error message when error has no message', async () => {
			vi.mocked(apiClient.getStatistics).mockRejectedValue({});

			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('Failed to load statistics')).toBeTruthy();
			});
		});

		it('should display error alert with proper styling', async () => {
			vi.mocked(apiClient.getStatistics).mockRejectedValue(new Error('Test error'));

			const { container } = render(StatisticsPage);

			await waitFor(() => {
				const alert = container.querySelector('.alert-error');
				expect(alert).toBeTruthy();
			});
		});
	});

	describe('Data Loading', () => {
		it('should load statistics and trends on mount', async () => {
			render(StatisticsPage);

			await waitFor(() => {
				expect(apiClient.getStatistics).toHaveBeenCalled();
				expect(apiClient.getTrends).toHaveBeenCalledWith(7);
			});
		});

		it('should load both statistics and trends in parallel', async () => {
			let statsResolved = false;
			let trendsResolved = false;

			vi.mocked(apiClient.getStatistics).mockImplementation(async () => {
				await new Promise((resolve) => setTimeout(resolve, 10));
				statsResolved = true;
				return mockStatistics;
			});

			vi.mocked(apiClient.getTrends).mockImplementation(async () => {
				await new Promise((resolve) => setTimeout(resolve, 10));
				trendsResolved = true;
				return mockTrends;
			});

			render(StatisticsPage);

			await waitFor(() => {
				expect(statsResolved).toBe(true);
				expect(trendsResolved).toBe(true);
			});
		});
	});

	describe('Number Formatting', () => {
		it('should format large numbers with commas', async () => {
			vi.mocked(apiClient.getStatistics).mockResolvedValue({
				...mockStatistics,
				total_transactions: 1234567
			});

			render(StatisticsPage);

			await waitFor(() => {
				expect(screen.getByText('1,234,567')).toBeTruthy();
			});
		});

		it('should format percentages to one decimal place', async () => {
			vi.mocked(apiClient.getStatistics).mockResolvedValue({
				...mockStatistics,
				total_outliers: 333,
				outliers_by_severity: { low: 100, medium: 100, high: 100, critical: 33 }
			});

			render(StatisticsPage);

			await waitFor(() => {
				// 100/333 = 30.03003%
				expect(screen.getByText('30.0%')).toBeTruthy();
			});
		});

		it('should format detection rate to two decimal places', async () => {
			vi.mocked(apiClient.getStatistics).mockResolvedValue({
				...mockStatistics,
				total_transactions: 12345,
				total_outliers: 123
			});

			render(StatisticsPage);

			await waitFor(() => {
				// 123/12345 = 0.99635%
				expect(screen.getByText('1.00%')).toBeTruthy();
			});
		});
	});
});
