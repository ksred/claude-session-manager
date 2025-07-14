-- Drop and recreate session_summary view to include source field
DROP VIEW IF EXISTS session_summary;

CREATE VIEW session_summary AS
SELECT 
    s.id,
    s.project_name,
    s.project_path,
    s.start_time,
    s.last_activity,
    s.is_active,
    s.status,
    s.model,
    s.message_count,
    s.duration_seconds,
    s.source,
    COALESCE(tu.total_input_tokens, 0) as total_input_tokens,
    COALESCE(tu.total_output_tokens, 0) as total_output_tokens,
    COALESCE(tu.total_cache_creation_tokens, 0) as total_cache_creation_tokens,
    COALESCE(tu.total_cache_read_tokens, 0) as total_cache_read_tokens,
    COALESCE(tu.total_tokens, 0) as total_tokens,
    COALESCE(tu.total_cost, 0.0) as total_estimated_cost,
    COALESCE(fr.modified_files, '[]') as files_modified
FROM sessions s
LEFT JOIN (
    SELECT 
        session_id,
        SUM(input_tokens) as total_input_tokens,
        SUM(output_tokens) as total_output_tokens,
        SUM(cache_creation_input_tokens) as total_cache_creation_tokens,
        SUM(cache_read_input_tokens) as total_cache_read_tokens,
        SUM(total_tokens) as total_tokens,
        SUM(estimated_cost) as total_cost
    FROM token_usage 
    GROUP BY session_id
) tu ON s.id = tu.session_id
LEFT JOIN (
    SELECT 
        session_id,
        JSON_GROUP_ARRAY(DISTINCT file_path) as modified_files
    FROM tool_results 
    WHERE file_path IS NOT NULL
    GROUP BY session_id
) fr ON s.id = fr.session_id;