// Use a configurable base URL. In development the Vite dev server proxies
// /api and /ws to the backend, so a relative path works out of the box.
// For a custom backend origin, set VITE_API_BASE_URL (e.g. http://localhost:8080).
const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || '/api/v1') as string;

interface ApiResponse<T> {
  data?: T;
  error?: string;
  total?: number;
}

class ApiClient {
  private token: string | null = null;

  constructor() {
    // Load token from localStorage
    this.token = localStorage.getItem('auth_token');
  }

  setToken(token: string) {
    this.token = token;
    localStorage.setItem('auth_token', token);
  }

  clearToken() {
    this.token = null;
    localStorage.removeItem('auth_token');
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    try {
      const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        ...options,
        headers,
      });

      if (response.status === 401) {
        // Try to read the backend error message (e.g. "invalid credentials").
        let message = 'Unauthorized';
        try {
          const errorBody = await response.json();
          if (errorBody && typeof errorBody.error === 'string' && errorBody.error) {
            message = errorBody.error;
          }
        } catch {
          // ignore body parse errors
        }

        // Only redirect to login for protected endpoints. Login itself (and other
        // auth endpoints) may legitimately return 401 for bad credentials.
        if (endpoint !== '/auth/login' && endpoint !== '/auth/register') {
          this.clearToken();
          if (window.location.pathname !== '/login') {
            window.location.href = '/login';
          }
        }
        return { error: message };
      }

      if (!response.ok) {
        const error = await response.json();
        return { error: error.error || 'Request failed' };
      }

      const body = await response.json();
      // Unwrap paginated responses ({ data: [...], total }) so callers receive the array.
      // For non-paginated responses (e.g. auth: { token, user }) return the whole body.
      const data =
        body && typeof body === 'object' && 'data' in body
          ? (body as { data: T }).data
          : (body as T);
      return { data };
    } catch (error) {
      return { error: 'Network error' };
    }
  }

  // Auth APIs
  async register(username: string, password: string, email?: string) {
    return this.request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, password, email }),
    });
  }

  async login(username: string, password: string) {
    const result = await this.request<{ token: string; user: any }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
    
    if (result.data?.token) {
      this.setToken(result.data.token);
    }
    
    return result;
  }

  async getCurrentUser() {
    return this.request('/auth/me');
  }

  // Platform APIs
  async getPlatforms(limit = 20, offset = 0) {
    return this.request<any[]>(`/platforms?limit=${limit}&offset=${offset}`);
  }

  async getPlatform(id: string) {
    return this.request<any>(`/platforms/${id}`);
  }

  async createPlatform(data: any) {
    return this.request('/platforms', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updatePlatform(id: string, data: any) {
    return this.request(`/platforms/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deletePlatform(id: string) {
    return this.request(`/platforms/${id}`, { method: 'DELETE' });
  }

  // Adapter APIs
  async getAdapters(limit = 20, offset = 0) {
    return this.request<any[]>(`/adapters?limit=${limit}&offset=${offset}`);
  }

  async createAdapter(data: any) {
    return this.request('/adapters', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateAdapter(id: string, data: any) {
    return this.request(`/adapters/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteAdapter(id: string) {
    return this.request(`/adapters/${id}`, { method: 'DELETE' });
  }

  async startAdapter(id: string) {
    return this.request(`/adapters/${id}/start`, { method: 'POST' });
  }

  async stopAdapter(id: string) {
    return this.request(`/adapters/${id}/stop`, { method: 'POST' });
  }

  // Instance APIs
  async getInstances(limit = 20, offset = 0) {
    return this.request<any[]>(`/instances?limit=${limit}&offset=${offset}`);
  }

  async createInstance(data: any) {
    return this.request('/instances', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateInstance(id: string, data: any) {
    return this.request(`/instances/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteInstance(id: string) {
    return this.request(`/instances/${id}`, { method: 'DELETE' });
  }

  // Message APIs
  async getMessages(limit = 20, offset = 0) {
    return this.request<any[]>(`/messages?limit=${limit}&offset=${offset}`);
  }

  async ackMessage(id: string) {
    return this.request(`/messages/${id}/ack`, { method: 'POST' });
  }

  // Command APIs
  async getCommands(limit = 20, offset = 0) {
    return this.request<any[]>(`/commands?limit=${limit}&offset=${offset}`);
  }

  async createCommand(data: any) {
    return this.request('/commands', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Route APIs
  async getRoutes(limit = 20, offset = 0) {
    return this.request<any[]>(`/routes?limit=${limit}&offset=${offset}`);
  }

  async createRoute(data: any) {
    return this.request('/routes', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateRoute(id: string, data: any) {
    return this.request(`/routes/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteRoute(id: string) {
    return this.request(`/routes/${id}`, { method: 'DELETE' });
  }

  // Documentation APIs
  async getDocs() {
    return this.request<any[]>('/docs');
  }

  async getDoc(key: string) {
    return this.request<any>(`/docs/${key}`);
  }

  // Health check
  async healthCheck() {
    try {
      const base = (import.meta.env.VITE_API_BASE_URL as string) || '';
      const response = await fetch(`${base}/health`);
      return await response.json();
    } catch {
      return { status: 'error', message: 'Backend not available' };
    }
  }
}

export const apiClient = new ApiClient();
export default apiClient;
