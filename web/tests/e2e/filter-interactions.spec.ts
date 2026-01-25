import { test, expect } from '@playwright/test';
import { login } from './helpers';

/**
 * E2E Test: Filter Interactions Across Pages
 *
 * Tests filter functionality on the Outliers page:
 * - Type filters
 * - Severity filters
 * - Acknowledged status filters
 * - Filter reset
 * - Pagination with filters
 */

test.describe('Filter Interactions', () => {
	test.beforeEach(async ({ page }) => {
		// Login before each test
		await login(page);
		await expect(page).toHaveURL('/');

		// Navigate to outliers page
		await page.click('a[href="/outliers"]');
		await expect(page).toHaveURL('/outliers');
		await page.waitForSelector('table', { timeout: 10000 });
	});

	test('should filter outliers by type', async ({ page }) => {
		await test.step('Apply type filter', async () => {
			// Find type select
			const typeSelect = page.locator('select[id="type-filter"]');
			if (await typeSelect.isVisible()) {
				// Select ZSCORE type
				await typeSelect.selectOption('zscore');

				// Wait for table to update
				await page.waitForTimeout(1000);

				// Verify URL contains filter parameter
				const url = page.url();
				expect(url).toContain('type=zscore');
			}
		});
	});

	test('should filter outliers by severity', async ({ page }) => {
		await test.step('Apply severity filter', async () => {
			const severitySelect = page.locator('select[id="severity-filter"]');
			if (await severitySelect.isVisible()) {
				// Select high severity
				await severitySelect.selectOption('high');

				// Wait for update
				await page.waitForTimeout(1000);

				// Verify URL contains filter
				const url = page.url();
				expect(url).toContain('severity=high');
			}
		});
	});

	test('should filter by acknowledged status', async ({ page }) => {
		await test.step('Filter unacknowledged outliers', async () => {
			const acknowledgedSelect = page.locator('select[id="acknowledged-filter"]');
			if (await acknowledgedSelect.isVisible()) {
				// Select unacknowledged only
				await acknowledgedSelect.selectOption('false');

				// Wait for update
				await page.waitForTimeout(1000);

				// Verify URL contains filter
				const url = page.url();
				expect(url).toContain('acknowledged=false');
			}
		});
	});

	test('should combine multiple filters', async ({ page }) => {
		await test.step('Apply multiple filters simultaneously', async () => {
			// Apply type filter
			const typeSelect = page.locator('select[id="type-filter"]');
			if (await typeSelect.isVisible()) {
				await typeSelect.selectOption('iqr');
				await page.waitForTimeout(500);
			}

			// Apply severity filter
			const severitySelect = page.locator('select[id="severity-filter"]');
			if (await severitySelect.isVisible()) {
				await severitySelect.selectOption('critical');
				await page.waitForTimeout(500);
			}

			// Verify URL contains both filters
			const url = page.url();
			expect(url).toContain('type=iqr');
			expect(url).toContain('severity=critical');
		});
	});

	test('should reset filters', async ({ page }) => {
		await test.step('Reset all filters to default', async () => {
			// Apply some filters first
			const typeSelect = page.locator('select[id="type-filter"]');
			if (await typeSelect.isVisible()) {
				await typeSelect.selectOption('zscore');
				await page.waitForTimeout(500);
			}

			// Click reset button
			const resetButton = page.locator('button:has-text("Reset")');
			if (await resetButton.isVisible()) {
				await resetButton.click();
				await page.waitForTimeout(500);

				// Verify URL no longer has filter params (except maybe page/limit defaults)
				const url = page.url();
				expect(url).not.toContain('type=zscore');
			}
		});
	});

	test('should reset to page 1 when filter changes', async ({ page }) => {
		await test.step('Change filter and verify page resets', async () => {
			// Navigate to page 2 if possible
			const nextButton = page.locator('button:has-text("Next")');
			if (await nextButton.isEnabled()) {
				await nextButton.click();
				await page.waitForTimeout(500);

				// Verify we're on page 2
				expect(page.url()).toContain('page=2');

				// Now change a filter
				const typeSelect = page.locator('select[id="type-filter"]');
				if (await typeSelect.isVisible()) {
					await typeSelect.selectOption('iqr');
					await page.waitForTimeout(500);

					// Should be back on page 1
					const url = page.url();
					expect(url).not.toContain('page=2');
				}
			}
		});
	});

	test('should maintain filters during pagination', async ({ page }) => {
		await test.step('Apply filter and paginate', async () => {
			// Apply a filter
			const severitySelect = page.locator('select[id="severity-filter"]');
			if (await severitySelect.isVisible()) {
				await severitySelect.selectOption('high');
				await page.waitForTimeout(500);

				// Navigate to next page if available
				const nextButton = page.locator('button:has-text("Next")');
				if (await nextButton.isEnabled()) {
					await nextButton.click();
					await page.waitForTimeout(500);

					// Filter should still be applied
					const url = page.url();
					expect(url).toContain('severity=high');
				}
			}
		});
	});

	test('should handle empty filter results', async ({ page }) => {
		await test.step('Apply filters that return no results', async () => {
			// Apply very specific filters that likely return no results
			const typeSelect = page.locator('select[id="type-filter"]');
			if (await typeSelect.isVisible()) {
				await typeSelect.selectOption('pattern_velocity');
				await page.waitForTimeout(500);

				const severitySelect = page.locator('select[id="severity-filter"]');
				if (await severitySelect.isVisible()) {
					await severitySelect.selectOption('critical');
					await page.waitForTimeout(1000);

					// Check if empty state is shown or table is empty
					const emptyState = await page
						.locator('text=No outliers found')
						.isVisible()
						.catch(() => false);
					const rowCount = await page.locator('table tbody tr').count();

					// Either empty state should be shown or table should have no data rows
					expect(emptyState || rowCount === 0).toBeTruthy();
				}
			}
		});
	});

	test('should persist filters in URL for sharing', async ({ page }) => {
		await test.step('Apply filters and copy URL', async () => {
			// Apply filters
			const typeSelect = page.locator('select[id="type-filter"]');
			if (await typeSelect.isVisible()) {
				await typeSelect.selectOption('zscore');
				await page.waitForTimeout(500);
			}

			// Get the URL
			const urlWithFilters = page.url();

			// Navigate away
			await page.click('a[href="/"]');
			await expect(page).toHaveURL('/');

			// Navigate back using the URL with filters
			await page.goto(urlWithFilters);

			// Filters should still be applied
			const typeSelectAfter = page.locator('select[id="type-filter"]');
			if (await typeSelectAfter.isVisible()) {
				const selectedValue = await typeSelectAfter.inputValue();
				expect(selectedValue).toBe('zscore');
			}
		});
	});
});
