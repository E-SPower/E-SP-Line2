// Use a configurable base URL. In development the Vite dev server proxies
// /api and /ws to the backend, so a relative path works out of the box.
// For a custom backend origin, set VITE_API_BASE_URL (e.g. http://localhost:8080).
const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || '/api/v1') as string;

// requestTimeoutMs bounds every API request. Without a timeout, a stalled
// backend leaves fetches pending forever, accumulating in-flight requests and
// making the UI appear frozen.
const requestTimeoutMs = 30000;

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

    // Abort the request if the backend does not respond in time.
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), requestTimeoutMs);

    try {
      const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        ...options,
        headers,
        signal: controller.signal,
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
      if (error instanceof DOMException && error.name === 'AbortError') {
        return { error: 'Request timeout' };
      }
      return { error: 'Network error' };
    } finally {
      clearTimeout(timeout);
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

  // Adapter catalog (scanned from adapters/*/adapter.yaml)
  async getAdapterCatalog() {
    return this.request<any[]>(`/adapters/catalog`);
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

  async startInstance(id: string) {
    return this.request(`/instances/${id}/start`, { method: 'POST' });
  }

  async stopInstance(id: string) {
    return this.request(`/instances/${id}/stop`, { method: 'POST' });
  }

  async getInstanceLogs(id: string, opts: { lines?: number | 'all'; level?: string; keyword?: string; from?: string; to?: string } = {}) {
    const params = new URLSearchParams()
    // Cap the default log read so a huge log file never stalls the request.
    params.set('lines', opts.lines === undefined ? '2000' : String(opts.lines))
    if (opts.level) params.set('level', opts.level)
    if (opts.keyword) params.set('keyword', opts.keyword)
    if (opts.from) params.set('from', opts.from)
    if (opts.to) params.set('to', opts.to)
    return this.request<any>(`/instances/${id}/logs?${params.toString()}`);
  }

  // Per-hour log level distribution (heatmap)
  async getInstanceLogHeatmap(id: string, level?: string) {
    const params = new URLSearchParams()
    if (level) params.set('level', level)
    return this.request<any[]>(`/instances/${id}/logs/heatmap?${params.toString()}`);
  }

  // Clear (truncate) an instance's log file
  async clearInstanceLogs(id: string) {
    return this.request<any>(`/instances/${id}/logs`, { method: 'DELETE' });
  }

  // ---- User management (admin) ----
  async listUsers() {
    return this.request<any[]>('/users');
  }
  async createUser(username: string, password: string, role: string) {
    return this.request<any>('/users', { method: 'POST', body: JSON.stringify({ username, password, role }) });
  }
  async updateUserRole(id: string, role: string) {
    return this.request<any>(`/users/${id}/role`, { method: 'PUT', body: JSON.stringify({ role }) });
  }
  async updateUserStatus(id: string, status: string) {
    return this.request<any>(`/users/${id}/status`, { method: 'PUT', body: JSON.stringify({ status }) });
  }
  async deleteUser(id: string) {
    return this.request<any>(`/users/${id}`, { method: 'DELETE' });
  }

  // ---- Registration toggle (admin) ----
  async getRegistrationStatus() {
    return this.request<boolean>('/settings/registration');
  }
  async setRegistrationStatus(enabled: boolean) {
    return this.request<any>('/settings/registration', { method: 'PUT', body: JSON.stringify({ enabled }) });
  }

  // Database / storage usage stats
  async getStorageStats() {
    return this.request<any>(`/stats/storage`);
  }

  // Dependency installation (initialization) status for an instance
  async getInstanceInitStatus(id: string) {
    return this.request<any>(`/instances/${id}/init`);
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

  // Form option APIs (from YAML registry)
  async getFormOptions() {
    return this.request<any[]>(`/options`);
  }

  async getFormOptionGroup(key: string) {
    return this.request<any>(`/options/${key}`);
  }

  // ---- Adapter gateway (接入器) entity & connection APIs ----
  async getGatewayAdapters(limit = 20, offset = 0) {
    return this.request<any[]>(`/adapter-gateways?limit=${limit}&offset=${offset}`);
  }

  async createGatewayAdapter(data: any) {
    return this.request<any>('/adapter-gateways', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async getGatewayAdapter(id: string) {
    return this.request<any>(`/adapter-gateways/${id}`);
  }

  async updateGatewayAdapter(id: string, data: any) {
    return this.request<any>(`/adapter-gateways/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteGatewayAdapter(id: string) {
    return this.request<any>(`/adapter-gateways/${id}`, { method: 'DELETE' });
  }

  async getAdapterConnections(limit = 20, offset = 0) {
    return this.request<any[]>(`/adapter-connections?limit=${limit}&offset=${offset}`);
  }

  async getAdapterConnectionsByAdapter(id: string, limit = 20, offset = 0) {
    return this.request<any[]>(`/adapter-gateways/${id}/connections?limit=${limit}&offset=${offset}`);
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
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 5000);
      const response = await fetch(`${base}/health`, { signal: controller.signal });
      clearTimeout(timeout);
      return await response.json();
    } catch {
      return { status: 'error', message: 'Backend not available' };
    }
  }
}

export const apiClient = new ApiClient();
export default apiClient;
