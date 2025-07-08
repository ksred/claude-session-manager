# Postman Collection Generation

Generate a Postman collection from the Swagger documentation for easy API testing.

## Quick Start

### Method 1: Using Make (Recommended - Fixed URLs)

```bash
# Install the openapi-to-postmanv2 tool first (official Postman converter)
npm install -g openapi-to-postmanv2

# Generate Postman collection with working URLs
make postman-fixed
```

This will:
1. Generate fresh Swagger documentation
2. Convert it to a Postman collection
3. **Fix all URLs automatically** - no variable setup needed!
4. Create `./postman_collection_fixed.json` ready to import

### Method 2: Standard Collection (Requires Variable Setup)

```bash
# Generate standard collection (uses {{baseUrl}} variables)
make postman
```

This creates `./postman_collection.json` but **requires setting the baseUrl variable** in Postman (see troubleshooting below).

### Method 3: Manual Conversion

If you prefer not to install npm tools, you can use the online converter:

1. Generate Swagger docs: `make swagger`
2. Upload `./docs/swagger.json` to https://www.postman.com/api-platform/api-import/
3. Download the generated collection

### Method 4: Direct Import in Postman

1. Open Postman
2. Click "Import" > "Link"
3. Use: `http://localhost:8080/docs/swagger.json` (when server is running)
4. Postman will automatically convert and import

## What's Included

The generated Postman collection includes:

### 📁 **Folders by Category**
- **Sessions** - All session management endpoints
- **Metrics** - Analytics and usage statistics  
- **Search** - Session search functionality
- **Health** - API health check
- **WebSocket** - Real-time connection info

### 🔗 **All API Endpoints**
- `GET /api/v1/sessions` - List all sessions
- `GET /api/v1/sessions/{id}` - Get specific session
- `GET /api/v1/sessions/active` - Get active sessions
- `GET /api/v1/sessions/recent` - Get recent sessions
- `GET /api/v1/metrics/summary` - Overall metrics
- `GET /api/v1/metrics/activity` - Activity timeline
- `GET /api/v1/metrics/usage` - Usage statistics
- `GET /api/v1/search` - Search sessions
- `GET /api/v1/health` - Health check

### ⚙️ **Pre-configured Settings**
- Base URL: `http://localhost:8080/api/v1`
- Content-Type headers
- Example request parameters
- Response examples
- Query parameter validation

## Environment Variables

After importing, set up these Postman environment variables:

```json
{
  "baseUrl": "http://localhost:8080/api/v1",
  "sessionId": "session_123456"
}
```

## Testing Workflow

1. **Start the backend server:**
   ```bash
   make run
   ```

2. **Import the collection in Postman**

3. **Test basic endpoints:**
   - Health check: `GET {{baseUrl}}/health`
   - List sessions: `GET {{baseUrl}}/sessions`
   - Get metrics: `GET {{baseUrl}}/metrics/summary`

4. **Test with real data:**
   - Make sure you have Claude sessions in `~/.claude/projects/`
   - The API will return real session data

## Alternative Tools

If you prefer other tools, the Swagger JSON works with:

- **Insomnia**: Import > From URL > `http://localhost:8080/docs/swagger.json`
- **curl**: Generate curl commands from swagger
- **HTTPie**: Use with swagger definitions
- **REST Client**: VS Code extension for API testing

## Troubleshooting

### ❌ Blank URLs in Postman

**Problem:** After importing, request URLs show as blank or `{{baseUrl}}`

**Solution:** Set the collection variable in Postman:

1. **Import the collection**
2. **Right-click the collection name** in Postman sidebar
3. **Select "Edit"**
4. **Go to "Variables" tab**
5. **Set the variable:**
   - **Variable:** `baseUrl`
   - **Initial Value:** `http://localhost:8080/api/v1`
   - **Current Value:** `http://localhost:8080/api/v1`
6. **Save** and test requests

**Alternative:** Create a Postman Environment with the same variable.

### openapi2postmanv2 not found
```bash
npm install -g openapi-to-postmanv2
```

### Empty collection
- Make sure the backend server generated valid Swagger docs
- Check that `./docs/swagger.json` exists and is valid JSON

### Missing examples
- The collection includes all examples from our Swagger annotations
- If something is missing, it needs to be added to the backend swagger comments

## Development

To add new endpoints to the Postman collection:

1. Add Swagger annotations to the Go handler
2. Run `make swagger` to regenerate docs
3. Run `make postman` to update the collection
4. Re-import in Postman

The collection will automatically include new endpoints with their documentation.