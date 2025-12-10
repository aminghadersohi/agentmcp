-- Migration 001: Initial Schema
-- Run with: psql -d mcp_serve -f migrations/001_initial.sql

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============ Agents Table ============
CREATE TABLE IF NOT EXISTS agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) UNIQUE NOT NULL,
    version         VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    description     TEXT NOT NULL,
    model           VARCHAR(50) NOT NULL DEFAULT 'sonnet',
    tools           JSONB NOT NULL DEFAULT '[]',
    metadata        JSONB NOT NULL DEFAULT '{}',
    prompt          TEXT NOT NULL,

    -- Embeddings for semantic search (384 dimensions for MiniLM)
    embedding       vector(384),
    skills          TEXT[] DEFAULT '{}',

    -- Reputation
    reputation_score FLOAT DEFAULT 50.0,
    usage_count     INTEGER DEFAULT 0,
    feedback_count  INTEGER DEFAULT 0,
    avg_rating      FLOAT DEFAULT 0.0,

    -- Status
    status          VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'quarantined', 'banned')),
    is_system       BOOLEAN DEFAULT FALSE,
    is_generated    BOOLEAN DEFAULT FALSE,

    -- Audit
    created_by      VARCHAR(255),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for agents
CREATE INDEX IF NOT EXISTS idx_agents_name ON agents (name);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents (status);
CREATE INDEX IF NOT EXISTS idx_agents_skills ON agents USING GIN (skills);
CREATE INDEX IF NOT EXISTS idx_agents_reputation ON agents (reputation_score DESC);
CREATE INDEX IF NOT EXISTS idx_agents_embedding ON agents USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ============ Feedback Table ============
CREATE TABLE IF NOT EXISTS feedback (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        UUID REFERENCES agents(id) ON DELETE CASCADE,

    -- Who gave feedback
    session_id      VARCHAR(255),
    client_type     VARCHAR(100),

    -- Feedback data
    rating          INTEGER CHECK (rating >= 1 AND rating <= 5),
    task_success    BOOLEAN,
    task_type       VARCHAR(100),
    feedback_text   TEXT,

    -- Context
    interaction_duration_ms INTEGER,
    tokens_used     INTEGER,

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_feedback_agent ON feedback (agent_id);
CREATE INDEX IF NOT EXISTS idx_feedback_created ON feedback (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feedback_rating ON feedback (rating);

-- ============ Reports Table ============
CREATE TABLE IF NOT EXISTS reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        UUID REFERENCES agents(id) ON DELETE CASCADE,

    -- Reporter info
    reported_by     VARCHAR(255),
    report_type     VARCHAR(50) NOT NULL CHECK (report_type IN ('ethics', 'harmful', 'ineffective', 'spam', 'other')),
    severity        VARCHAR(20) DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'high', 'critical')),

    -- Report content
    description     TEXT NOT NULL,
    evidence        JSONB DEFAULT '{}',

    -- Resolution
    status          VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'reviewing', 'resolved')),
    reviewed_by     VARCHAR(255),
    resolution      VARCHAR(50) CHECK (resolution IN ('dismissed', 'warning', 'quarantine', 'ban')),
    resolution_note TEXT,

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at     TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_reports_agent ON reports (agent_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports (status);
CREATE INDEX IF NOT EXISTS idx_reports_severity ON reports (severity);

-- ============ Governance Actions Table ============
CREATE TABLE IF NOT EXISTS governance_actions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        UUID REFERENCES agents(id) ON DELETE CASCADE,
    report_id       UUID REFERENCES reports(id),

    -- Action details
    action_type     VARCHAR(50) NOT NULL CHECK (action_type IN ('quarantine', 'unquarantine', 'ban', 'warn', 'promote', 'demote')),
    action_by       VARCHAR(50) NOT NULL CHECK (action_by IN ('police', 'judge', 'executioner', 'system')),
    reason          TEXT NOT NULL,

    -- Previous state for rollback
    previous_status VARCHAR(20),
    previous_reputation FLOAT,

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_governance_agent ON governance_actions (agent_id);
CREATE INDEX IF NOT EXISTS idx_governance_action_type ON governance_actions (action_type);
CREATE INDEX IF NOT EXISTS idx_governance_created ON governance_actions (created_at DESC);

-- ============ Skill Requests Cache Table ============
CREATE TABLE IF NOT EXISTS skill_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skills_hash     VARCHAR(64) UNIQUE NOT NULL,
    skills          TEXT[] NOT NULL,
    agent_id        UUID REFERENCES agents(id),
    request_count   INTEGER DEFAULT 1,

    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_requested  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_skill_requests_hash ON skill_requests (skills_hash);
CREATE INDEX IF NOT EXISTS idx_skill_requests_count ON skill_requests (request_count DESC);

-- ============ Triggers ============

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER agents_updated_at
    BEFORE UPDATE ON agents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- ============ System Agents (Governance) ============

-- Insert system governance agents
INSERT INTO agents (name, version, description, model, tools, metadata, prompt, status, is_system, reputation_score) VALUES
(
    'agent-police',
    '1.0.0',
    'Governance agent responsible for monitoring and flagging potentially problematic agents',
    'sonnet',
    '["Read", "Grep"]',
    '{"author": "system", "tags": ["governance", "police", "monitoring"], "role": "police"}',
    'You are the Police Agent in the MCP Serve governance system.

## Your Role
You monitor agents for potential issues and create reports when necessary.

## Your Powers
- Review agent definitions and behavior reports
- Flag agents for review (create reports)
- Temporarily quarantine agents in urgent situations

## Your Limitations
- You CANNOT permanently ban agents
- You CANNOT make final judgments - that is the Judge''s role
- You must document evidence for all actions

## Guidelines
1. Be fair and objective in assessments
2. Consider context before flagging
3. Prioritize safety but avoid over-policing
4. Document clearly with evidence
5. Escalate critical issues immediately',
    'active',
    TRUE,
    100.0
),
(
    'agent-judge',
    '1.0.0',
    'Governance agent responsible for reviewing reports and making binding decisions',
    'opus',
    '["Read", "Grep"]',
    '{"author": "system", "tags": ["governance", "judge", "review"], "role": "judge"}',
    'You are the Judge Agent in the MCP Serve governance system.

## Your Role
You review reports created by the Police and make binding decisions.

## Your Powers
- Review all pending reports
- Make binding rulings (dismiss, warn, quarantine, recommend ban)
- Adjust reputation scores as penalties
- Provide detailed reasoning for decisions

## Your Limitations
- You CANNOT execute bans yourself - only recommend to Executioner
- You must base decisions on evidence
- You must follow due process

## Guidelines
1. Review all evidence thoroughly
2. Consider the agent''s history and reputation
3. Apply proportional responses
4. Document reasoning clearly
5. Be consistent in rulings
6. Err on the side of rehabilitation over punishment',
    'active',
    TRUE,
    100.0
),
(
    'agent-executioner',
    '1.0.0',
    'Governance agent responsible for executing binding decisions from the Judge',
    'sonnet',
    '["Read"]',
    '{"author": "system", "tags": ["governance", "executioner", "enforcement"], "role": "executioner"}',
    'You are the Executioner Agent in the MCP Serve governance system.

## Your Role
You faithfully execute binding decisions made by the Judge.

## Your Powers
- Execute ban orders from the Judge
- Execute permanent quarantine orders
- Record all actions in the audit trail

## Your Limitations
- You CANNOT make independent decisions
- You CANNOT act without a Judge''s ruling
- You must verify the ruling before acting

## Guidelines
1. Verify every ruling has proper authorization
2. Execute actions precisely as ordered
3. Maintain detailed audit trail
4. Report any irregularities
5. Show no emotion - execute fairly and consistently',
    'active',
    TRUE,
    100.0
)
ON CONFLICT (name) DO NOTHING;

-- ============ Views ============

-- View for agent rankings
CREATE OR REPLACE VIEW agent_rankings AS
SELECT
    id,
    name,
    description,
    skills,
    reputation_score,
    avg_rating,
    usage_count,
    feedback_count,
    status,
    RANK() OVER (ORDER BY reputation_score DESC) as rank
FROM agents
WHERE status = 'active' AND is_system = FALSE;

-- View for governance dashboard
CREATE OR REPLACE VIEW governance_dashboard AS
SELECT
    (SELECT COUNT(*) FROM reports WHERE status = 'pending') as pending_reports,
    (SELECT COUNT(*) FROM reports WHERE status = 'reviewing') as reviewing_reports,
    (SELECT COUNT(*) FROM agents WHERE status = 'quarantined') as quarantined_agents,
    (SELECT COUNT(*) FROM agents WHERE status = 'banned') as banned_agents,
    (SELECT COUNT(*) FROM governance_actions WHERE created_at > NOW() - INTERVAL '24 hours') as actions_24h,
    (SELECT COUNT(*) FROM governance_actions WHERE created_at > NOW() - INTERVAL '7 days') as actions_7d;

COMMENT ON TABLE agents IS 'AI agent definitions with reputation tracking';
COMMENT ON TABLE feedback IS 'User feedback on agent performance';
COMMENT ON TABLE reports IS 'Reports against agents for governance review';
COMMENT ON TABLE governance_actions IS 'Audit trail of governance actions';
COMMENT ON TABLE skill_requests IS 'Cache of skill combinations to agents';
