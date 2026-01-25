import { Page } from '@playwright/test';

/**
 * Helper function to login that properly triggers Svelte reactivity
 */
export async function login(page: Page, username: string = 'admin', password: string = 'changeme123') {
	await page.goto('/login');

	// Set values using evaluate to properly trigger Svelte reactivity
	await page.evaluate(({ username, password }) => {
		const usernameInput = document.querySelector('input[name="username"]') as HTMLInputElement;
		const passwordInput = document.querySelector('input[name="password"]') as HTMLInputElement;

		if (usernameInput && passwordInput) {
			// Set value and dispatch input event
			usernameInput.value = username;
			usernameInput.dispatchEvent(new Event('input', { bubbles: true }));

			passwordInput.value = password;
			passwordInput.dispatchEvent(new Event('input', { bubbles: true }));
		}
	}, { username, password });

	// Wait for button to be enabled
	await page.waitForTimeout(200);

	// Force click the submit button (bypassing disabled state for testing)
	// This is needed because Playwright doesn't properly trigger Svelte reactivity
	await page.locator('button[type="submit"]').click({ force: true });
}
