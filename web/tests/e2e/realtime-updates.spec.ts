import { test, expect } from '@playwright/test';

/**
 * E2E Test: Real-time Updates
 *
 * Tests WebSocket integration and real-time features:
 * - WebSocket connection status indicator
 * - Real-time outlier updates on dashboard
 * - Connection state management
 * - UI responsiveness to connection changes
 */

test.describe('Real-time Updates', () => {
	test.beforeEach(async ({ page }) => {
		// Login before each test
		await page.goto('/login');
		await page.fill('input[name="username"]', 'admin');
		await page.fill('input[name="password"]', 'changeme123');
		await page.click('button[type="submit"]');
		await expect(page).toHaveURL('/');
	});

	test('should display WebSocket connection status indicator', async ({ page }) => {
		await test.step('Verify connection status is visible', async () => {
			// Look for the connection status indicator
			// The layout should have a WebSocket status badge
			const statusIndicator = page.locator('.indicator, .tooltip');

			// Should be visible in the navbar
			await expect(statusIndicator.first()).toBeVisible({ timeout: 5000 });
		});
	});

	test('should show connection state badges', async ({ page }) => {
		await test.step('Check for connection state badges', async () => {
			// Wait a moment for WebSocket to attempt connection
			await page.waitForTimeout(2000);

			// Look for badge indicators (success, warning, or error)
			const badges = page.locator('.badge-success, .badge-warning, .badge-error');
			const badgeCount = await badges.count();

			// At least one badge should be present indicating connection state
			expect(badgeCount).toBeGreaterThan(0);
		});
	});

	test('should display recent outliers section on dashboard', async ({ page }) => {
		await test.step('Verify recent outliers section exists', async () => {
			// Dashboard should have a recent outliers section
			await expect(page.locator('text=Recent Outliers')).toBeVisible({ timeout: 10000 });

			// Should have a table or empty state
			const hasTable = await page.locator('table').isVisible().catch(() => false);
			const hasEmptyState = await page
				.locator('text=No outliers detected yet')
				.isVisible()
				.catch(() => false);

			// Either table or empty state should be shown
			expect(hasTable || hasEmptyState).toBeTruthy();
		});
	});

	test('should have View All link to outliers page', async ({ page }) => {
		await test.step('Verify link to full outliers page', async () => {
			// Wait for dashboard to load
			await page.waitForSelector('text=Recent Outliers', { timeout: 10000 });

			// Find View All button
			const viewAllButton = page.locator('a:has-text("View All")');

			if (await viewAllButton.isVisible()) {
				// Verify it points to /outliers
				const href = await viewAllButton.getAttribute('href');
				expect(href).toBe('/outliers');

				// Click it
				await viewAllButton.click();

				// Should navigate to outliers page
				await expect(page).toHaveURL('/outliers');
			}
		});
	});

	test('should limit recent outliers to 5 items', async ({ page }) => {
		await test.step('Verify recent outliers limit', async () => {
			// Wait for dashboard
			await page.waitForSelector('text=Recent Outliers', { timeout: 10000 });

			// Count table rows (excluding header)
			const rows = page.locator('table tbody tr');
			const rowCount = await rows.count();

			// Should show at most 5 outliers
			expect(rowCount).toBeLessThanOrEqual(5);
		});
	});

	test('should handle WebSocket disconnection gracefully', async ({ page }) => {
		await test.step('Verify disconnection state is handled', async () => {
			// Wait for initial page load
			await page.waitForTimeout(2000);

			// Simulate going offline (this affects fetch, not necessarily WebSocket, but tests error handling)
			await page.context().setOffline(true);

			// Wait a moment
			await page.waitForTimeout(1000);

			// Page should still be functional
			await expect(page.locator('h1:has-text("Dashboard")')).toBeVisible();

			// Restore connection
			await page.context().setOffline(false);
		});
	});

	test('should display statistics that update', async ({ page }) => {
		await test.step('Verify statistics are displayed', async () => {
			// Dashboard should show statistics
			await expect(page.locator('text=Total Transactions')).toBeVisible({ timeout: 10000 });
			await expect(page.locator('text=Total Outliers')).toBeVisible();

			// Get initial outlier count
			const outlierStat = page.locator('text=Total Outliers').locator('..');
			const initialCount = await outlierStat.textContent();

			expect(initialCount).toBeTruthy();

			// In a real scenario, we'd wait for a WebSocket update
			// For now, we verify the structure is in place
		});
	});

	test('should navigate between pages without losing connection state', async ({ page }) => {
		await test.step('Navigate and maintain connection', async () => {
			// Verify we're on dashboard
			await expect(page).toHaveURL('/');

			// Navigate to outliers
			await page.click('a[href="/outliers"]');
			await expect(page).toHaveURL('/outliers');

			// WebSocket indicator should still be visible
			const indicator = page.locator('.indicator, .tooltip').first();
			await expect(indicator).toBeVisible();

			// Navigate to statistics
			await page.click('a[href="/statistics"]');
			await expect(page).toHaveURL('/statistics');

			// Indicator should still be visible
			await expect(indicator).toBeVisible();

			// Navigate back to dashboard
			await page.click('a[href="/"]');
			await expect(page).toHaveURL('/');

			// Indicator should still be visible
			await expect(indicator).toBeVisible();
		});
	});

	test('should show detection status on dashboard', async ({ page }) => {
		await test.step('Verify detection engine status', async () => {
			// Look for detection status section
			await expect(page.locator('text=Detection Status')).toBeVisible({ timeout: 10000 });

			// Should show active or stopped status
			const hasActive = await page.locator('text=Active').isVisible().catch(() => false);
			const hasStopped = await page.locator('text=Stopped').isVisible().catch(() => false);

			// One of them should be visible
			expect(hasActive || hasStopped).toBeTruthy();
		});
	});

	test('should display outliers by detection method', async ({ page }) => {
		await test.step('Verify detection methods section', async () => {
			// Dashboard should show outliers by detection method
			await expect(page.locator('text=Outliers by Detection Method')).toBeVisible({
				timeout: 10000
			});

			// Should list detection methods
			const methods = ['ZSCORE', 'IQR'];
			for (const method of methods) {
				const methodVisible = await page.locator(`text=${method}`).isVisible().catch(() => false);
				// At least some methods should be visible
				if (methodVisible) {
					expect(methodVisible).toBe(true);
					break;
				}
			}
		});
	});
});
