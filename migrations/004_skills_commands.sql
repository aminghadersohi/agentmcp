-- Migration 004: Skills and Commands tables
-- Run with: psql -d mcp_serve -f migrations/004_skills_commands.sql

-- ============ Skills Table ============
-- Skills are packaged knowledge/documentation for specific tools
-- Examples: kubectl, docker-cli, curl, jenkins, datadog, terraform, aws-cli
CREATE TABLE IF NOT EXISTS skills (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) UNIQUE NOT NULL,
    version         VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    description     TEXT NOT NULL,
    category        VARCHAR(100),  -- devops, api, database, cloud, cli, etc.

    -- The actual skill content (knowledge, documentation, patterns)
    content         TEXT NOT NULL,
    examples        JSONB DEFAULT '[]',  -- Code examples with descriptions

    -- Metadata and discovery
    metadata        JSONB NOT NULL DEFAULT '{}',
    tags            TEXT[] DEFAULT '{}',

    -- Embeddings for semantic search
    embedding       vector(384),

    -- Reputation tracking
    reputation_score FLOAT DEFAULT 50.0,
    usage_count     INTEGER DEFAULT 0,
    feedback_count  INTEGER DEFAULT 0,
    avg_rating      FLOAT DEFAULT 0.0,

    -- Status
    status          VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'disabled')),
    is_system       BOOLEAN DEFAULT FALSE,

    -- Audit
    created_by      VARCHAR(255),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for skills
CREATE INDEX IF NOT EXISTS idx_skills_name ON skills (name);
CREATE INDEX IF NOT EXISTS idx_skills_category ON skills (category);
CREATE INDEX IF NOT EXISTS idx_skills_tags ON skills USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_skills_status ON skills (status);
CREATE INDEX IF NOT EXISTS idx_skills_embedding ON skills USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ============ Commands Table ============
-- Commands are reusable slash commands that can be synced to projects
CREATE TABLE IF NOT EXISTS commands (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) UNIQUE NOT NULL,  -- e.g., "review-pr", "fix-tests"
    version         VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    description     TEXT NOT NULL,

    -- Command content
    prompt          TEXT NOT NULL,  -- The actual command prompt/template
    arguments       JSONB DEFAULT '[]',  -- Expected arguments [{name, description, required}]

    -- Metadata and discovery
    metadata        JSONB NOT NULL DEFAULT '{}',
    tags            TEXT[] DEFAULT '{}',
    category        VARCHAR(100),  -- code, git, test, deploy, etc.

    -- Embeddings for semantic search
    embedding       vector(384),

    -- Reputation tracking
    reputation_score FLOAT DEFAULT 50.0,
    usage_count     INTEGER DEFAULT 0,
    feedback_count  INTEGER DEFAULT 0,
    avg_rating      FLOAT DEFAULT 0.0,

    -- Status
    status          VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'disabled')),
    is_system       BOOLEAN DEFAULT FALSE,

    -- Audit
    created_by      VARCHAR(255),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for commands
CREATE INDEX IF NOT EXISTS idx_commands_name ON commands (name);
CREATE INDEX IF NOT EXISTS idx_commands_category ON commands (category);
CREATE INDEX IF NOT EXISTS idx_commands_tags ON commands USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_commands_status ON commands (status);
CREATE INDEX IF NOT EXISTS idx_commands_embedding ON commands USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ============ Skill Feedback Table ============
CREATE TABLE IF NOT EXISTS skill_feedback (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id        UUID REFERENCES skills(id) ON DELETE CASCADE,
    rating          INTEGER CHECK (rating >= 1 AND rating <= 5),
    task_success    BOOLEAN,
    feedback_text   TEXT,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_skill_feedback_skill ON skill_feedback (skill_id);

-- ============ Command Feedback Table ============
CREATE TABLE IF NOT EXISTS command_feedback (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id      UUID REFERENCES commands(id) ON DELETE CASCADE,
    rating          INTEGER CHECK (rating >= 1 AND rating <= 5),
    task_success    BOOLEAN,
    feedback_text   TEXT,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_command_feedback_command ON command_feedback (command_id);

-- ============ Triggers ============
CREATE TRIGGER skills_updated_at
    BEFORE UPDATE ON skills
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER commands_updated_at
    BEFORE UPDATE ON commands
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- ============ Comments ============
COMMENT ON TABLE skills IS 'Packaged knowledge and documentation for tools (kubectl, docker, curl, etc.)';
COMMENT ON TABLE commands IS 'Reusable slash commands that can be synced to projects';
COMMENT ON TABLE skill_feedback IS 'User feedback on skill usefulness';
COMMENT ON TABLE command_feedback IS 'User feedback on command effectiveness';
