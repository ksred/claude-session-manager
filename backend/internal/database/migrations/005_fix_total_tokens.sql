-- Migration: Fix total_tokens values in token_usage table
-- This updates any records where total_tokens is 0 but other token values exist

UPDATE token_usage
SET total_tokens = input_tokens + output_tokens + cache_creation_input_tokens + cache_read_input_tokens
WHERE total_tokens = 0 
AND (input_tokens > 0 OR output_tokens > 0 OR cache_creation_input_tokens > 0 OR cache_read_input_tokens > 0);

-- Log the update
SELECT 'Updated ' || changes() || ' token_usage records with corrected total_tokens values' as migration_result;