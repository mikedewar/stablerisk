import type {
	LoginRequest,
	LoginResponse,
	User,
	Outlier,
	OutlierListRequest,
	OutlierListResponse,
	AcknowledgeRequest,
	Statistics,
	HealthResponse,
	APIError
} from './types';

export class APIClient {
	private baseURL: string;
	private token: string | null = null;

	constructor(baseURL: string = '/api/v1') {
		this.baseURL = baseURL;
		// Load token from localStorage if available (browser only)
		if (typeof window !== 'undefined') {
			this.token = localStorage.getItem('token');
		}
	}

	setToken(token: string | null) {
		this.token = token;
		if (typeof window !== 'undefined') {
			if (token) {
				localStorage.setItem('token', token);
			} else {
				localStorage.removeItem('token');
			}
		}
	}

	getToken(): string | null {
		return this.token;
	}

	private async request<T>(
		method: string,
		path: string,
		body?: any,
		requiresAuth: boolean = true
	): Promise<T> {
		const headers: HeadersInit = {
			'Content-Type': 'application/json'
		};

		if (requiresAuth && this.token) {
			headers['Authorization'] = `Bearer ${this.token}`;
		}

		const options: RequestInit = {
			method,
			headers
		};

		if (body) {
			options.body = JSON.stringify(body);
		}

		const response = await fetch(`${this.baseURL}${path}`, options);

		if (!response.ok) {
			const error: APIError = await response.json().catch(() => ({
				error: 'network_error',
				message: `HTTP ${response.status}: ${response.statusText}`
			}));
			throw new Error(error.message || error.error);
		}

		return response.json();
	}

	// Authentication
	async login(credentials: LoginRequest): Promise<LoginResponse> {
		const response = await this.request<LoginResponse>('POST', '/auth/login', credentials, false);
		this.setToken(response.token);
		return response;
	}

	async refreshToken(refreshToken: string): Promise<{ token: string; expires_in: number }> {
		const response = await this.request<{ token: string; expires_in: number }>(
			'POST',
			'/auth/refresh',
			{ refresh_token: refreshToken },
			false
		);
		this.setToken(response.token);
		return response;
	}

	async getProfile(): Promise<User> {
		return this.request<User>('GET', '/auth/profile');
	}

	logout() {
		this.setToken(null);
	}

	// Outliers
	async listOutliers(params?: OutlierListRequest): Promise<OutlierListResponse> {
		const queryParams = new URLSearchParams();
		if (params?.page) queryParams.append('page', params.page.toString());
		if (params?.limit) queryParams.append('limit', params.limit.toString());
		if (params?.type) queryParams.append('type', params.type);
		if (params?.severity) queryParams.append('severity', params.severity);
		if (params?.address) queryParams.append('address', params.address);
		if (params?.acknowledged !== undefined)
			queryParams.append('acknowledged', params.acknowledged.toString());
		if (params?.from) queryParams.append('from', params.from);
		if (params?.to) queryParams.append('to', params.to);

		const query = queryParams.toString();
		return this.request<OutlierListResponse>('GET', `/outliers${query ? `?${query}` : ''}`);
	}

	async getOutlier(id: string): Promise<Outlier> {
		return this.request<Outlier>('GET', `/outliers/${id}`);
	}

	async acknowledgeOutlier(id: string, data: AcknowledgeRequest): Promise<void> {
		await this.request('POST', `/outliers/${id}/acknowledge`, data);
	}

	// Statistics
	async getStatistics(): Promise<Statistics> {
		return this.request<Statistics>('GET', '/statistics');
	}

	async getTrends(days: number = 7): Promise<any> {
		return this.request('GET', `/statistics/trends?days=${days}`);
	}

	// Health
	async getHealth(): Promise<HealthResponse> {
		return this.request<HealthResponse>('GET', '/health', undefined, false);
	}
}

export const apiClient = new APIClient();
export default apiClient;
