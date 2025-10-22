# Activity and Timeline Query Fixes

## Issues Identified and Fixed

### Problem
Activity and timeline endpoints were sometimes returning only single values instead of the expected number of records (20-50). This was affecting:
- Activity timelines
- Token usage timelines (including cost metrics)
- Per-session timeline queries

### Root Causes

#### 1. **Activity Queries - SELECT DISTINCT Problem**
- `GetSessionActivity` and `GetProjectActivity` used `SELECT DISTINCT *` 
- This eliminated legitimate activity entries that appeared similar but weren't identical
- **Fix**: Replaced with explicit column selection and `ROW_NUMBER()` for unique IDs

#### 2. **Timeline Queries - JOIN Issues**
- `GetTokenTimeline` and `GetProjectTokenTimeline` used `INNER JOIN` with `token_usage`
- This excluded messages without token usage data, reducing result count
- **Fix**: Changed to `LEFT JOIN` and added `COALESCE()` for NULL handling

#### 3. **Inconsistent Limit Handling**
- Session and project activity handlers capped at 100 records
- Overall activity handler allowed up to 500 records
- **Fix**: Standardized all activity handlers to allow up to 500 records

## Changes Made

### 1. Activity Query Fixes

**File**: `backend/internal/database/session_repository.go`

**GetSessionActivity** and **GetProjectActivity**:
```sql
-- BEFORE (problematic)
SELECT DISTINCT * FROM combined_activity

-- AFTER (fixed)
SELECT 
    COALESCE(id, ROW_NUMBER() OVER (ORDER BY timestamp DESC)) as id,
    session_id,
    activity_type,
    details,
    timestamp,
    created_at
FROM combined_activity
```

### 2. Timeline Query Fixes

**File**: `backend/internal/database/session_repository.go`

**GetTokenTimeline** and **GetProjectTokenTimeline**:
```sql
-- BEFORE (problematic)
FROM messages m
JOIN token_usage tu ON m.id = tu.message_id
SUM(tu.input_tokens) as input_tokens,

-- AFTER (fixed)  
FROM messages m
LEFT JOIN token_usage tu ON m.id = tu.message_id
COALESCE(SUM(tu.input_tokens), 0) as input_tokens,
```

### 3. Limit Standardization

**File**: `backend/internal/api/handlers_sqlite.go`

**GetSessionActivityHandler** and **GetProjectActivityHandler**:
```go
// BEFORE (inconsistent)
limit := 50
if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
    limit = l
}

// AFTER (standardized)
limit, err := strconv.Atoi(limitStr)
if err != nil || limit <= 0 {
    limit = 50
}
if limit > 500 {
    limit = 500
}
```

## Impact

### Activity Endpoints
- **GetRecentActivity**: Returns up to 50/500 distinct activity entries
- **GetSessionActivity**: Returns up to 50/500 activity entries per session  
- **GetProjectActivity**: Returns up to 50/500 activity entries per project

### Timeline Endpoints (Including Metrics/Cost)
- **GetTokenTimeline**: Returns complete hourly/daily timeline data
- **GetSessionTokenTimeline**: Returns minute-by-minute session timeline
- **GetProjectTokenTimeline**: Returns project timeline with proper data points

### Data Consistency
- All timeline queries now include cost metrics (`estimated_cost`)
- All activity queries return consistent data structure
- Limit handling is standardized across all endpoints

## Testing Recommendations

1. **Activity Endpoints**: Test with sessions that have multiple message types (user, assistant, tool results)
2. **Timeline Endpoints**: Test with minute granularity for active sessions
3. **Limit Parameters**: Test with various limit values (1, 50, 100, 500)
4. **Edge Cases**: Test with sessions that have no token usage data

## Notes

- Timeline queries include all cost and token metrics in `TokenTimelineEntry`
- Activity queries combine data from multiple sources (messages, tool_results, activity_log)
- For per-session timelines, minute granularity is recommended as mentioned in requirements