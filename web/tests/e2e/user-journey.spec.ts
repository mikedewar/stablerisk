import { test, expect } from '@playwright/test';

/**
 * E2E Test: Complete User Journey
 *
 * Tests the full user flow through the application:
 * 1. Login with credentials
 * 2. View dashboard statistics
 * 3. Navigate to outliers page
 * 4. View outlier details
 * 5. Logout
 */

test.describe('Complete User Journey', () => {
	test.beforeEach(async ({ page }) => {
		// Navigate to the application
		await page.goto('/');
	});

	test('should complete full user journey: login → dashboard → outliers → details → logout', async ({
		page
	}) => {
		// Step 1: Login
		await test.step('Login with credentials', async () => {
			// Should be redirected to login page if not authenticated
			await expect(page).toHaveURL('/login');

			// Fill in login form
			await page.fill('input[name="username"]', 'admin');
			await page.fill('input[name="password"]', 'changeme123');

			// Submit form
			await page.click('button[type="submit"]');

			// Should redirect to dashboard after successful login
			await expect(page).toHaveURL('/');

			// Verify logged in state - should see user menu
			await expect(page.locator('text=admin')).toBeVisible();
		});

		// Step 2: View Dashboard
		await test.step('View dashboard statistics', async () => {
			// Verify dashboard title
			await expect(page.locator('h1:has-text("Dashboard")')).toBeVisible();

			// Wait for statistics to load
			await page.waitForSelector('text=Total Transactions', { timeout: 10000 });

			// Verify key metrics are displayed
			await expect(page.locator('text=Total Transactions')).toBeVisible();
			await expect(page.locator('text=Total Outliers')).toBeVisible();

			// Verify detection status
			await expect(page.locator('text=Detection Status')).toBeVisible();
		});

		// Step 3: Navigate to Outliers Page
		await test.step('Navigate to outliers page', async () => {
			// Click on Outliers in navigation
			await page.click('a[href="/outliers"]');

			// Verify we're on the outliers page
			await expect(page).toHaveURL('/outliers');
			await expect(page.locator('h1:has-text("Outliers")')).toBeVisible();

			// Wait for outliers table to load
			await page.waitForSelector('table', { timeout: 10000 });
		});

		// Step 4: View Outlier Details
		await test.step('View outlier details', async () => {
			// Wait for table rows to be visible
			const detailsButton = page.locator('button:has-text("Details")').first();

			// Check if there are any outliers to view
			const hasOutliers = await detailsButton.isVisible({ timeout: 5000 }).catch(() => false);

			if (hasOutliers) {
				// Click the first Details button
				await detailsButton.click();

				// Verify modal opened
				await expect(page.locator('text=Outlier Details')).toBeVisible();

				// Verify details are displayed
				await expect(page.locator('text=Type')).toBeVisible();
				await expect(page.locator('text=Severity')).toBeVisible();

				// Close modal
				await page.click('button:has-text("Close")');

				// Verify modal closed
				await expect(page.locator('text=Outlier Details')).not.toBeVisible();
			}
		});

		// Step 5: Logout
		await test.step('Logout', async () => {
			// Click on user dropdown (find by username)
			await page.click('text=admin');

			// Click logout button
			await page.click('button:has-text("Logout")');

			// Should redirect to login page
			await expect(page).toHaveURL('/login');

			// Verify logged out - login form should be visible
			await expect(page.locator('text=Sign In')).toBeVisible();
		});
	});

	test('should handle invalid login credentials', async ({ page }) => {
		await test.step('Attempt login with invalid credentials', async () => {
			await expect(page).toHaveURL('/login');

			// Fill in wrong credentials
			await page.fill('input[name="username"]', 'wronguser');
			await page.fill('input[name="password"]', 'wrongpass');

			// Submit form
			await page.click('button[type="submit"]');

			// Should stay on login page
			await expect(page).toHaveURL('/login');

			// Should show error message
			await expect(page.locator('.alert-error')).toBeVisible({ timeout: 5000 });
		});
	});

	test('should redirect to login when not authenticated', async ({ page }) => {
		await test.step('Access protected route without login', async () => {
			// Try to navigate to outliers page directly
			await page.goto('/outliers');

			// Should redirect to login
			await expect(page).toHaveURL('/login');
		});
	});

	test('should persist session after page reload', async ({ page }) => {
		await test.step('Login and reload page', async () => {
			// Login first
			await page.fill('input[name="username"]', 'admin');
			await page.fill('input[name="password"]', 'changeme123');
			await page.click('button[type="submit"]');

			// Wait for dashboard
			await expect(page).toHaveURL('/');
			await expect(page.locator('text=admin')).toBeVisible();

			// Reload page
			await page.reload();

			// Should still be logged in
			await expect(page).toHaveURL('/');
			await expect(page.locator('text=admin')).toBeVisible();
		});
	});

	test('should navigate between all pages', async ({ page }) => {
		await test.step('Login and navigate through all pages', async () => {
			// Login
			await page.fill('input[name="username"]', 'admin');
			await page.fill('input[name="password"]', 'changeme123');
			await page.click('button[type="submit"]');
			await expect(page).toHaveURL('/');

			// Navigate to Outliers
			await page.click('a[href="/outliers"]');
			await expect(page).toHaveURL('/outliers');
			await expect(page.locator('h1:has-text("Outliers")')).toBeVisible();

			// Navigate to Statistics
			await page.click('a[href="/statistics"]');
			await expect(page).toHaveURL('/statistics');
			await expect(page.locator('h1:has-text("Statistics")')).toBeVisible();

			// Navigate back to Dashboard
			await page.click('a[href="/"]');
			await expect(page).toHaveURL('/');
			await expect(page.locator('h1:has-text("Dashboard")')).toBeVisible();
		});
	});
});
