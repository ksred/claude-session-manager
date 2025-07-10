-- Migration: Recalculate token costs with correct pricing
-- This migration updates the estimated_cost column in token_usage table with accurate pricing

-- Create a temporary table to store recalculated costs
CREATE TEMPORARY TABLE token_cost_updates (
    message_id TEXT PRIMARY KEY,
    new_cost REAL
);

-- Insert recalculated costs based on correct pricing model
INSERT INTO token_cost_updates (message_id, new_cost)
SELECT 
    tu.message_id,
    CASE 
        -- Claude Opus 3 and Opus 4
        WHEN s.model LIKE '%claude-3-opus%' OR s.model LIKE '%claude-opus-4%' THEN
            (tu.input_tokens * 15.0 / 1000000) +
            (tu.output_tokens * 75.0 / 1000000) +
            (tu.cache_read_input_tokens * 1.50 / 1000000) +
            (tu.cache_creation_input_tokens * 18.75 / 1000000)
            
        -- Claude Sonnet 3.5, 3.7, 4
        WHEN s.model LIKE '%claude-3-5-sonnet%' OR s.model LIKE '%claude-3.5-sonnet%' 
             OR s.model LIKE '%claude-3-sonnet%' OR s.model LIKE '%claude-sonnet%' THEN
            (tu.input_tokens * 3.0 / 1000000) +
            (tu.output_tokens * 15.0 / 1000000) +
            (tu.cache_read_input_tokens * 0.30 / 1000000) +
            (tu.cache_creation_input_tokens * 3.75 / 1000000)
            
        -- Claude Haiku 3.5
        WHEN s.model LIKE '%claude-3-5-haiku%' OR s.model LIKE '%claude-3.5-haiku%' THEN
            (tu.input_tokens * 0.80 / 1000000) +
            (tu.output_tokens * 4.0 / 1000000) +
            (tu.cache_read_input_tokens * 0.08 / 1000000) +
            (tu.cache_creation_input_tokens * 1.0 / 1000000)
            
        -- Claude Haiku 3
        WHEN s.model LIKE '%claude-3-haiku%' THEN
            (tu.input_tokens * 0.25 / 1000000) +
            (tu.output_tokens * 1.25 / 1000000) +
            (tu.cache_read_input_tokens * 0.03 / 1000000) +
            (tu.cache_creation_input_tokens * 0.30 / 1000000)
            
        -- Default to Sonnet pricing for unknown models
        ELSE
            (tu.input_tokens * 3.0 / 1000000) +
            (tu.output_tokens * 15.0 / 1000000) +
            (tu.cache_read_input_tokens * 0.30 / 1000000) +
            (tu.cache_creation_input_tokens * 3.75 / 1000000)
    END as new_cost
FROM token_usage tu
JOIN sessions s ON tu.session_id = s.id;

-- Update the token_usage table with recalculated costs
UPDATE token_usage
SET estimated_cost = (
    SELECT new_cost 
    FROM token_cost_updates 
    WHERE token_cost_updates.message_id = token_usage.message_id
)
WHERE EXISTS (
    SELECT 1 
    FROM token_cost_updates 
    WHERE token_cost_updates.message_id = token_usage.message_id
);

-- Clean up temporary table
DROP TABLE token_cost_updates;

-- Log the migration
INSERT INTO activity_log (activity_type, details, timestamp)
VALUES ('migration', 'Recalculated token costs with correct pricing', CURRENT_TIMESTAMP);