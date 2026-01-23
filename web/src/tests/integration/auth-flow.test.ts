import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { auth } from '$stores/auth';
import apiClient from '$api/client';

/**
 * Integration Test: Authentication Flow
 *
 * Tests the complete authentication workflow including:
 * - Login flow (auth store + API client)
 * - Token management across components
 * - Token refresh workflow
 * - Logout flow
 */

describe('Integration: Authentication Flow', () => {
	const mockUser = {
		id: '1',
		username: 'testuser',
		role: 'admin'
	};

	const mockLoginResponse = {
		user: mockUser,
		token: 'test-access-token',
		refresh_token: 'test-refresh-token',
		expires_in: 3600
	};

	// Shared storage object for localStorage mock
	let storage: Record<string, string> = {};

	beforeEach(() => {
		// Reset storage
		storage = {};

		// Setup a functional localStorage mock
		vi.spyOn(Storage.prototype, 'getItem').mockImplementation((key: string) => {
			return storage[key] || null;
		});
		vi.spyOn(Storage.prototype, 'setItem').mockImplementation((key: string, value: string) => {
			storage[key] = value;
		});
		vi.spyOn(Storage.prototype, 'removeItem').mockImplementation((key: string) => {
			delete storage[key];
		});
		vi.spyOn(Storage.prototype, 'clear').mockImplementation(() => {
			Object.keys(storage).forEach((key) => delete storage[key]);
		});

		// Clear API client state
		apiClient.setToken(null);

		// Reset auth store - mock logout to avoid API call
		vi.mocked(global.fetch).mockResolvedValueOnce({
			ok: true,
			status: 200,
			json: async () => ({})
		} as Response);
		auth.logout();

		// Setup fetch mock for tests
		global.fetch = vi.fn();
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	describe('Login Flow Integration', () => {
		it('should integrate auth store and API client during login', async () => {
			// Mock successful login API call
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockLoginResponse
			} as Response);

			// Execute login through auth store
			const success = await auth.login('testuser', 'password123');

			// Verify integration succeeded
			expect(success).toBe(true);

			// Verify auth store was updated
			const state = get(auth);
			expect(state.user).toEqual(mockUser);
			expect(state.token).toBe('test-access-token');

			// Verify API client received the token
			expect(apiClient.getToken()).toBe('test-access-token');
		});

		it('should make authenticated API calls after login', async () => {
			// Login first
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockLoginResponse
			} as Response);

			await auth.login('testuser', 'password123');

			// Mock statistics API call
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					total_transactions: 1000,
					total_outliers: 50
				})
			} as Response);

			// Make authenticated API call
			await apiClient.getStatistics();

			// Verify the request included Authorization header
			const calls = vi.mocked(global.fetch).mock.calls;
			const statisticsCall = calls[1]; // Second call (first was login)
			const [, options] = statisticsCall;

			expect((options?.headers as Record<string, string>)['Authorization']).toBe(
				'Bearer test-access-token'
			);
		});

		it('should handle login failure consistently', async () => {
			// Mock failed login
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: false,
				status: 401,
				json: async () => ({ error: 'Invalid credentials' })
			} as Response);

			const success = await auth.login('wronguser', 'wrongpass');

			// Verify login failed
			expect(success).toBe(false);

			// Verify auth store reflects failure
			const state = get(auth);
			expect(state.user).toBeNull();
			expect(state.error).toBeTruthy();

			// Verify API client has no token
			expect(apiClient.getToken()).toBeNull();
		});
	});

	describe('Token Refresh Integration', () => {
		it('should refresh token across all components', async () => {
			// Setup initial logged-in state
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockLoginResponse
			} as Response);

			await auth.login('testuser', 'password123');

			// Verify initial token
			expect(get(auth).token).toBe('test-access-token');
			expect(apiClient.getToken()).toBe('test-access-token');

			// Mock refresh token API call
			const newToken = 'new-access-token';
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({
					token: newToken,
					expires_in: 3600
				})
			} as Response);

			// Execute token refresh
			const success = await auth.refreshAuth();

			// Verify refresh succeeded
			expect(success).toBe(true);

			// Verify auth store has new token
			expect(get(auth).token).toBe(newToken);

			// Verify API client has new token
			expect(apiClient.getToken()).toBe(newToken);
		});

		it('should handle refresh failure by logging out', async () => {
			// Setup initial logged-in state
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockLoginResponse
			} as Response);

			await auth.login('testuser', 'password123');

			// Verify logged in
			expect(get(auth).user).toBeTruthy();

			// Mock failed refresh
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: false,
				status: 401,
				json: async () => ({ error: 'Invalid refresh token' })
			} as Response);

			// Mock logout call (refresh will trigger logout on failure)
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({})
			} as Response);

			// Execute token refresh
			const success = await auth.refreshAuth();

			// Verify refresh failed and logged out
			expect(success).toBe(false);
			expect(get(auth).user).toBeNull();
			expect(apiClient.getToken()).toBeNull();
		});
	});

	describe('Logout Integration', () => {
		it('should clear state across all components', async () => {
			// Setup initial logged-in state
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockLoginResponse
			} as Response);

			await auth.login('testuser', 'password123');

			// Verify logged in
			expect(get(auth).user).toBeTruthy();
			expect(apiClient.getToken()).toBeTruthy();

			// Mock logout API call
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => ({})
			} as Response);

			// Execute logout
			await auth.logout();

			// Verify auth store cleared
			const state = get(auth);
			expect(state.user).toBeNull();
			expect(state.token).toBeNull();

			// Verify API client token cleared
			expect(apiClient.getToken()).toBeNull();
		});
	});

	describe('Error Handling Integration', () => {
		it('should handle network errors gracefully', async () => {
			// Mock network error
			vi.mocked(global.fetch).mockRejectedValueOnce(new Error('Network error'));

			const success = await auth.login('testuser', 'password123');

			// Verify login failed
			expect(success).toBe(false);

			// Verify error is captured
			const state = get(auth);
			expect(state.error).toBeTruthy();
			expect(state.user).toBeNull();
		});

		it('should handle API errors consistently', async () => {
			// Mock API error response
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: false,
				status: 500,
				json: async () => ({ error: 'Internal server error' })
			} as Response);

			const success = await auth.login('testuser', 'password123');

			// Verify login failed
			expect(success).toBe(false);
			expect(get(auth).user).toBeNull();
			expect(apiClient.getToken()).toBeNull();
		});
	});

	describe('Concurrent API Calls', () => {
		it('should handle multiple authenticated requests with same token', async () => {
			// Login first
			vi.mocked(global.fetch).mockResolvedValueOnce({
				ok: true,
				status: 200,
				json: async () => mockLoginResponse
			} as Response);

			await auth.login('testuser', 'password123');

			// Mock multiple API responses
			vi.mocked(global.fetch)
				.mockResolvedValueOnce({
					ok: true,
					status: 200,
					json: async () => ({ total_transactions: 1000 })
				} as Response)
				.mockResolvedValueOnce({
					ok: true,
					status: 200,
					json: async () => ({ outliers: [], total: 0 })
				} as Response);

			// Make multiple concurrent API calls
			const [stats, outliers] = await Promise.all([
				apiClient.getStatistics(),
				apiClient.listOutliers({})
			]);

			// Verify both calls succeeded
			expect(stats).toBeTruthy();
			expect(outliers).toBeTruthy();

			// Verify both calls used the auth token
			const calls = vi.mocked(global.fetch).mock.calls;
			const statsCall = calls[1];
			const outliersCall = calls[2];

			expect((statsCall[1]?.headers as Record<string, string>)['Authorization']).toBe(
				'Bearer test-access-token'
			);
			expect((outliersCall[1]?.headers as Record<string, string>)['Authorization']).toBe(
				'Bearer test-access-token'
			);
		});
	});
});
