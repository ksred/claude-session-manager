-- Migration: Update session project paths from message CWD values
-- This migration updates sessions that have incorrect project_path values
-- by extracting the correct path from the CWD field in their messages

-- Update sessions where we have messages with CWD values
-- Note: We only update the project_path here. The project_name will be 
-- recalculated by the application code using filepath.Base()
UPDATE sessions
SET project_path = (
    SELECT m.cwd
    FROM messages m
    WHERE m.session_id = sessions.id
      AND m.cwd IS NOT NULL
      AND m.cwd != ''
    ORDER BY m.timestamp ASC
    LIMIT 1
),
updated_at = CURRENT_TIMESTAMP
WHERE EXISTS (
    SELECT 1
    FROM messages m
    WHERE m.session_id = sessions.id
      AND m.cwd IS NOT NULL
      AND m.cwd != ''
);

-- Log the update results
SELECT 'Updated ' || changes() || ' session records with correct project paths from CWD values' as migration_result;