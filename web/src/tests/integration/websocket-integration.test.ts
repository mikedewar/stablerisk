import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { websocket, outlierMessages } from '$stores/websocket';

/**
 * Integration Test: WebSocket Integration
 *
 * Tests the WebSocket store integration including:
 * - Connection state management
 * - Message filtering and routing
 * - Derived store updates
 * - Filter configuration
 */

describe('Integration: WebSocket Store', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		// Disconnect any existing connections
		websocket.disconnect();
	});

	describe('Store Structure Integration', () => {
		it('should have correct initial state', () => {
			const state = get(websocket);

			expect(state).toHaveProperty('connected');
			expect(state).toHaveProperty('reconnecting');
			expect(state).toHaveProperty('filters');
			expect(state.connected).toBe(false);
			expect(state.reconnecting).toBe(false);
		});

		it('should have derived message stores', () => {
			// Verify derived stores exist
			expect(outlierMessages).toBeDefined();

			// Verify they can be subscribed to
			const unsubscribe = outlierMessages.subscribe(() => {});
			expect(typeof unsubscribe).toBe('function');
			unsubscribe();
		});
	});

	describe('Filter Integration', () => {
		it('should update filters via setFilters method', () => {
			// Set filters
			websocket.setFilters({
				types: ['zscore', 'iqr'],
				severities: ['high', 'critical']
			});

			// Verify filters were updated
			const state = get(websocket);
			expect(state.filters.types).toEqual(['zscore', 'iqr']);
			expect(state.filters.severities).toEqual(['high', 'critical']);
		});

		it('should preserve other state when updating filters', () => {
			// Get initial state
			const initialState = get(websocket);

			// Update filters
			websocket.setFilters({
				types: ['pattern_circulation']
			});

			// Verify connected state wasn't affected
			const newState = get(websocket);
			expect(newState.connected).toBe(initialState.connected);
			expect(newState.reconnecting).toBe(initialState.reconnecting);
		});

		it('should handle partial filter updates', () => {
			// Set initial filters
			websocket.setFilters({
				types: ['zscore'],
				severities: ['high']
			});

			// Update only types
			websocket.setFilters({
				types: ['iqr', 'zscore']
			});

			// Verify types updated but severities remained
			const state = get(websocket);
			expect(state.filters.types).toEqual(['iqr', 'zscore']);
			expect(state.filters.severities).toEqual(['high']);
		});
	});

	describe('Disconnect Integration', () => {
		it('should have disconnect method', () => {
			expect(typeof websocket.disconnect).toBe('function');
		});

		it('should be safe to call disconnect multiple times', () => {
			// Should not throw
			expect(() => {
				websocket.disconnect();
				websocket.disconnect();
				websocket.disconnect();
			}).not.toThrow();
		});
	});

	describe('Store Subscription Integration', () => {
		it('should notify subscribers of state changes', () => {
			const states: any[] = [];

			const unsubscribe = websocket.subscribe((state) => {
				states.push({ ...state });
			});

			// Initial state
			expect(states.length).toBeGreaterThan(0);

			// Update filters
			websocket.setFilters({ types: ['zscore'] });

			// Should have captured the update
			expect(states.length).toBeGreaterThan(1);

			unsubscribe();
		});

		it('should allow multiple subscribers', () => {
			const states1: any[] = [];
			const states2: any[] = [];

			const unsubscribe1 = websocket.subscribe((state) => states1.push(state));
			const unsubscribe2 = websocket.subscribe((state) => states2.push(state));

			websocket.setFilters({ types: ['iqr'] });

			// Both subscribers should receive updates
			expect(states1.length).toBeGreaterThan(0);
			expect(states2.length).toBeGreaterThan(0);

			unsubscribe1();
			unsubscribe2();
		});
	});
});
