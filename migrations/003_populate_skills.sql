-- Migration 003: Populate skills from metadata->tags for existing agents
-- Run with: psql -d mcp_serve -f migrations/003_populate_skills.sql

-- Update agents to have skills populated from metadata->tags
UPDATE agents
SET skills = (
    SELECT array_agg(tag::text)
    FROM jsonb_array_elements_text(metadata->'tags') AS tag
)
WHERE metadata->'tags' IS NOT NULL
  AND (skills IS NULL OR skills = '{}');

-- Add some derived skills based on agent names for common patterns
UPDATE agents SET skills = array_cat(skills, ARRAY['code', 'review', 'quality'])
WHERE name = 'code-reviewer' AND NOT ('review' = ANY(skills));

UPDATE agents SET skills = array_cat(skills, ARRAY['security', 'audit', 'vulnerability'])
WHERE name = 'security-auditor' AND NOT ('security' = ANY(skills));

UPDATE agents SET skills = array_cat(skills, ARRAY['test', 'testing', 'qa'])
WHERE name = 'test-engineer' AND NOT ('test' = ANY(skills));

UPDATE agents SET skills = array_cat(skills, ARRAY['documentation', 'docs', 'writing'])
WHERE name = 'documentation-writer' AND NOT ('documentation' = ANY(skills));

UPDATE agents SET skills = array_cat(skills, ARRAY['data', 'analysis', 'analytics'])
WHERE name = 'data-analyst' AND NOT ('data' = ANY(skills));

UPDATE agents SET skills = array_cat(skills, ARRAY['architecture', 'design', 'system'])
WHERE name = 'system-architect' AND NOT ('architecture' = ANY(skills));

UPDATE agents SET skills = array_cat(skills, ARRAY['devops', 'deploy', 'infrastructure'])
WHERE name = 'devops-engineer' AND NOT ('devops' = ANY(skills));

-- Ensure no NULL skills
UPDATE agents SET skills = '{}' WHERE skills IS NULL;

-- Verify
SELECT name, skills, metadata->'tags' as tags FROM agents ORDER BY name;
