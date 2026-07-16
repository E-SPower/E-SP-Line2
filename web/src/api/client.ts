const API_BASE_URL = 'http://localhost:8080/api/v1';

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

      if (!response.ok) {
        const error = await response.json();
        return { error: error.error || 'Request failed' };
      }

      const data = await response.json();
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

  // Health check
  async healthCheck() {
    try {
      const response = await fetch('http://localhost:8080/health');
      return await response.json();
    } catch {
      return { status: 'error', message: 'Backend not available' };
    }
  }
}

export const apiClient = new ApiClient();
export default apiClient;
