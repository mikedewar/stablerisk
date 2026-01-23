import { describe, it, expect, vi, beforeEach } from 'vitest';
import { APIClient } from './client';
import type {
	LoginRequest,
	LoginResponse,
	User,
	Outlier,
	OutlierListResponse,
	Statistics,
	HealthResponse
} from './types';

describe('APIClient', () => {
	let client: APIClient;
	let fetchMock: any;

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
		// Clear localStorage
		localStorage.clear();

		// Create new client
		client = new APIClient('/api/v1');

		// Mock fetch
		fetchMock = vi.fn();
		global.fetch = fetchMock;
	});

	describe('Constructor and Token Management', () => {
		it('should initialize with base URL', () => {
			expect(client).toBeDefined();
			expect(client.getToken()).toBeNull();
		});

		it('should use default base URL if not provided', () => {
			const defaultClient = new APIClient();
			expect(defaultClient).toBeDefined();
		});

		it('should set token', () => {
			client.setToken('test-token');
			expect(client.getToken()).toBe('test-token');
		});

		it('should store token in localStorage when set', () => {
			client.setToken('test-token');
			expect(localStorage.setItem).toHaveBeenCalledWith('token', 'test-token');
		});

		it('should remove token from localStorage when cleared', () => {
			client.setToken('test-token');
			client.setToken(null);
			expect(localStorage.removeItem).toHaveBeenCalledWith('token');
		});

		it('should clear token', () => {
			client.setToken('test-token');
			client.setToken(null);
			expect(client.getToken()).toBeNull();
		});
	});

	describe('Authentication Methods', () => {
		describe('login', () => {
			it('should login successfully', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockLoginResponse
				});

				const credentials: LoginRequest = {
					username: 'testuser',
					password: 'password'
				};

				const response = await client.login(credentials);

				expect(fetchMock).toHaveBeenCalledWith(
					'/api/v1/auth/login',
					expect.objectContaining({
						method: 'POST',
						headers: {
							'Content-Type': 'application/json'
						},
						body: JSON.stringify(credentials)
					})
				);

				expect(response).toEqual(mockLoginResponse);
				expect(client.getToken()).toBe('test-token');
			});

			it('should not include Authorization header for login', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockLoginResponse
				});

				await client.login({ username: 'test', password: 'pass' });

				const callArgs = fetchMock.mock.calls[0][1];
				expect(callArgs.headers.Authorization).toBeUndefined();
			});

			it('should handle login failure', async () => {
				fetchMock.mockResolvedValue({
					ok: false,
					status: 401,
					statusText: 'Unauthorized',
					json: async () => ({
						error: 'invalid_credentials',
						message: 'Invalid username or password'
					})
				});

				await expect(
					client.login({ username: 'wrong', password: 'wrong' })
				).rejects.toThrow('Invalid username or password');
			});

			it('should handle network errors', async () => {
				fetchMock.mockResolvedValue({
					ok: false,
					status: 500,
					statusText: 'Internal Server Error',
					json: async () => {
						throw new Error('Parse error');
					}
				});

				await expect(
					client.login({ username: 'test', password: 'test' })
				).rejects.toThrow('HTTP 500: Internal Server Error');
			});
		});

		describe('refreshToken', () => {
			it('should refresh token successfully', async () => {
				const newToken = 'new-test-token';
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => ({
						token: newToken,
						expires_in: 3600
					})
				});

				const response = await client.refreshToken('refresh-token');

				expect(fetchMock).toHaveBeenCalledWith(
					'/api/v1/auth/refresh',
					expect.objectContaining({
						method: 'POST',
						body: JSON.stringify({ refresh_token: 'refresh-token' })
					})
				);

				expect(response.token).toBe(newToken);
				expect(client.getToken()).toBe(newToken);
			});

			it('should not include Authorization header for refresh', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => ({ token: 'new-token', expires_in: 3600 })
				});

				await client.refreshToken('refresh-token');

				const callArgs = fetchMock.mock.calls[0][1];
				expect(callArgs.headers.Authorization).toBeUndefined();
			});

			it('should handle refresh failure', async () => {
				fetchMock.mockResolvedValue({
					ok: false,
					status: 401,
					statusText: 'Unauthorized',
					json: async () => ({
						error: 'invalid_refresh_token',
						message: 'Invalid refresh token'
					})
				});

				await expect(client.refreshToken('invalid-token')).rejects.toThrow(
					'Invalid refresh token'
				);
			});
		});

		describe('getProfile', () => {
			beforeEach(() => {
				client.setToken('test-token');
			});

			it('should fetch user profile', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockUser
				});

				const profile = await client.getProfile();

				expect(fetchMock).toHaveBeenCalledWith(
					'/api/v1/auth/profile',
					expect.objectContaining({
						method: 'GET',
						headers: expect.objectContaining({
							Authorization: 'Bearer test-token'
						})
					})
				);

				expect(profile).toEqual(mockUser);
			});

			it('should include Authorization header', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockUser
				});

				await client.getProfile();

				const callArgs = fetchMock.mock.calls[0][1];
				expect(callArgs.headers.Authorization).toBe('Bearer test-token');
			});
		});

		describe('logout', () => {
			it('should clear token', () => {
				client.setToken('test-token');
				client.logout();
				expect(client.getToken()).toBeNull();
			});
		});
	});

	describe('Outliers Methods', () => {
		beforeEach(() => {
			client.setToken('test-token');
		});

		describe('listOutliers', () => {
			const mockOutliersResponse: OutlierListResponse = {
				outliers: [],
				total: 0,
				page: 1,
				limit: 20,
				total_pages: 0
			};

			it('should fetch outliers list without parameters', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockOutliersResponse
				});

				const response = await client.listOutliers();

				expect(fetchMock).toHaveBeenCalledWith(
					'/api/v1/outliers',
					expect.objectContaining({
						method: 'GET'
					})
				);

				expect(response).toEqual(mockOutliersResponse);
			});

			it('should include query parameters', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockOutliersResponse
				});

				await client.listOutliers({
					page: 2,
					limit: 50,
					type: 'zscore',
					severity: 'high',
					acknowledged: false
				});

				const url = fetchMock.mock.calls[0][0];
				expect(url).toContain('page=2');
				expect(url).toContain('limit=50');
				expect(url).toContain('type=zscore');
				expect(url).toContain('severity=high');
				expect(url).toContain('acknowledged=false');
			});

			it('should include address filter', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockOutliersResponse
				});

				await client.listOutliers({ address: '0x123' });

				const url = fetchMock.mock.calls[0][0];
				expect(url).toContain('address=0x123');
			});

			it('should include date range filters', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockOutliersResponse
				});

				await client.listOutliers({
					from: '2024-01-01',
					to: '2024-01-31'
				});

				const url = fetchMock.mock.calls[0][0];
				expect(url).toContain('from=2024-01-01');
				expect(url).toContain('to=2024-01-31');
			});
		});

		describe('getOutlier', () => {
			const mockOutlier: Outlier = {
				id: '1',
				detected_at: '2024-01-01T00:00:00Z',
				type: 'zscore',
				severity: 'high',
				address: '0x123',
				details: {},
				acknowledged: false
			};

			it('should fetch single outlier', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockOutlier
				});

				const outlier = await client.getOutlier('1');

				expect(fetchMock).toHaveBeenCalledWith(
					'/api/v1/outliers/1',
					expect.objectContaining({
						method: 'GET'
					})
				);

				expect(outlier).toEqual(mockOutlier);
			});

			it('should handle not found', async () => {
				fetchMock.mockResolvedValue({
					ok: false,
					status: 404,
					statusText: 'Not Found',
					json: async () => ({
						error: 'not_found',
						message: 'Outlier not found'
					})
				});

				await expect(client.getOutlier('999')).rejects.toThrow('Outlier not found');
			});
		});

		describe('acknowledgeOutlier', () => {
			it('should acknowledge outlier', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => ({})
				});

				await client.acknowledgeOutlier('1', { notes: 'Reviewed' });

				expect(fetchMock).toHaveBeenCalledWith(
					'/api/v1/outliers/1/acknowledge',
					expect.objectContaining({
						method: 'POST',
						body: JSON.stringify({ notes: 'Reviewed' })
					})
				);
			});

			it('should require authentication', async () => {
				client.setToken(null);
				fetchMock.mockResolvedValue({
					ok: false,
					status: 401,
					statusText: 'Unauthorized',
					json: async () => ({
						error: 'unauthorized',
						message: 'Authentication required'
					})
				});

				await expect(
					client.acknowledgeOutlier('1', { notes: 'Test' })
				).rejects.toThrow('Authentication required');
			});
		});
	});

	describe('Statistics Methods', () => {
		beforeEach(() => {
			client.setToken('test-token');
		});

		describe('getStatistics', () => {
			const mockStats: Statistics = {
				total_transactions: 1000,
				total_outliers: 50,
				outliers_by_severity: {
					low: 10,
					medium: 20,
					high: 15,
					critical: 5
				},
				outliers_by_type: {
					zscore: 20,
					iqr: 15,
					pattern_circulation: 5,
					pattern_fanout: 3,
					pattern_fanin: 3,
					pattern_dormant: 2,
					pattern_velocity: 2
				},
				last_detection_run: '2024-01-01T00:00:00Z',
				detection_running: false
			};

			it('should fetch statistics', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockStats
				});

				const stats = await client.getStatistics();

				expect(fetchMock).toHaveBeenCalledWith(
					'/api/v1/statistics',
					expect.objectContaining({
						method: 'GET'
					})
				);

				expect(stats).toEqual(mockStats);
			});
		});

		describe('getTrends', () => {
			it('should fetch trends with default days', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => ({ trends: [] })
				});

				await client.getTrends();

				expect(fetchMock).toHaveBeenCalledWith(
					'/api/v1/statistics/trends?days=7',
					expect.objectContaining({
						method: 'GET'
					})
				);
			});

			it('should fetch trends with custom days', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => ({ trends: [] })
				});

				await client.getTrends(30);

				const url = fetchMock.mock.calls[0][0];
				expect(url).toContain('days=30');
			});
		});
	});

	describe('Health Methods', () => {
		describe('getHealth', () => {
			const mockHealth: HealthResponse = {
				status: 'healthy',
				timestamp: '2024-01-01T00:00:00Z',
				services: {
					database: { healthy: true },
					api: { healthy: true }
				},
				version: '1.0.0'
			};

			it('should fetch health status', async () => {
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockHealth
				});

				const health = await client.getHealth();

				expect(fetchMock).toHaveBeenCalledWith(
					'/api/v1/health',
					expect.objectContaining({
						method: 'GET'
					})
				);

				expect(health).toEqual(mockHealth);
			});

			it('should not require authentication', async () => {
				client.setToken(null);
				fetchMock.mockResolvedValue({
					ok: true,
					json: async () => mockHealth
				});

				await client.getHealth();

				const callArgs = fetchMock.mock.calls[0][1];
				expect(callArgs.headers.Authorization).toBeUndefined();
			});
		});
	});

	describe('Error Handling', () => {
		beforeEach(() => {
			client.setToken('test-token');
		});

		it('should handle HTTP errors with JSON response', async () => {
			fetchMock.mockResolvedValue({
				ok: false,
				status: 400,
				statusText: 'Bad Request',
				json: async () => ({
					error: 'validation_error',
					message: 'Invalid request parameters'
				})
			});

			await expect(client.getStatistics()).rejects.toThrow(
				'Invalid request parameters'
			);
		});

		it('should handle HTTP errors without JSON response', async () => {
			fetchMock.mockResolvedValue({
				ok: false,
				status: 500,
				statusText: 'Internal Server Error',
				json: async () => {
					throw new Error('Not JSON');
				}
			});

			await expect(client.getStatistics()).rejects.toThrow(
				'HTTP 500: Internal Server Error'
			);
		});

		it('should handle network failures', async () => {
			fetchMock.mockRejectedValue(new Error('Network error'));

			await expect(client.getStatistics()).rejects.toThrow('Network error');
		});

		it('should throw error message when available', async () => {
			fetchMock.mockResolvedValue({
				ok: false,
				status: 403,
				statusText: 'Forbidden',
				json: async () => ({
					error: 'forbidden',
					message: 'You do not have permission to perform this action'
				})
			});

			await expect(client.acknowledgeOutlier('1', { notes: 'test' })).rejects.toThrow(
				'You do not have permission to perform this action'
			);
		});

		it('should fallback to error code when message not available', async () => {
			fetchMock.mockResolvedValue({
				ok: false,
				status: 403,
				statusText: 'Forbidden',
				json: async () => ({
					error: 'forbidden'
				})
			});

			await expect(client.getStatistics()).rejects.toThrow('forbidden');
		});
	});

	describe('Request Headers', () => {
		beforeEach(() => {
			client.setToken('test-token');
		});

		it('should include Content-Type header for all requests', async () => {
			fetchMock.mockResolvedValue({
				ok: true,
				json: async () => ({})
			});

			await client.getStatistics();

			const callArgs = fetchMock.mock.calls[0][1];
			expect(callArgs.headers['Content-Type']).toBe('application/json');
		});

		it('should include Authorization header for authenticated requests', async () => {
			fetchMock.mockResolvedValue({
				ok: true,
				json: async () => ({})
			});

			await client.getStatistics();

			const callArgs = fetchMock.mock.calls[0][1];
			expect(callArgs.headers.Authorization).toBe('Bearer test-token');
		});

		it('should not include Authorization header when no token', async () => {
			client.setToken(null);
			fetchMock.mockResolvedValue({
				ok: true,
				json: async () => mockLoginResponse
			});

			await client.login({ username: 'test', password: 'test' });

			const callArgs = fetchMock.mock.calls[0][1];
			expect(callArgs.headers.Authorization).toBeUndefined();
		});
	});
});
