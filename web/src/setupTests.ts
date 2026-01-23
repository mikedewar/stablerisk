import '@testing-library/jest-dom';
import { vi } from 'vitest';

// Mock localStorage
const localStorageMock = {
	getItem: vi.fn(),
	setItem: vi.fn(),
	removeItem: vi.fn(),
	clear: vi.fn()
};
global.localStorage = localStorageMock as any;

// Mock WebSocket
class WebSocketMock {
	static CONNECTING = 0;
	static OPEN = 1;
	static CLOSING = 2;
	static CLOSED = 3;

	readyState = WebSocketMock.CONNECTING;
	url: string;
	onopen: ((event: Event) => void) | null = null;
	onmessage: ((event: MessageEvent) => void) | null = null;
	onerror: ((event: Event) => void) | null = null;
	onclose: ((event: CloseEvent) => void) | null = null;

	constructor(url: string) {
		this.url = url;
		setTimeout(() => {
			this.readyState = WebSocketMock.OPEN;
			if (this.onopen) {
				this.onopen(new Event('open'));
			}
		}, 0);
	}

	send(data: string) {
		// Mock send
	}

	close() {
		this.readyState = WebSocketMock.CLOSED;
		if (this.onclose) {
			this.onclose(new CloseEvent('close'));
		}
	}
}

global.WebSocket = WebSocketMock as any;

// Mock fetch
global.fetch = vi.fn();

// Reset mocks before each test
beforeEach(() => {
	vi.clearAllMocks();
	localStorageMock.getItem.mockReturnValue(null);
});
