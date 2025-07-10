import axios from 'axios';

// API configuration
const API_BASE_URL = '/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json'
  }
});

// Request interceptor for logging
api.interceptors.request.use(
  (config) => {
    console.log(`🚀 API Request: ${config.method?.toUpperCase()} ${config.url}`);
    return config;
  },
  (error) => {
    console.error('❌ API Request Error:', error);
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => {
    console.log(`✅ API Response: ${response.status} ${response.config.url}`);
    return response;
  },
  (error) => {
    console.error('❌ API Error:', error.response?.data || error.message);
    
    // Handle different error types
    if (error.code === 'ECONNREFUSED') {
      throw new Error('Unable to connect to API server. Please ensure the server is running on localhost:8080');
    }
    
    if (error.response?.status === 404) {
      throw new Error('API endpoint not found');
    }
    
    if (error.response?.status >= 500) {
      throw new Error('Server error occurred');
    }
    
    throw error;
  }
);

export default api;