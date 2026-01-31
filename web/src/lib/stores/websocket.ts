import { writable, derived } from 'svelte/store';
import type { WebSocketMessage, Outlier, Statistics, Severity, OutlierType } from '$api/types';
import { auth } from './auth';

interface WebSocketState {
	connected: boolean;
	reconnecting: boolean;
	error: string | null;
	lastMessage: WebSocketMessage | null;
	filters: {
		severities: Severity[];
		types: OutlierType[];
	};
}

function createWebSocketStore() {
	const { subscribe, set, update } = writable<WebSocketState>({
		connected: false,
		reconnecting: false,
		error: null,
		lastMessage: null,
		filters: {
			severities: [],
			types: []
		}
	});

	let ws: WebSocket | null = null;
	let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
	let reconnectAttempts = 0;
	const maxReconnectDelay = 30000; // 30 seconds

	function getReconnectDelay(): number {
		// Exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (max)
		const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), maxReconnectDelay);
		return delay;
	}

	function connect(token: string) {
		// Only run in browser environment
		if (typeof window === 'undefined') {
			return;
		}

		if (ws && ws.readyState === WebSocket.OPEN) {
			return; // Already connected
		}

		// Determine WebSocket URL based on environment
		// In development: connect directly to backend at ws://localhost:8080
		// In Docker: connect to backend via service name or through current host on port 8080
		let wsHost = window.location.host;

		// If we're on the default web port (3000), connect directly to API port (8080)
		// This handles both development and Docker scenarios
		if (window.location.port === '3000' || window.location.port === '') {
			// Extract hostname without port
			const hostname = window.location.hostname;
			wsHost = `${hostname}:8080`;
		}

		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsURL = `${protocol}//${wsHost}/api/v1/ws?token=${encodeURIComponent(token)}`;

		console.log('Connecting to WebSocket:', wsURL);

		try {
			ws = new WebSocket(wsURL);

			ws.onopen = () => {
				console.log('WebSocket connected');
				reconnectAttempts = 0;
				update((state) => ({
					...state,
					connected: true,
					reconnecting: false,
					error: null
				}));

				// Send subscription filters if any
				const currentState = getCurrentState();
				if (
					currentState.filters.severities.length > 0 ||
					currentState.filters.types.length > 0
				) {
					sendSubscription(currentState.filters);
				}
			};

			ws.onmessage = (event) => {
				try {
					const message: WebSocketMessage = JSON.parse(event.data);
					update((state) => ({ ...state, lastMessage: message }));
				} catch (e) {
					console.error('Failed to parse WebSocket message:', e);
				}
			};

			ws.onerror = (error) => {
				console.error('WebSocket error:', error);
				update((state) => ({ ...state, error: 'Connection error' }));
			};

			ws.onclose = () => {
				console.log('WebSocket closed');
				update((state) => ({ ...state, connected: false }));

				// Attempt to reconnect
				if (reconnectAttempts < 10) {
					// Max 10 attempts
					const delay = getReconnectDelay();
					reconnectAttempts++;

					update((state) => ({ ...state, reconnecting: true }));

					reconnectTimeout = setTimeout(() => {
						const authState = getCurrentAuthState();
						if (authState.token) {
							connect(authState.token);
						}
					}, delay);
				} else {
					update((state) => ({
						...state,
						reconnecting: false,
						error: 'Failed to reconnect after multiple attempts'
					}));
				}
			};
		} catch (error: any) {
			update((state) => ({ ...state, error: error.message || 'Connection failed' }));
		}
	}

	function disconnect() {
		if (reconnectTimeout) {
			clearTimeout(reconnectTimeout);
			reconnectTimeout = null;
		}

		if (ws) {
			ws.close();
			ws = null;
		}

		reconnectAttempts = 0;
		update((state) => ({
			...state,
			connected: false,
			reconnecting: false,
			error: null
		}));
	}

	function sendSubscription(filters: { severities: Severity[]; types: OutlierType[] }) {
		if (ws && ws.readyState === WebSocket.OPEN) {
			const message = {
				type: 'subscribe',
				data: filters,
				timestamp: new Date().toISOString()
			};
			ws.send(JSON.stringify(message));
		}
	}

	function getCurrentState(): WebSocketState {
		let currentState: WebSocketState;
		subscribe((state) => {
			currentState = state;
		})();
		return currentState!;
	}

	function getCurrentAuthState() {
		let authState: any;
		auth.subscribe((state) => {
			authState = state;
		})();
		return authState;
	}

	// Auto-connect when authenticated
	auth.subscribe((authState) => {
		if (authState.token && typeof window !== 'undefined') {
			connect(authState.token);
		} else {
			disconnect();
		}
	});

	return {
		subscribe,

		setFilters(filters: { severities?: Severity[]; types?: OutlierType[] }) {
			update((state) => ({
				...state,
				filters: {
					severities: filters.severities || state.filters.severities,
					types: filters.types || state.filters.types
				}
			}));

			const currentState = getCurrentState();
			sendSubscription(currentState.filters);
		},

		disconnect
	};
}

export const websocket = createWebSocketStore();

// Derived stores for specific message types
export const outlierMessages = derived(websocket, ($ws) =>
	$ws.lastMessage?.type === 'outlier' ? ($ws.lastMessage.data as Outlier) : null
);

export const statisticsMessages = derived(websocket, ($ws) =>
	$ws.lastMessage?.type === 'statistics' ? ($ws.lastMessage.data as Statistics) : null
);

export const systemMessages = derived(websocket, ($ws) =>
	$ws.lastMessage?.type === 'system' ? ($ws.lastMessage.data as { message: string }) : null
);
