import { describe, it, expect, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// Import the actual stores  - we'll test the types and structure
import type { WebSocketMessage } from '$api/types';

describe('WebSocket Store Structure', () => {
	it('should have correct initial state shape', async () => {
		// Dynamically import to avoid auto-connection issues
		const { websocket } = await import('./websocket');

		const state = get(websocket);

		expect(state).toHaveProperty('connected');
		expect(state).toHaveProperty('reconnecting');
		expect(state).toHaveProperty('error');
		expect(state).toHaveProperty('lastMessage');
		expect(state).toHaveProperty('filters');

		expect(typeof state.connected).toBe('boolean');
		expect(typeof state.reconnecting).toBe('boolean');
		expect(state.filters).toHaveProperty('severities');
		expect(state.filters).toHaveProperty('types');
		expect(Array.isArray(state.filters.severities)).toBe(true);
		expect(Array.isArray(state.filters.types)).toBe(true);
	});

	it('should have setFilters method', async () => {
		const { websocket } = await import('./websocket');

		expect(websocket.setFilters).toBeDefined();
		expect(typeof websocket.setFilters).toBe('function');
	});

	it('should have disconnect method', async () => {
		const { websocket } = await import('./websocket');

		expect(websocket.disconnect).toBeDefined();
		expect(typeof websocket.disconnect).toBe('function');
	});

	it('should update filters correctly', async () => {
		const { websocket } = await import('./websocket');

		websocket.setFilters({ severities: ['high', 'critical'] });

		const state = get(websocket);
		expect(state.filters.severities).toEqual(['high', 'critical']);
	});

	it('should update types filter correctly', async () => {
		const { websocket } = await import('./websocket');

		websocket.setFilters({ types: ['zscore', 'iqr'] });

		const state = get(websocket);
		expect(state.filters.types).toEqual(['zscore', 'iqr']);
	});

	it('should have derived stores for message types', async () => {
		const { outlierMessages, statisticsMessages, systemMessages } = await import('./websocket');

		expect(outlierMessages).toBeDefined();
		expect(statisticsMessages).toBeDefined();
		expect(systemMessages).toBeDefined();

		// Verify they're stores with subscribe method
		expect(outlierMessages.subscribe).toBeDefined();
		expect(statisticsMessages.subscribe).toBeDefined();
		expect(systemMessages.subscribe).toBeDefined();
	});
});
