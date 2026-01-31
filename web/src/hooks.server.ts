import type { Handle } from '@sveltejs/kit';

/**
 * SvelteKit server hook to proxy API requests to the backend
 *
 * This hook intercepts requests to /api/* and forwards them to the backend API service.
 * - In development: proxies to http://localhost:8080
 * - In production (Docker): proxies to API_URL environment variable (http://api:8080)
 *
 * This is necessary because:
 * 1. Vite's proxy config only works for client-side dev requests
 * 2. SSR requests need server-side proxying
 * 3. Production builds with adapter-node need explicit proxying
 */
export const handle: Handle = async ({ event, resolve }) => {
	const { request } = event;
	const url = new URL(request.url);

	// Check if this is an API request
	if (url.pathname.startsWith('/api/')) {
		// Check if this is a WebSocket upgrade request
		const upgradeHeader = request.headers.get('upgrade');
		const isWebSocketUpgrade = upgradeHeader?.toLowerCase() === 'websocket';

		// Skip proxying for WebSocket upgrade requests
		// WebSocket connections need to be handled directly without HTTP proxying
		if (isWebSocketUpgrade) {
			// Let the request pass through to SvelteKit's normal handling
			// This allows WebSocket connections to connect directly to the backend
			return resolve(event);
		}

		// Get API base URL from environment
		// In Docker: API_URL=http://api:8080
		// In dev: defaults to http://localhost:8080
		const apiBaseUrl = process.env.API_URL || 'http://localhost:8080';

		// Build the target URL (keep the full path including /api/v1/...)
		const targetUrl = `${apiBaseUrl}${url.pathname}${url.search}`;

		try {
			// Forward the request to the backend API
			const headers = new Headers(request.headers);

			// Remove host header to avoid conflicts
			headers.delete('host');
			headers.delete('connection');

			const apiResponse = await fetch(targetUrl, {
				method: request.method,
				headers,
				body: request.method !== 'GET' && request.method !== 'HEAD'
					? await request.text()
					: undefined,
			});

			// Create response with the API's response
			const responseHeaders = new Headers(apiResponse.headers);

			// Remove headers that might cause issues
			responseHeaders.delete('transfer-encoding');

			return new Response(await apiResponse.text(), {
				status: apiResponse.status,
				statusText: apiResponse.statusText,
				headers: responseHeaders,
			});
		} catch (error) {
			console.error(`Error proxying request to ${targetUrl}:`, error);

			return new Response(
				JSON.stringify({
					error: 'proxy_error',
					message: 'Failed to connect to API server'
				}),
				{
					status: 502,
					headers: {
						'Content-Type': 'application/json'
					}
				}
			);
		}
	}

	// For non-API requests, proceed with normal SvelteKit handling
	return resolve(event);
};
