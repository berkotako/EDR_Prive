import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

// Create axios instance with default config
const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor - add auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor - handle errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      // Handle specific error codes
      switch (error.response.status) {
        case 401:
          // Unauthorized - redirect to login
          localStorage.removeItem('auth_token');
          window.location.href = '/login';
          break;
        case 403:
          console.error('Forbidden:', error.response.data);
          break;
        case 404:
          console.error('Not found:', error.response.data);
          break;
        case 500:
          console.error('Server error:', error.response.data);
          break;
        default:
          console.error('API error:', error.response.data);
      }
    } else if (error.request) {
      console.error('Network error:', error.message);
    } else {
      console.error('Error:', error.message);
    }
    return Promise.reject(error);
  }
);

// API endpoints
export const endpoints = {
  // License endpoints
  licenses: {
    list: '/api/v1/licenses',
    get: (id) => `/api/v1/licenses/${id}`,
    create: '/api/v1/licenses',
    validate: '/api/v1/licenses/validate',
    trial: '/api/v1/licenses/trial',
    revoke: (id) => `/api/v1/licenses/${id}`,
    usage: (id) => `/api/v1/licenses/${id}/usage`,
  },

  // DLP endpoints
  dlp: {
    policies: '/api/v1/dlp/policies',
    policy: (id) => `/api/v1/dlp/policies/${id}`,
    fingerprints: (id) => `/api/v1/dlp/policies/${id}/fingerprints`,
    test: '/api/v1/dlp/test',
  },

  // Agent endpoints
  agents: {
    list: '/api/v1/agents',
    get: (id) => `/api/v1/agents/${id}`,
    config: (id) => `/api/v1/agents/${id}/config`,
  },

  // Telemetry endpoints
  telemetry: {
    query: '/api/v1/telemetry/query',
    event: (id) => `/api/v1/telemetry/events/${id}`,
    statistics: '/api/v1/telemetry/statistics',
  },

  // MITRE ATT&CK endpoints
  mitre: {
    tactics: '/api/v1/mitre/tactics',
    techniques: '/api/v1/mitre/techniques',
    coverage: '/api/v1/mitre/coverage',
  },

  // Alert endpoints
  alerts: {
    rules: '/api/v1/alerts/rules',
    rule: (id) => `/api/v1/alerts/rules/${id}`,
  },
};

export default api;
