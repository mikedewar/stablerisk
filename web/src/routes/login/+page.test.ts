import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';
import LoginPage from './+page.svelte';

// Mock the auth store
vi.mock('$stores/auth', () => ({
	auth: {
		subscribe: vi.fn(),
		login: vi.fn()
	}
}));

// Import after mocking
import { auth } from '$stores/auth';
import { goto } from '$app/navigation';

describe('Login Page', () => {
	let authState: any;

	beforeEach(() => {
		vi.clearAllMocks();

		// Default auth state
		authState = {
			user: null,
			token: null,
			loading: false,
			error: null
		};

		// Setup auth store mock
		vi.mocked(auth.subscribe).mockImplementation((callback) => {
			callback(authState);
			return () => {};
		});

		vi.mocked(auth.login).mockResolvedValue(true);
	});

	describe('Rendering', () => {
		it('should render login form', () => {
			render(LoginPage);

			expect(screen.getByText('StableRisk')).toBeTruthy();
			expect(screen.getByText('Sign In')).toBeTruthy();
			expect(screen.getByLabelText('Username')).toBeTruthy();
			expect(screen.getByLabelText('Password')).toBeTruthy();
			expect(screen.getByRole('button', { name: /sign in/i })).toBeTruthy();
		});

		it('should display page title', () => {
			render(LoginPage);

			expect(screen.getByText('StableRisk')).toBeTruthy();
		});

		it('should display description', () => {
			render(LoginPage);

			const description = screen.getByText(/Real-time USDT transaction monitoring/);
			expect(description).toBeTruthy();
		});

		it('should render username input field', () => {
			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			expect(usernameInput).toBeTruthy();
			expect(usernameInput.type).toBe('text');
			expect(usernameInput.placeholder).toBe('username');
		});

		it('should render password input field', () => {
			render(LoginPage);

			const passwordInput = screen.getByLabelText('Password') as HTMLInputElement;
			expect(passwordInput).toBeTruthy();
			expect(passwordInput.type).toBe('password');
			expect(passwordInput.placeholder).toBe('password');
		});

		it('should display default credentials', () => {
			render(LoginPage);

			expect(screen.getByText('Default Credentials')).toBeTruthy();
			expect(screen.getByText(/admin \/ changeme123/)).toBeTruthy();
			expect(screen.getByText(/analyst \/ changeme123/)).toBeTruthy();
			expect(screen.getByText(/viewer \/ changeme123/)).toBeTruthy();
		});
	});

	describe('Form Validation', () => {
		it('should disable submit button when username is empty', () => {
			render(LoginPage);

			const submitButton = screen.getByRole('button', { name: /sign in/i }) as HTMLButtonElement;
			expect(submitButton.disabled).toBe(true);
		});

		it('should disable submit button when password is empty', async () => {
			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			await fireEvent.input(usernameInput, { target: { value: 'testuser' } });

			const submitButton = screen.getByRole('button', { name: /sign in/i }) as HTMLButtonElement;
			expect(submitButton.disabled).toBe(true);
		});

		it('should enable submit button when both fields are filled', async () => {
			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			const passwordInput = screen.getByLabelText('Password') as HTMLInputElement;

			await fireEvent.input(usernameInput, { target: { value: 'testuser' } });
			await fireEvent.input(passwordInput, { target: { value: 'password123' } });

			await waitFor(() => {
				const submitButton = screen.getByRole('button', { name: /sign in/i }) as HTMLButtonElement;
				expect(submitButton.disabled).toBe(false);
			});
		});

		it('should have required attribute on username field', () => {
			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			expect(usernameInput.required).toBe(true);
		});

		it('should have required attribute on password field', () => {
			render(LoginPage);

			const passwordInput = screen.getByLabelText('Password') as HTMLInputElement;
			expect(passwordInput.required).toBe(true);
		});
	});

	describe('Login Submission', () => {
		it('should call auth.login when form is submitted', async () => {
			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			const passwordInput = screen.getByLabelText('Password') as HTMLInputElement;
			const submitButton = screen.getByRole('button', { name: /sign in/i });

			await fireEvent.input(usernameInput, { target: { value: 'testuser' } });
			await fireEvent.input(passwordInput, { target: { value: 'password123' } });
			await fireEvent.click(submitButton);

			expect(auth.login).toHaveBeenCalledWith('testuser', 'password123');
		});

		it('should redirect to home on successful login', async () => {
			vi.mocked(auth.login).mockResolvedValue(true);
			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			const passwordInput = screen.getByLabelText('Password') as HTMLInputElement;
			const submitButton = screen.getByRole('button', { name: /sign in/i });

			await fireEvent.input(usernameInput, { target: { value: 'testuser' } });
			await fireEvent.input(passwordInput, { target: { value: 'password123' } });
			await fireEvent.click(submitButton);

			await waitFor(() => {
				expect(goto).toHaveBeenCalledWith('/');
			});
		});

		it('should not redirect on failed login', async () => {
			vi.mocked(auth.login).mockResolvedValue(false);
			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			const passwordInput = screen.getByLabelText('Password') as HTMLInputElement;
			const submitButton = screen.getByRole('button', { name: /sign in/i });

			await fireEvent.input(usernameInput, { target: { value: 'wronguser' } });
			await fireEvent.input(passwordInput, { target: { value: 'wrongpass' } });
			await fireEvent.click(submitButton);

			await waitFor(() => {
				expect(auth.login).toHaveBeenCalled();
			});

			expect(goto).not.toHaveBeenCalled();
		});

		it('should not submit if username is empty', async () => {
			render(LoginPage);

			const passwordInput = screen.getByLabelText('Password') as HTMLInputElement;
			const submitButton = screen.getByRole('button', { name: /sign in/i });

			await fireEvent.input(passwordInput, { target: { value: 'password123' } });

			// Button should be disabled
			expect((submitButton as HTMLButtonElement).disabled).toBe(true);
		});

		it('should not submit if password is empty', async () => {
			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			const submitButton = screen.getByRole('button', { name: /sign in/i });

			await fireEvent.input(usernameInput, { target: { value: 'testuser' } });

			// Button should be disabled
			expect((submitButton as HTMLButtonElement).disabled).toBe(true);
		});
	});

	describe('Keyboard Navigation', () => {
		it('should submit form when Enter is pressed in username field', async () => {
			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			const passwordInput = screen.getByLabelText('Password') as HTMLInputElement;

			await fireEvent.input(usernameInput, { target: { value: 'testuser' } });
			await fireEvent.input(passwordInput, { target: { value: 'password123' } });
			await fireEvent.keyPress(usernameInput, { key: 'Enter', code: 'Enter' });

			expect(auth.login).toHaveBeenCalledWith('testuser', 'password123');
		});

		it('should submit form when Enter is pressed in password field', async () => {
			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			const passwordInput = screen.getByLabelText('Password') as HTMLInputElement;

			await fireEvent.input(usernameInput, { target: { value: 'testuser' } });
			await fireEvent.input(passwordInput, { target: { value: 'password123' } });
			await fireEvent.keyPress(passwordInput, { key: 'Enter', code: 'Enter' });

			expect(auth.login).toHaveBeenCalledWith('testuser', 'password123');
		});
	});

	describe('Error Display', () => {
		it('should display error message when login fails', () => {
			authState.error = 'Invalid credentials';

			render(LoginPage);

			expect(screen.getByText('Invalid credentials')).toBeTruthy();
		});

		it('should not display error initially', () => {
			render(LoginPage);

			const errorElement = screen.queryByRole('alert');
			expect(errorElement).toBeFalsy();
		});

		it('should show error alert with proper styling', () => {
			authState.error = 'Invalid credentials';

			const { container } = render(LoginPage);

			const alert = container.querySelector('.alert-error');
			expect(alert).toBeTruthy();
			expect(alert?.textContent).toContain('Invalid credentials');
		});
	});

	describe('Loading State', () => {
		it('should show loading spinner when submitting', async () => {
			let resolveLogin: any;
			const loginPromise = new Promise((resolve) => {
				resolveLogin = resolve;
			});
			vi.mocked(auth.login).mockReturnValue(loginPromise);

			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			const passwordInput = screen.getByLabelText('Password') as HTMLInputElement;
			const submitButton = screen.getByRole('button', { name: /sign in/i });

			await fireEvent.input(usernameInput, { target: { value: 'testuser' } });
			await fireEvent.input(passwordInput, { target: { value: 'password123' } });
			await fireEvent.click(submitButton);

			await waitFor(() => {
				expect(screen.getByText('Signing in...')).toBeTruthy();
			});

			resolveLogin(true);
		});

		it('should disable inputs during loading', async () => {
			let resolveLogin: any;
			const loginPromise = new Promise((resolve) => {
				resolveLogin = resolve;
			});
			vi.mocked(auth.login).mockReturnValue(loginPromise);

			render(LoginPage);

			const usernameInput = screen.getByLabelText('Username') as HTMLInputElement;
			const passwordInput = screen.getByLabelText('Password') as HTMLInputElement;
			const submitButton = screen.getByRole('button', { name: /sign in/i });

			await fireEvent.input(usernameInput, { target: { value: 'testuser' } });
			await fireEvent.input(passwordInput, { target: { value: 'password123' } });
			await fireEvent.click(submitButton);

			await waitFor(() => {
				expect(usernameInput.disabled).toBe(true);
				expect(passwordInput.disabled).toBe(true);
				expect(submitButton.disabled).toBe(true);
			});

			resolveLogin(true);
		});
	});

	describe('Auto-redirect', () => {
		it('should redirect to home if user is already logged in', () => {
			authState.user = { id: '1', username: 'testuser', role: 'admin' };

			render(LoginPage);

			expect(goto).toHaveBeenCalledWith('/');
		});

		it('should not redirect if user is not logged in', () => {
			authState.user = null;

			render(LoginPage);

			expect(goto).not.toHaveBeenCalled();
		});
	});
});
