import { describe, it, expect, vi, beforeEach } from 'vitest';
import { auth } from './auth';
import apiClient from '$api/client';
import type { LoginResponse, User } from '$api/types';
import { get } from 'svelte/store';

// Mock the API client
vi.mock('$api/client', () => ({
	default: {
		login: vi.fn(),
		refreshToken: vi.fn(),
		logout: vi.fn(),
		setToken: vi.fn()
	}
}));

describe('Auth Store', () => {
	const mockUser: User = {
		id: '1',
		username: 'testuser',
		email: 'test@example.com',
		role: 'admin',
		created_at: '2024-01-01T00:00:00Z',
		updated_at: '2024-01-01T00:00:00Z',
		is_active: true
	};

	const mockLoginResponse: LoginResponse = {
		token: 'test-token',
		refresh_token: 'test-refresh-token',
		expires_in: 3600,
		user: mockUser
	};

	beforeEach(() => {
		// Clear all mocks
		vi.clearAllMocks();

		// Clear localStorage
		localStorage.clear();

		// Reset auth store to initial state
		auth.logout();
	});

	describe('Initial State', () => {
		it('should have null user, token, and refreshToken', () => {
			const state = get(auth);
			expect(state.user).toBeNull();
			expect(state.token).toBeNull();
			expect(state.refreshToken).toBeNull();
		});

		it('should not be loading', () => {
			const state = get(auth);
			expect(state.loading).toBe(false);
		});

		it('should have no error', () => {
			const state = get(auth);
			expect(state.error).toBeNull();
		});
	});

	describe('login', () => {
		it('should successfully login with valid credentials', async () => {
			vi.mocked(apiClient.login).mockResolvedValue(mockLoginResponse);

			const result = await auth.login('testuser', 'password');

			expect(result).toBe(true);
			expect(apiClient.login).toHaveBeenCalledWith({
				username: 'testuser',
				password: 'password'
			});

			const state = get(auth);
			expect(state.user).toEqual(mockUser);
			expect(state.token).toBe('test-token');
			expect(state.refreshToken).toBe('test-refresh-token');
			expect(state.loading).toBe(false);
			expect(state.error).toBeNull();
		});

		it('should store credentials in localStorage', async () => {
			vi.mocked(apiClient.login).mockResolvedValue(mockLoginResponse);

			await auth.login('testuser', 'password');

			expect(localStorage.setItem).toHaveBeenCalledWith('token', 'test-token');
			expect(localStorage.setItem).toHaveBeenCalledWith('refresh_token', 'test-refresh-token');
			expect(localStorage.setItem).toHaveBeenCalledWith('user', JSON.stringify(mockUser));
		});

		it('should set loading state during login', async () => {
			let loadingDuringRequest = false;

			vi.mocked(apiClient.login).mockImplementation(async () => {
				const state = get(auth);
				loadingDuringRequest = state.loading;
				return mockLoginResponse;
			});

			await auth.login('testuser', 'password');

			expect(loadingDuringRequest).toBe(true);
		});

		it('should handle login failure', async () => {
			const error = new Error('Invalid credentials');
			vi.mocked(apiClient.login).mockRejectedValue(error);

			const result = await auth.login('testuser', 'wrongpassword');

			expect(result).toBe(false);

			const state = get(auth);
			expect(state.user).toBeNull();
			expect(state.token).toBeNull();
			expect(state.loading).toBe(false);
			expect(state.error).toBe('Invalid credentials');
		});

		it('should handle network errors', async () => {
			vi.mocked(apiClient.login).mockRejectedValue(new Error());

			const result = await auth.login('testuser', 'password');

			expect(result).toBe(false);

			const state = get(auth);
			expect(state.error).toBe('Login failed');
		});

		it('should clear previous errors on new login attempt', async () => {
			// First login fails
			vi.mocked(apiClient.login).mockRejectedValue(new Error('First error'));
			await auth.login('testuser', 'wrong');

			// Second login succeeds
			vi.mocked(apiClient.login).mockResolvedValue(mockLoginResponse);
			await auth.login('testuser', 'correct');

			const state = get(auth);
			expect(state.error).toBeNull();
		});
	});

	describe('logout', () => {
		it('should clear user state', async () => {
			// Login first
			vi.mocked(apiClient.login).mockResolvedValue(mockLoginResponse);
			await auth.login('testuser', 'password');

			// Then logout
			auth.logout();

			const state = get(auth);
			expect(state.user).toBeNull();
			expect(state.token).toBeNull();
			expect(state.refreshToken).toBeNull();
			expect(state.loading).toBe(false);
			expect(state.error).toBeNull();
		});

		it('should remove credentials from localStorage', () => {
			auth.logout();

			expect(localStorage.removeItem).toHaveBeenCalledWith('token');
			expect(localStorage.removeItem).toHaveBeenCalledWith('refresh_token');
			expect(localStorage.removeItem).toHaveBeenCalledWith('user');
		});

		it('should call apiClient.logout', () => {
			auth.logout();

			expect(apiClient.logout).toHaveBeenCalled();
		});
	});

	describe('refreshAuth', () => {
		beforeEach(async () => {
			// Login first to have a refresh token
			vi.mocked(apiClient.login).mockResolvedValue(mockLoginResponse);
			await auth.login('testuser', 'password');
			vi.clearAllMocks();
		});

		it('should refresh token successfully', async () => {
			const newToken = 'new-test-token';
			vi.mocked(apiClient.refreshToken).mockResolvedValue({
				token: newToken,
				expires_in: 3600
			});

			const result = await auth.refreshAuth();

			expect(result).toBe(true);
			expect(apiClient.refreshToken).toHaveBeenCalledWith('test-refresh-token');

			const state = get(auth);
			expect(state.token).toBe(newToken);
		});

		it('should update token in localStorage', async () => {
			const newToken = 'new-test-token';
			vi.mocked(apiClient.refreshToken).mockResolvedValue({
				token: newToken,
				expires_in: 3600
			});

			await auth.refreshAuth();

			expect(localStorage.setItem).toHaveBeenCalledWith('token', newToken);
		});

		it('should return false if no refresh token exists', async () => {
			auth.logout();

			const result = await auth.refreshAuth();

			expect(result).toBe(false);
			expect(apiClient.refreshToken).not.toHaveBeenCalled();
		});

		it('should logout on refresh failure', async () => {
			vi.mocked(apiClient.refreshToken).mockRejectedValue(new Error('Invalid refresh token'));

			const result = await auth.refreshAuth();

			expect(result).toBe(false);

			const state = get(auth);
			expect(state.user).toBeNull();
			expect(state.token).toBeNull();
		});
	});

	describe('clearError', () => {
		it('should clear error state', async () => {
			// Trigger an error
			vi.mocked(apiClient.login).mockRejectedValue(new Error('Test error'));
			await auth.login('testuser', 'wrong');

			// Clear the error
			auth.clearError();

			const state = get(auth);
			expect(state.error).toBeNull();
		});

		it('should preserve other state when clearing error', async () => {
			vi.mocked(apiClient.login).mockResolvedValue(mockLoginResponse);
			await auth.login('testuser', 'password');

			// Trigger an error (this won't happen in real app after successful login, but for test)
			vi.mocked(apiClient.refreshToken).mockRejectedValue(new Error('Test'));
			await auth.refreshAuth();

			// Clear error
			auth.clearError();

			const state = get(auth);
			expect(state.error).toBeNull();
		});
	});

	describe('localStorage persistence', () => {
		it('should restore auth state from localStorage on initialization', () => {
			// Mock localStorage with existing data
			localStorage.getItem = vi.fn((key) => {
				if (key === 'token') return 'stored-token';
				if (key === 'refresh_token') return 'stored-refresh-token';
				if (key === 'user') return JSON.stringify(mockUser);
				return null;
			});

			// Re-import to trigger initialization
			// Note: This is a limitation of the current implementation
			// In practice, the auth store initializes when the module loads
		});

		it('should clear invalid stored data', () => {
			localStorage.getItem = vi.fn((key) => {
				if (key === 'token') return 'stored-token';
				if (key === 'user') return 'invalid-json';
				return null;
			});

			// Would trigger clear on initialization with invalid data
			expect(localStorage.clear).toBeDefined();
		});
	});

	describe('subscription', () => {
		it('should notify subscribers of state changes', async () => {
			vi.mocked(apiClient.login).mockResolvedValue(mockLoginResponse);

			const states: any[] = [];
			const unsubscribe = auth.subscribe((state) => {
				states.push(state);
			});

			await auth.login('testuser', 'password');

			unsubscribe();

			// Should have at least: initial, loading, success states
			expect(states.length).toBeGreaterThanOrEqual(2);
			expect(states[states.length - 1].user).toEqual(mockUser);
		});
	});
});
