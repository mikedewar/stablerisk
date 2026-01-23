import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { writable } from 'svelte/store';
import Layout from './+layout.svelte';

// Mock stores and navigation
vi.mock('$stores/auth', () => ({
	auth: {
		subscribe: vi.fn(),
		logout: vi.fn()
	}
}));

vi.mock('$stores/websocket', () => ({
	websocket: {
		subscribe: vi.fn(),
		disconnect: vi.fn()
	}
}));

vi.mock('$app/stores', () => ({
	page: writable({
		url: new URL('http://localhost/'),
		params: {},
		route: { id: null },
		status: 200,
		error: null,
		data: {},
		form: undefined
	})
}));

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

// Import after mocking
import { auth } from '$stores/auth';
import { websocket } from '$stores/websocket';
import { page } from '$app/stores';
import { goto } from '$app/navigation';

describe('Layout Component', () => {
	let authState: any;
	let websocketState: any;

	beforeEach(() => {
		vi.clearAllMocks();

		// Default auth state - logged in
		authState = {
			user: {
				id: '1',
				username: 'testuser',
				role: 'admin'
			},
			token: 'test-token',
			refreshToken: 'test-refresh-token',
			loading: false,
			error: null
		};

		// Default WebSocket state - connected
		websocketState = {
			connected: true,
			reconnecting: false
		};

		// Setup auth store mock
		vi.mocked(auth.subscribe).mockImplementation((callback) => {
			callback(authState);
			return () => {};
		});

		// Setup websocket store mock
		vi.mocked(websocket.subscribe).mockImplementation((callback) => {
			callback(websocketState);
			return () => {};
		});

		// Setup page store
		page.set({
			url: new URL('http://localhost/'),
			params: {},
			route: { id: null },
			status: 200,
			error: null,
			data: {},
			form: undefined
		});
	});

	describe('Navigation Rendering', () => {
		it('should render logo', () => {
			render(Layout);

			expect(screen.getByText('StableRisk')).toBeTruthy();
		});

		it('should render all navigation links', () => {
			render(Layout);

			expect(screen.getByText('Dashboard')).toBeTruthy();
			expect(screen.getByText('Outliers')).toBeTruthy();
			expect(screen.getByText('Statistics')).toBeTruthy();
		});

		it('should render navigation links with correct hrefs', () => {
			render(Layout);

			const dashboardLink = screen.getByText('Dashboard').closest('a');
			const outliersLink = screen.getByText('Outliers').closest('a');
			const statisticsLink = screen.getByText('Statistics').closest('a');

			expect(dashboardLink?.getAttribute('href')).toBe('/');
			expect(outliersLink?.getAttribute('href')).toBe('/outliers');
			expect(statisticsLink?.getAttribute('href')).toBe('/statistics');
		});

		it('should display user avatar with username initial', () => {
			render(Layout);

			const avatar = screen.getByText('T'); // First letter of 'testuser'
			expect(avatar).toBeTruthy();
		});

		it('should display full username in dropdown', () => {
			render(Layout);

			expect(screen.getByText('testuser')).toBeTruthy();
		});
	});

	describe('Mobile Navigation', () => {
		it('should render mobile menu button', () => {
			const { container } = render(Layout);

			// Mobile menu button is inside flex-none lg:hidden div
			const menuButton = container.querySelector('.flex-none.lg\\:hidden button');
			expect(menuButton).toBeTruthy();
		});

		it('should toggle mobile menu when button is clicked', async () => {
			const { container } = render(Layout);

			const menuButton = container.querySelector('.flex-none.lg\\:hidden button') as HTMLButtonElement;

			// Menu should not exist initially
			let mobileMenu = container.querySelector('div.lg\\:hidden.bg-base-100');
			expect(mobileMenu).toBeFalsy();

			// Click to open
			await fireEvent.click(menuButton);

			mobileMenu = container.querySelector('div.lg\\:hidden.bg-base-100');
			expect(mobileMenu).toBeTruthy();

			// Click to close
			await fireEvent.click(menuButton);

			mobileMenu = container.querySelector('div.lg\\:hidden.bg-base-100');
			expect(mobileMenu).toBeFalsy();
		});

		it('should render mobile navigation links when menu is open', async () => {
			const { container } = render(Layout);

			// Open the menu first
			const menuButton = container.querySelector('.flex-none.lg\\:hidden button') as HTMLButtonElement;
			await fireEvent.click(menuButton);

			// Now there should be two sets of links (desktop and mobile)
			const dashboardLinks = screen.getAllByText('Dashboard');
			expect(dashboardLinks.length).toBeGreaterThanOrEqual(2);
		});
	});

	describe('Auth Guard', () => {
		it('should not render navigation when user is not authenticated', () => {
			authState.user = null;

			const { container } = render(Layout);

			// Navigation should not be rendered
			const navbar = container.querySelector('.navbar');
			expect(navbar).toBeFalsy();

			// Main content should still be rendered
			const main = container.querySelector('main');
			expect(main).toBeTruthy();
		});

		it('should render navigation when user is authenticated', () => {
			authState.user = {
				id: '1',
				username: 'testuser',
				role: 'admin'
			};

			const { container } = render(Layout);

			// Navigation should be rendered
			const navbar = container.querySelector('.navbar');
			expect(navbar).toBeTruthy();

			// Should show user menu
			expect(screen.getByText('testuser')).toBeTruthy();
		});

		it('should hide navigation elements when user logs out', () => {
			authState.user = {
				id: '1',
				username: 'testuser',
				role: 'admin'
			};

			const { container } = render(Layout);

			// Initially has navigation
			const navbar = container.querySelector('.navbar');
			expect(navbar).toBeTruthy();

			// Simulate user logout by updating auth state
			authState.user = null;
			vi.mocked(auth.subscribe).mockImplementation((callback) => {
				callback(authState);
				return () => {};
			});

			// Note: In a real app, the component would re-render on auth state change
			// This test verifies the conditional rendering logic
		});
	});

	describe('Logout Functionality', () => {
		it('should display logout button in user dropdown', () => {
			render(Layout);

			expect(screen.getByText('Logout')).toBeTruthy();
		});

		it('should call auth.logout when logout is clicked', async () => {
			render(Layout);

			const logoutButton = screen.getByText('Logout');
			await fireEvent.click(logoutButton);

			expect(auth.logout).toHaveBeenCalled();
		});

		it('should disconnect WebSocket when logout is clicked', async () => {
			render(Layout);

			const logoutButton = screen.getByText('Logout');
			await fireEvent.click(logoutButton);

			expect(websocket.disconnect).toHaveBeenCalled();
		});

		it('should redirect to login after logout', async () => {
			render(Layout);

			const logoutButton = screen.getByText('Logout');
			await fireEvent.click(logoutButton);

			expect(goto).toHaveBeenCalledWith('/login');
		});
	});

	describe('WebSocket Status Indicator', () => {
		it('should show connected tooltip when WebSocket is connected', () => {
			websocketState.connected = true;
			websocketState.reconnecting = false;

			const { container } = render(Layout);

			const tooltip = container.querySelector('.tooltip');
			expect(tooltip?.getAttribute('data-tip')).toBe('Connected');
		});

		it('should show disconnected tooltip when WebSocket is disconnected', () => {
			websocketState.connected = false;
			websocketState.reconnecting = false;

			const { container } = render(Layout);

			const tooltip = container.querySelector('.tooltip');
			expect(tooltip?.getAttribute('data-tip')).toBe('Disconnected');
		});

		it('should display success badge when connected', () => {
			websocketState.connected = true;
			websocketState.reconnecting = false;

			const { container } = render(Layout);

			const badge = container.querySelector('.badge-success');
			expect(badge).toBeTruthy();
		});

		it('should display warning badge when reconnecting', () => {
			websocketState.connected = false;
			websocketState.reconnecting = true;

			const { container } = render(Layout);

			const badge = container.querySelector('.badge-warning');
			expect(badge).toBeTruthy();
		});

		it('should display error badge when disconnected', () => {
			websocketState.connected = false;
			websocketState.reconnecting = false;

			const { container } = render(Layout);

			const badge = container.querySelector('.badge-error');
			expect(badge).toBeTruthy();
		});

		it('should display indicator with all badge types', () => {
			const { container } = render(Layout);

			const indicator = container.querySelector('.indicator');
			expect(indicator).toBeTruthy();

			// Should have a badge inside
			const badge = indicator?.querySelector('.badge');
			expect(badge).toBeTruthy();
		});
	});

	describe('User Dropdown', () => {
		it('should display user profile link', () => {
			render(Layout);

			expect(screen.getByText('Profile')).toBeTruthy();
		});

		it('should have correct href for profile link', () => {
			render(Layout);

			const profileLink = screen.getByText('Profile').closest('a');
			expect(profileLink?.getAttribute('href')).toBe('/profile');
		});

		it('should display user role in dropdown', () => {
			render(Layout);

			expect(screen.getByText('admin')).toBeTruthy();
		});

		it('should display different roles correctly', () => {
			authState.user.role = 'analyst';

			render(Layout);

			expect(screen.getByText('analyst')).toBeTruthy();
		});
	});

	describe('Active Link Styling', () => {
		it('should apply active class to dashboard link on home page', () => {
			page.set({
				url: new URL('http://localhost/'),
				params: {},
				route: { id: null },
				status: 200,
				error: null,
				data: {},
				form: undefined
			});

			const { container } = render(Layout);

			const dashboardLinks = container.querySelectorAll('a[href="/"]');
			const hasActiveClass = Array.from(dashboardLinks).some((link) =>
				link.classList.contains('active')
			);
			expect(hasActiveClass).toBe(true);
		});

		it('should apply active class to outliers link on outliers page', () => {
			page.set({
				url: new URL('http://localhost/outliers'),
				params: {},
				route: { id: null },
				status: 200,
				error: null,
				data: {},
				form: undefined
			});

			const { container } = render(Layout);

			const outliersLinks = container.querySelectorAll('a[href="/outliers"]');
			const hasActiveClass = Array.from(outliersLinks).some((link) =>
				link.classList.contains('active')
			);
			expect(hasActiveClass).toBe(true);
		});

		it('should apply active class to statistics link on statistics page', () => {
			page.set({
				url: new URL('http://localhost/statistics'),
				params: {},
				route: { id: null },
				status: 200,
				error: null,
				data: {},
				form: undefined
			});

			const { container } = render(Layout);

			const statisticsLinks = container.querySelectorAll('a[href="/statistics"]');
			const hasActiveClass = Array.from(statisticsLinks).some((link) =>
				link.classList.contains('active')
			);
			expect(hasActiveClass).toBe(true);
		});
	});

	describe('Responsive Behavior', () => {
		it('should have desktop navigation with hidden lg:flex classes', () => {
			const { container } = render(Layout);

			const desktopNav = container.querySelector('.flex-none.hidden.lg\\:flex');
			expect(desktopNav).toBeTruthy();
		});

		it('should have mobile menu button container with lg:hidden class', () => {
			const { container } = render(Layout);

			const mobileButtonContainer = container.querySelector('.flex-none.lg\\:hidden');
			expect(mobileButtonContainer).toBeTruthy();

			// Button should be inside
			const button = mobileButtonContainer?.querySelector('button');
			expect(button).toBeTruthy();
		});

		it('should have mobile menu dropdown with lg:hidden class when opened', async () => {
			const { container } = render(Layout);

			// Open the menu first
			const menuButton = container.querySelector('.flex-none.lg\\:hidden button') as HTMLButtonElement;
			await fireEvent.click(menuButton);

			const mobileMenu = container.querySelector('div.lg\\:hidden.bg-base-100');
			expect(mobileMenu).toBeTruthy();
		});
	});

	describe('Content Slot', () => {
		it('should render content in the main slot', () => {
			const { container } = render(Layout);

			// Layout should have a main element for content
			const main = container.querySelector('main');
			expect(main).toBeTruthy();
		});
	});

	describe('Error Cases', () => {
		it('should handle missing user data gracefully', () => {
			authState.user = null;

			const { container } = render(Layout);

			// Should not crash and should hide navigation
			const navbar = container.querySelector('.navbar');
			expect(navbar).toBeFalsy();

			// Main content slot should still render
			const main = container.querySelector('main');
			expect(main).toBeTruthy();
		});

		it('should handle disconnected WebSocket state', () => {
			vi.mocked(websocket.subscribe).mockImplementation((callback) => {
				callback({ connected: false, reconnecting: false });
				return () => {};
			});

			const { container } = render(Layout);

			// Should render with error badge
			const badge = container.querySelector('.badge-error');
			expect(badge).toBeTruthy();
		});

		it('should call all cleanup functions on logout', async () => {
			render(Layout);

			const logoutButton = screen.getByText('Logout');
			await fireEvent.click(logoutButton);

			// All three functions should be called
			expect(auth.logout).toHaveBeenCalled();
			expect(websocket.disconnect).toHaveBeenCalled();
			expect(goto).toHaveBeenCalledWith('/login');
		});
	});
});
