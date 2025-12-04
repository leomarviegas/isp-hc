/**
 * API service for ISP Health Checker
 * Handles all communication with the backend API
 */

const API_BASE_URL = process.env.REACT_APP_API_URL || '/api/v1';

/**
 * Custom error class for API errors
 */
class ApiError extends Error {
  constructor(message, status, data) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

/**
 * Get the API key from localStorage or environment
 */
const getApiKey = () => {
  return localStorage.getItem('apiKey') || process.env.REACT_APP_API_KEY || '';
};

/**
 * Set the API key in localStorage
 */
export const setApiKey = (key) => {
  localStorage.setItem('apiKey', key);
};

/**
 * Clear the API key from localStorage
 */
export const clearApiKey = () => {
  localStorage.removeItem('apiKey');
};

/**
 * Make an API request with authentication
 */
const apiRequest = async (endpoint, options = {}) => {
  const apiKey = getApiKey();

  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };

  if (apiKey) {
    headers['Authorization'] = `Bearer ${apiKey}`;
  }

  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers,
  });

  // Handle rate limiting
  if (response.status === 429) {
    const retryAfter = response.headers.get('Retry-After') || 60;
    throw new ApiError(
      `Rate limit exceeded. Retry after ${retryAfter} seconds.`,
      429,
      { retryAfter: parseInt(retryAfter, 10) }
    );
  }

  // Handle other errors
  if (!response.ok) {
    let errorData;
    try {
      errorData = await response.json();
    } catch {
      errorData = { detail: response.statusText };
    }
    throw new ApiError(
      errorData.detail || 'An error occurred',
      response.status,
      errorData
    );
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return null;
  }

  return response.json();
};

/**
 * Fetch list of runs with optional filters
 */
export const getRuns = async (options = {}) => {
  const { limit = 20, offset = 0, target } = options;
  const params = new URLSearchParams({ limit, offset });

  if (target) {
    params.append('target', target);
  }

  return apiRequest(`/runs?${params.toString()}`);
};

/**
 * Fetch a single run by ID
 */
export const getRun = async (runId) => {
  return apiRequest(`/runs/${runId}`);
};

/**
 * Fetch raw probe output for a run
 */
export const getRunRaw = async (runId) => {
  return apiRequest(`/runs/${runId}/raw`);
};

/**
 * Fetch probes for a run
 */
export const getRunProbes = async (runId) => {
  return apiRequest(`/runs/${runId}/probes`);
};

/**
 * Submit a new run (start health check)
 */
export const createRun = async (runData) => {
  return apiRequest('/runs', {
    method: 'POST',
    body: JSON.stringify(runData),
  });
};

/**
 * Delete a run by ID
 */
export const deleteRun = async (runId) => {
  return apiRequest(`/runs/${runId}`, {
    method: 'DELETE',
  });
};

/**
 * Check API health
 */
export const checkHealth = async () => {
  const response = await fetch('/health');
  return response.json();
};

/**
 * Check API readiness
 */
export const checkReady = async () => {
  const response = await fetch('/ready');
  return response.json();
};

export { ApiError };
