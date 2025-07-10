// Quick test script to verify API endpoints
const API_BASE = 'http://localhost:8080/api/v1';

async function testEndpoints() {
  console.log('Testing Claude Session Manager API endpoints...\n');

  const endpoints = [
    { name: 'Health Check', path: '/health' },
    { name: 'All Sessions', path: '/sessions' },
    { name: 'Metrics Summary', path: '/metrics/summary' },
    { name: 'Usage Stats', path: '/metrics/usage' },
    { name: 'Activity Timeline', path: '/metrics/activity?limit=10' }
  ];

  for (const endpoint of endpoints) {
    try {
      console.log(`Testing ${endpoint.name}...`);
      const response = await fetch(`${API_BASE}${endpoint.path}`);
      console.log(`  Status: ${response.status} ${response.statusText}`);
      
      if (response.ok) {
        const data = await response.json();
        console.log(`  Success! Response has ${JSON.stringify(data).length} chars`);
      }
    } catch (error) {
      console.log(`  Error: ${error.message}`);
    }
    console.log('');
  }
}

testEndpoints();