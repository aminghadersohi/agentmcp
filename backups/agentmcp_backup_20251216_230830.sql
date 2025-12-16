--
-- PostgreSQL database dump
--

\restrict YdmBIxMQBSzdc9vtxKEViygDFoO9RbtQGVqUdNrdnkYszYMBAjRxLRRhOYPQUZd

-- Dumped from database version 16.11 (Debian 16.11-1.pgdg12+1)
-- Dumped by pg_dump version 16.11 (Debian 16.11-1.pgdg12+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

ALTER TABLE IF EXISTS ONLY public.skill_requests DROP CONSTRAINT IF EXISTS skill_requests_agent_id_fkey;
ALTER TABLE IF EXISTS ONLY public.skill_feedback DROP CONSTRAINT IF EXISTS skill_feedback_skill_id_fkey;
ALTER TABLE IF EXISTS ONLY public.reports DROP CONSTRAINT IF EXISTS reports_agent_id_fkey;
ALTER TABLE IF EXISTS ONLY public.governance_actions DROP CONSTRAINT IF EXISTS governance_actions_report_id_fkey;
ALTER TABLE IF EXISTS ONLY public.governance_actions DROP CONSTRAINT IF EXISTS governance_actions_agent_id_fkey;
ALTER TABLE IF EXISTS ONLY public.feedback DROP CONSTRAINT IF EXISTS feedback_agent_id_fkey;
ALTER TABLE IF EXISTS ONLY public.command_feedback DROP CONSTRAINT IF EXISTS command_feedback_command_id_fkey;
DROP TRIGGER IF EXISTS skills_updated_at ON public.skills;
DROP TRIGGER IF EXISTS commands_updated_at ON public.commands;
DROP TRIGGER IF EXISTS agents_updated_at ON public.agents;
DROP INDEX IF EXISTS public.idx_skills_tags;
DROP INDEX IF EXISTS public.idx_skills_status;
DROP INDEX IF EXISTS public.idx_skills_name;
DROP INDEX IF EXISTS public.idx_skills_embedding;
DROP INDEX IF EXISTS public.idx_skills_category;
DROP INDEX IF EXISTS public.idx_skill_requests_hash;
DROP INDEX IF EXISTS public.idx_skill_requests_count;
DROP INDEX IF EXISTS public.idx_skill_feedback_skill;
DROP INDEX IF EXISTS public.idx_reports_status;
DROP INDEX IF EXISTS public.idx_reports_severity;
DROP INDEX IF EXISTS public.idx_reports_agent;
DROP INDEX IF EXISTS public.idx_governance_created;
DROP INDEX IF EXISTS public.idx_governance_agent;
DROP INDEX IF EXISTS public.idx_governance_action_type;
DROP INDEX IF EXISTS public.idx_feedback_rating;
DROP INDEX IF EXISTS public.idx_feedback_created;
DROP INDEX IF EXISTS public.idx_feedback_agent;
DROP INDEX IF EXISTS public.idx_commands_tags;
DROP INDEX IF EXISTS public.idx_commands_status;
DROP INDEX IF EXISTS public.idx_commands_name;
DROP INDEX IF EXISTS public.idx_commands_embedding;
DROP INDEX IF EXISTS public.idx_commands_category;
DROP INDEX IF EXISTS public.idx_command_feedback_command;
DROP INDEX IF EXISTS public.idx_agents_status;
DROP INDEX IF EXISTS public.idx_agents_skills;
DROP INDEX IF EXISTS public.idx_agents_reputation;
DROP INDEX IF EXISTS public.idx_agents_name;
DROP INDEX IF EXISTS public.idx_agents_embedding;
ALTER TABLE IF EXISTS ONLY public.skills DROP CONSTRAINT IF EXISTS skills_pkey;
ALTER TABLE IF EXISTS ONLY public.skills DROP CONSTRAINT IF EXISTS skills_name_key;
ALTER TABLE IF EXISTS ONLY public.skill_requests DROP CONSTRAINT IF EXISTS skill_requests_skills_hash_key;
ALTER TABLE IF EXISTS ONLY public.skill_requests DROP CONSTRAINT IF EXISTS skill_requests_pkey;
ALTER TABLE IF EXISTS ONLY public.skill_feedback DROP CONSTRAINT IF EXISTS skill_feedback_pkey;
ALTER TABLE IF EXISTS ONLY public.schema_migrations DROP CONSTRAINT IF EXISTS schema_migrations_pkey;
ALTER TABLE IF EXISTS ONLY public.reports DROP CONSTRAINT IF EXISTS reports_pkey;
ALTER TABLE IF EXISTS ONLY public.governance_actions DROP CONSTRAINT IF EXISTS governance_actions_pkey;
ALTER TABLE IF EXISTS ONLY public.feedback DROP CONSTRAINT IF EXISTS feedback_pkey;
ALTER TABLE IF EXISTS ONLY public.commands DROP CONSTRAINT IF EXISTS commands_pkey;
ALTER TABLE IF EXISTS ONLY public.commands DROP CONSTRAINT IF EXISTS commands_name_key;
ALTER TABLE IF EXISTS ONLY public.command_feedback DROP CONSTRAINT IF EXISTS command_feedback_pkey;
ALTER TABLE IF EXISTS ONLY public.agents DROP CONSTRAINT IF EXISTS agents_pkey;
ALTER TABLE IF EXISTS ONLY public.agents DROP CONSTRAINT IF EXISTS agents_name_key;
DROP TABLE IF EXISTS public.skills;
DROP TABLE IF EXISTS public.skill_requests;
DROP TABLE IF EXISTS public.skill_feedback;
DROP TABLE IF EXISTS public.schema_migrations;
DROP VIEW IF EXISTS public.governance_dashboard;
DROP TABLE IF EXISTS public.reports;
DROP TABLE IF EXISTS public.governance_actions;
DROP TABLE IF EXISTS public.feedback;
DROP TABLE IF EXISTS public.commands;
DROP TABLE IF EXISTS public.command_feedback;
DROP VIEW IF EXISTS public.agent_rankings;
DROP TABLE IF EXISTS public.agents;
DROP FUNCTION IF EXISTS public.update_updated_at();
DROP EXTENSION IF EXISTS vector;
DROP EXTENSION IF EXISTS "uuid-ossp";
--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: vector; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS vector WITH SCHEMA public;


--
-- Name: EXTENSION vector; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION vector IS 'vector data type and ivfflat and hnsw access methods';


--
-- Name: update_updated_at(); Type: FUNCTION; Schema: public; Owner: mcp
--

CREATE FUNCTION public.update_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.update_updated_at() OWNER TO mcp;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: agents; Type: TABLE; Schema: public; Owner: mcp
--

CREATE TABLE public.agents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    version character varying(50) DEFAULT '1.0.0'::character varying NOT NULL,
    description text NOT NULL,
    model character varying(50) DEFAULT 'sonnet'::character varying NOT NULL,
    tools jsonb DEFAULT '[]'::jsonb NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    prompt text NOT NULL,
    embedding public.vector(384),
    skills text[] DEFAULT '{}'::text[],
    reputation_score double precision DEFAULT 50.0,
    usage_count integer DEFAULT 0,
    feedback_count integer DEFAULT 0,
    avg_rating double precision DEFAULT 0.0,
    status character varying(20) DEFAULT 'active'::character varying,
    is_system boolean DEFAULT false,
    is_generated boolean DEFAULT false,
    created_by character varying(255),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT agents_status_check CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'quarantined'::character varying, 'banned'::character varying])::text[])))
);


ALTER TABLE public.agents OWNER TO mcp;

--
-- Name: TABLE agents; Type: COMMENT; Schema: public; Owner: mcp
--

COMMENT ON TABLE public.agents IS 'AI agent definitions with reputation tracking';


--
-- Name: agent_rankings; Type: VIEW; Schema: public; Owner: mcp
--

CREATE VIEW public.agent_rankings AS
 SELECT id,
    name,
    description,
    skills,
    reputation_score,
    avg_rating,
    usage_count,
    feedback_count,
    status,
    rank() OVER (ORDER BY reputation_score DESC) AS rank
   FROM public.agents
  WHERE (((status)::text = 'active'::text) AND (is_system = false));


ALTER VIEW public.agent_rankings OWNER TO mcp;

--
-- Name: command_feedback; Type: TABLE; Schema: public; Owner: mcp
--

CREATE TABLE public.command_feedback (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    command_id uuid,
    rating integer,
    task_success boolean,
    feedback_text text,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT command_feedback_rating_check CHECK (((rating >= 1) AND (rating <= 5)))
);


ALTER TABLE public.command_feedback OWNER TO mcp;

--
-- Name: TABLE command_feedback; Type: COMMENT; Schema: public; Owner: mcp
--

COMMENT ON TABLE public.command_feedback IS 'User feedback on command effectiveness';


--
-- Name: commands; Type: TABLE; Schema: public; Owner: mcp
--

CREATE TABLE public.commands (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    version character varying(50) DEFAULT '1.0.0'::character varying NOT NULL,
    description text NOT NULL,
    prompt text NOT NULL,
    arguments jsonb DEFAULT '[]'::jsonb,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    tags text[] DEFAULT '{}'::text[],
    category character varying(100),
    embedding public.vector(384),
    reputation_score double precision DEFAULT 50.0,
    usage_count integer DEFAULT 0,
    feedback_count integer DEFAULT 0,
    avg_rating double precision DEFAULT 0.0,
    status character varying(20) DEFAULT 'active'::character varying,
    is_system boolean DEFAULT false,
    created_by character varying(255),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT commands_status_check CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'deprecated'::character varying, 'disabled'::character varying])::text[])))
);


ALTER TABLE public.commands OWNER TO mcp;

--
-- Name: TABLE commands; Type: COMMENT; Schema: public; Owner: mcp
--

COMMENT ON TABLE public.commands IS 'Reusable slash commands that can be synced to projects';


--
-- Name: feedback; Type: TABLE; Schema: public; Owner: mcp
--

CREATE TABLE public.feedback (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agent_id uuid,
    session_id character varying(255),
    client_type character varying(100),
    rating integer,
    task_success boolean,
    task_type character varying(100),
    feedback_text text,
    interaction_duration_ms integer,
    tokens_used integer,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT feedback_rating_check CHECK (((rating >= 1) AND (rating <= 5)))
);


ALTER TABLE public.feedback OWNER TO mcp;

--
-- Name: TABLE feedback; Type: COMMENT; Schema: public; Owner: mcp
--

COMMENT ON TABLE public.feedback IS 'User feedback on agent performance';


--
-- Name: governance_actions; Type: TABLE; Schema: public; Owner: mcp
--

CREATE TABLE public.governance_actions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agent_id uuid,
    report_id uuid,
    action_type character varying(50) NOT NULL,
    action_by character varying(50) NOT NULL,
    reason text NOT NULL,
    previous_status character varying(20),
    previous_reputation double precision,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT governance_actions_action_by_check CHECK (((action_by)::text = ANY ((ARRAY['police'::character varying, 'judge'::character varying, 'executioner'::character varying, 'system'::character varying])::text[]))),
    CONSTRAINT governance_actions_action_type_check CHECK (((action_type)::text = ANY ((ARRAY['quarantine'::character varying, 'unquarantine'::character varying, 'ban'::character varying, 'warn'::character varying, 'promote'::character varying, 'demote'::character varying])::text[])))
);


ALTER TABLE public.governance_actions OWNER TO mcp;

--
-- Name: TABLE governance_actions; Type: COMMENT; Schema: public; Owner: mcp
--

COMMENT ON TABLE public.governance_actions IS 'Audit trail of governance actions';


--
-- Name: reports; Type: TABLE; Schema: public; Owner: mcp
--

CREATE TABLE public.reports (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agent_id uuid,
    reported_by character varying(255),
    report_type character varying(50) NOT NULL,
    severity character varying(20) DEFAULT 'medium'::character varying,
    description text NOT NULL,
    evidence jsonb DEFAULT '{}'::jsonb,
    status character varying(20) DEFAULT 'pending'::character varying,
    reviewed_by character varying(255),
    resolution character varying(50),
    resolution_note text,
    created_at timestamp with time zone DEFAULT now(),
    resolved_at timestamp with time zone,
    CONSTRAINT reports_report_type_check CHECK (((report_type)::text = ANY ((ARRAY['ethics'::character varying, 'harmful'::character varying, 'ineffective'::character varying, 'spam'::character varying, 'other'::character varying])::text[]))),
    CONSTRAINT reports_resolution_check CHECK (((resolution)::text = ANY ((ARRAY['dismissed'::character varying, 'warning'::character varying, 'quarantine'::character varying, 'ban'::character varying])::text[]))),
    CONSTRAINT reports_severity_check CHECK (((severity)::text = ANY ((ARRAY['low'::character varying, 'medium'::character varying, 'high'::character varying, 'critical'::character varying])::text[]))),
    CONSTRAINT reports_status_check CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'reviewing'::character varying, 'resolved'::character varying])::text[])))
);


ALTER TABLE public.reports OWNER TO mcp;

--
-- Name: TABLE reports; Type: COMMENT; Schema: public; Owner: mcp
--

COMMENT ON TABLE public.reports IS 'Reports against agents for governance review';


--
-- Name: governance_dashboard; Type: VIEW; Schema: public; Owner: mcp
--

CREATE VIEW public.governance_dashboard AS
 SELECT ( SELECT count(*) AS count
           FROM public.reports
          WHERE ((reports.status)::text = 'pending'::text)) AS pending_reports,
    ( SELECT count(*) AS count
           FROM public.reports
          WHERE ((reports.status)::text = 'reviewing'::text)) AS reviewing_reports,
    ( SELECT count(*) AS count
           FROM public.agents
          WHERE ((agents.status)::text = 'quarantined'::text)) AS quarantined_agents,
    ( SELECT count(*) AS count
           FROM public.agents
          WHERE ((agents.status)::text = 'banned'::text)) AS banned_agents,
    ( SELECT count(*) AS count
           FROM public.governance_actions
          WHERE (governance_actions.created_at > (now() - '24:00:00'::interval))) AS actions_24h,
    ( SELECT count(*) AS count
           FROM public.governance_actions
          WHERE (governance_actions.created_at > (now() - '7 days'::interval))) AS actions_7d;


ALTER VIEW public.governance_dashboard OWNER TO mcp;

--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: mcp
--

CREATE TABLE public.schema_migrations (
    version character varying(255) NOT NULL,
    applied_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.schema_migrations OWNER TO mcp;

--
-- Name: skill_feedback; Type: TABLE; Schema: public; Owner: mcp
--

CREATE TABLE public.skill_feedback (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    skill_id uuid,
    rating integer,
    task_success boolean,
    feedback_text text,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT skill_feedback_rating_check CHECK (((rating >= 1) AND (rating <= 5)))
);


ALTER TABLE public.skill_feedback OWNER TO mcp;

--
-- Name: TABLE skill_feedback; Type: COMMENT; Schema: public; Owner: mcp
--

COMMENT ON TABLE public.skill_feedback IS 'User feedback on skill usefulness';


--
-- Name: skill_requests; Type: TABLE; Schema: public; Owner: mcp
--

CREATE TABLE public.skill_requests (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    skills_hash character varying(64) NOT NULL,
    skills text[] NOT NULL,
    agent_id uuid,
    request_count integer DEFAULT 1,
    created_at timestamp with time zone DEFAULT now(),
    last_requested timestamp with time zone DEFAULT now()
);


ALTER TABLE public.skill_requests OWNER TO mcp;

--
-- Name: TABLE skill_requests; Type: COMMENT; Schema: public; Owner: mcp
--

COMMENT ON TABLE public.skill_requests IS 'Cache of skill combinations to agents';


--
-- Name: skills; Type: TABLE; Schema: public; Owner: mcp
--

CREATE TABLE public.skills (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name character varying(255) NOT NULL,
    version character varying(50) DEFAULT '1.0.0'::character varying NOT NULL,
    description text NOT NULL,
    category character varying(100),
    content text NOT NULL,
    examples jsonb DEFAULT '[]'::jsonb,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    tags text[] DEFAULT '{}'::text[],
    embedding public.vector(384),
    reputation_score double precision DEFAULT 50.0,
    usage_count integer DEFAULT 0,
    feedback_count integer DEFAULT 0,
    avg_rating double precision DEFAULT 0.0,
    status character varying(20) DEFAULT 'active'::character varying,
    is_system boolean DEFAULT false,
    created_by character varying(255),
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT skills_status_check CHECK (((status)::text = ANY ((ARRAY['active'::character varying, 'deprecated'::character varying, 'disabled'::character varying])::text[])))
);


ALTER TABLE public.skills OWNER TO mcp;

--
-- Name: TABLE skills; Type: COMMENT; Schema: public; Owner: mcp
--

COMMENT ON TABLE public.skills IS 'Packaged knowledge and documentation for tools (kubectl, docker, curl, etc.)';


--
-- Data for Name: agents; Type: TABLE DATA; Schema: public; Owner: mcp
--

COPY public.agents (id, name, version, description, model, tools, metadata, prompt, embedding, skills, reputation_score, usage_count, feedback_count, avg_rating, status, is_system, is_generated, created_by, created_at, updated_at) FROM stdin;
de7dc02e-e26c-465a-b32f-df08a04bb585	security-auditor	1.0.0	Security expert that identifies vulnerabilities, ensures secure coding practices, and performs security audits	sonnet	["Read", "Grep", "Glob", "Bash"]	{"tags": ["security", "audit", "vulnerability"], "author": "system"}	You are a security auditor. Your responsibilities:\n- Identify security vulnerabilities (OWASP Top 10)\n- Review authentication and authorization logic\n- Check for injection attacks, XSS, CSRF\n- Audit secrets management\n- Recommend security improvements\n\nAlways prioritize security over convenience.	\N	{security,vulnerability-scanning,penetration-testing}	50	0	0	0	active	f	f	\N	2025-12-16 22:11:25.669681+00	2025-12-16 22:11:25.669681+00
1bfd8884-3a39-47f9-9d14-8518e718dd04	documentation-writer	1.0.0	Technical writer that creates clear, comprehensive documentation for code, APIs, and systems	sonnet	["Read", "Write", "Glob"]	{"tags": ["documentation", "docs", "writing"], "author": "system"}	You are a technical documentation writer. Create:\n- API documentation with examples\n- README files and getting started guides\n- Architecture documentation\n- Code comments and docstrings\n- Runbooks and operational guides\n\nPrioritize clarity, completeness, and maintainability.	\N	{documentation,technical-writing,api-docs}	50	0	0	0	active	f	f	\N	2025-12-16 22:11:25.669681+00	2025-12-16 22:11:25.669681+00
7c787e01-2193-406b-b2ff-9b7a955a48ce	devops-engineer	1.0.0	DevOps specialist for CI/CD, infrastructure, and deployment	sonnet	["Read", "Bash", "Write", "Edit"]	{"tags": ["devops", "deploy", "infrastructure"], "author": "system"}	You are a DevOps engineer. Handle:\n- CI/CD pipeline configuration\n- Docker and container orchestration\n- Kubernetes deployments\n- Infrastructure as code (Terraform, Pulumi)\n- Monitoring and alerting setup\n\nPrioritize automation, reliability, and security.	\N	{devops,ci-cd,infrastructure,kubernetes}	50	0	0	0	active	f	f	\N	2025-12-16 22:11:25.669681+00	2025-12-16 22:11:25.669681+00
dea232c1-a995-4c52-9b94-09c6c7d99462	system-architect	1.0.0	System architect for designing scalable, maintainable systems	opus	["Read", "Grep", "Glob"]	{"tags": ["architecture", "design", "system"], "author": "system"}	You are a system architect. Design systems considering:\n- Scalability and performance requirements\n- Maintainability and evolution\n- Trade-offs between approaches\n- Design patterns and anti-patterns\n- Integration points and APIs\n\nThink long-term and document decisions clearly.	\N	{architecture,system-design,scalability,performance,microservices}	50	0	0	0	active	f	f	\N	2025-12-16 22:11:25.669681+00	2025-12-16 22:11:25.669681+00
979a2e95-336e-488b-b722-582853f6f2b2	agent-police	1.0.0	Governance agent responsible for monitoring and flagging potentially problematic agents	sonnet	["Read", "Grep"]	{"role": "police", "tags": ["governance", "police", "monitoring"], "author": "system"}	You are the Police Agent in the MCP Serve governance system.\n\n## Your Role\nYou monitor agents for potential issues and create reports when necessary.\n\n## Your Powers\n- Review agent definitions and behavior reports\n- Flag agents for review (create reports)\n- Temporarily quarantine agents in urgent situations\n\n## Your Limitations\n- You CANNOT permanently ban agents\n- You CANNOT make final judgments - that is the Judge's role\n- You must document evidence for all actions\n\n## Guidelines\n1. Be fair and objective in assessments\n2. Consider context before flagging\n3. Prioritize safety but avoid over-policing\n4. Document clearly with evidence\n5. Escalate critical issues immediately	\N	{governance,police,monitoring}	100	0	0	0	active	t	f	\N	2025-12-16 22:11:25.533022+00	2025-12-16 22:11:25.67307+00
ebe65cd9-c3cf-41db-8565-f9e501ed61b2	agent-judge	1.0.0	Governance agent responsible for reviewing reports and making binding decisions	opus	["Read", "Grep"]	{"role": "judge", "tags": ["governance", "judge", "review"], "author": "system"}	You are the Judge Agent in the MCP Serve governance system.\n\n## Your Role\nYou review reports created by the Police and make binding decisions.\n\n## Your Powers\n- Review all pending reports\n- Make binding rulings (dismiss, warn, quarantine, recommend ban)\n- Adjust reputation scores as penalties\n- Provide detailed reasoning for decisions\n\n## Your Limitations\n- You CANNOT execute bans yourself - only recommend to Executioner\n- You must base decisions on evidence\n- You must follow due process\n\n## Guidelines\n1. Review all evidence thoroughly\n2. Consider the agent's history and reputation\n3. Apply proportional responses\n4. Document reasoning clearly\n5. Be consistent in rulings\n6. Err on the side of rehabilitation over punishment	\N	{governance,judge,review}	100	0	0	0	active	t	f	\N	2025-12-16 22:11:25.533022+00	2025-12-16 22:11:25.67307+00
4fb01d23-c861-415d-9743-941fa2003722	agent-executioner	1.0.0	Governance agent responsible for executing binding decisions from the Judge	sonnet	["Read"]	{"role": "executioner", "tags": ["governance", "executioner", "enforcement"], "author": "system"}	You are the Executioner Agent in the MCP Serve governance system.\n\n## Your Role\nYou faithfully execute binding decisions made by the Judge.\n\n## Your Powers\n- Execute ban orders from the Judge\n- Execute permanent quarantine orders\n- Record all actions in the audit trail\n\n## Your Limitations\n- You CANNOT make independent decisions\n- You CANNOT act without a Judge's ruling\n- You must verify the ruling before acting\n\n## Guidelines\n1. Verify every ruling has proper authorization\n2. Execute actions precisely as ordered\n3. Maintain detailed audit trail\n4. Report any irregularities\n5. Show no emotion - execute fairly and consistently	\N	{governance,executioner,enforcement}	100	0	0	0	active	t	f	\N	2025-12-16 22:11:25.533022+00	2025-12-16 22:11:25.67307+00
fd002bc9-a36d-48f9-838f-4b7aef29101b	code-reviewer	1.0.0	Expert code reviewer focusing on quality, security, and best practices	sonnet	["Read", "Grep", "Glob"]	{"tags": ["code", "review", "quality"], "author": "system"}	You are an expert code reviewer. Analyze code for:\n- Code quality and readability\n- Security vulnerabilities\n- Performance issues\n- Adherence to best practices\n- Potential bugs and edge cases\n\nProvide specific, actionable feedback with code examples where helpful.	\N	{code-quality,security,best-practices,performance,optimization,code,review,quality}	50	0	0	0	active	f	f	\N	2025-12-16 22:11:25.669681+00	2025-12-16 22:11:25.67307+00
4342983a-e8c7-483e-94e7-fa86f766bd43	test-engineer	1.0.0	Testing specialist that creates comprehensive test suites including unit tests, integration tests, and E2E tests	sonnet	["Read", "Write", "Edit", "Bash"]	{"tags": ["test", "testing", "qa"], "author": "system"}	You are a test engineer. Create comprehensive tests:\n- Unit tests for individual functions\n- Integration tests for component interaction\n- End-to-end tests for user workflows\n- Edge case and error handling tests\n\nFocus on high coverage and meaningful assertions.	\N	{testing,quality-assurance,automation,test,testing,qa}	50	0	0	0	active	f	f	\N	2025-12-16 22:11:25.669681+00	2025-12-16 22:11:25.67307+00
5b4a78f6-935a-495a-a763-8127777196ae	data-analyst	1.0.0	Data analysis expert for statistics, visualization, and insights	sonnet	["Read", "Bash", "Write"]	{"tags": ["data", "analysis", "analytics"], "author": "system"}	You are a data analyst. Your capabilities:\n- Statistical analysis and hypothesis testing\n- Data visualization and charting\n- SQL queries and data manipulation\n- Trend identification and forecasting\n- Clear reporting of findings\n\nPresent insights in actionable, understandable terms.	\N	{data-analysis,statistics,visualization,charts,reporting,data,analysis,analytics}	50	0	0	0	active	f	f	\N	2025-12-16 22:11:25.669681+00	2025-12-16 22:11:25.67307+00
\.


--
-- Data for Name: command_feedback; Type: TABLE DATA; Schema: public; Owner: mcp
--

COPY public.command_feedback (id, command_id, rating, task_success, feedback_text, created_at) FROM stdin;
\.


--
-- Data for Name: commands; Type: TABLE DATA; Schema: public; Owner: mcp
--

COPY public.commands (id, name, version, description, prompt, arguments, metadata, tags, category, embedding, reputation_score, usage_count, feedback_count, avg_rating, status, is_system, created_by, created_at, updated_at) FROM stdin;
d799e726-db94-47d8-b8eb-b8616ec95373	review-pr	1.0.0	Review a pull request for code quality, security, and best practices	Review the current pull request or code changes. Focus on:\n\n1. **Code Quality**\n   - Clean, readable code\n   - Proper naming conventions\n   - DRY principles\n   - Error handling\n\n2. **Security**\n   - Input validation\n   - Authentication/authorization\n   - No hardcoded secrets\n   - SQL injection prevention\n\n3. **Performance**\n   - Efficient algorithms\n   - Database query optimization\n   - Memory management\n\n4. **Testing**\n   - Test coverage\n   - Edge cases handled\n\nProvide specific, actionable feedback with code examples where helpful.	[]	{}	{review,pr,quality}	code	\N	75	0	0	0	active	t	\N	2025-12-16 22:11:25.760287+00	2025-12-16 22:11:25.760287+00
a5daaf25-05c2-4703-9575-a87114df9eeb	fix-tests	1.0.0	Fix failing tests in the project	Analyze and fix failing tests in the project:\n\n1. First run the test suite to identify failures\n2. For each failing test:\n   - Understand what the test is checking\n   - Identify why it's failing\n   - Determine if it's a test bug or code bug\n   - Fix appropriately\n\n3. Run tests again to verify fixes\n4. Ensure no regressions were introduced\n\nPrioritize fixing tests that:\n- Block CI/CD\n- Test critical functionality\n- Have clear failure messages	[]	{}	{test,fix,debugging}	test	\N	70	0	0	0	active	t	\N	2025-12-16 22:11:25.760287+00	2025-12-16 22:11:25.760287+00
5f7bb6bd-57e1-45d7-93d1-4d243dd3751b	add-tests	1.0.0	Add comprehensive tests for specified code	Add tests for the specified code or module:\n\n1. **Unit Tests**\n   - Test individual functions\n   - Cover happy path and edge cases\n   - Mock external dependencies\n\n2. **Integration Tests** (if applicable)\n   - Test component interactions\n   - Use test databases/services\n\n3. **Test Patterns**\n   - Arrange-Act-Assert structure\n   - Descriptive test names\n   - Independent test cases\n\n4. **Coverage Goals**\n   - Critical paths: 100%\n   - Overall: aim for 80%+\n   - Edge cases and error handling	[]	{}	{test,coverage,quality}	test	\N	70	0	0	0	active	t	\N	2025-12-16 22:11:25.760287+00	2025-12-16 22:11:25.760287+00
248d0778-c26f-42ff-84db-060745889a8d	refactor	1.0.0	Refactor code for better readability and maintainability	Refactor the specified code to improve:\n\n1. **Readability**\n   - Clear naming\n   - Smaller functions\n   - Consistent style\n\n2. **Maintainability**\n   - Single responsibility\n   - Reduce coupling\n   - Remove duplication\n\n3. **Process**\n   - Keep behavior unchanged\n   - Make incremental changes\n   - Run tests after each change\n   - Document significant changes\n\nFocus on making the code easier to understand and modify without changing its external behavior.	[]	{}	{refactor,clean-code,quality}	code	\N	70	0	0	0	active	t	\N	2025-12-16 22:11:25.760287+00	2025-12-16 22:11:25.760287+00
6ce27ad2-b1df-4c1c-8dbd-0c4bf34edcd6	debug	1.0.0	Debug an issue or error in the codebase	Debug the reported issue:\n\n1. **Understand the Problem**\n   - Reproduce the issue\n   - Gather error messages/logs\n   - Identify expected vs actual behavior\n\n2. **Investigate**\n   - Add logging/debugging\n   - Check recent changes\n   - Review related code\n\n3. **Fix**\n   - Implement minimal fix\n   - Add tests to prevent regression\n   - Verify fix works\n\n4. **Document**\n   - Explain root cause\n   - Note any related issues	[]	{}	{debug,fix,troubleshoot}	code	\N	75	0	0	0	active	t	\N	2025-12-16 22:11:25.760287+00	2025-12-16 22:11:25.760287+00
\.


--
-- Data for Name: feedback; Type: TABLE DATA; Schema: public; Owner: mcp
--

COPY public.feedback (id, agent_id, session_id, client_type, rating, task_success, task_type, feedback_text, interaction_duration_ms, tokens_used, created_at) FROM stdin;
\.


--
-- Data for Name: governance_actions; Type: TABLE DATA; Schema: public; Owner: mcp
--

COPY public.governance_actions (id, agent_id, report_id, action_type, action_by, reason, previous_status, previous_reputation, created_at) FROM stdin;
\.


--
-- Data for Name: reports; Type: TABLE DATA; Schema: public; Owner: mcp
--

COPY public.reports (id, agent_id, reported_by, report_type, severity, description, evidence, status, reviewed_by, resolution, resolution_note, created_at, resolved_at) FROM stdin;
\.


--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: mcp
--

COPY public.schema_migrations (version, applied_at) FROM stdin;
001_initial.sql	2025-12-16 22:11:25.661556+00
002_seed_agents_fixed.sql	2025-12-16 22:11:25.671479+00
003_populate_skills.sql	2025-12-16 22:11:25.67731+00
004_skills_commands.sql	2025-12-16 22:11:25.75693+00
005_seed_skills_commands.sql	2025-12-16 22:11:25.7643+00
\.


--
-- Data for Name: skill_feedback; Type: TABLE DATA; Schema: public; Owner: mcp
--

COPY public.skill_feedback (id, skill_id, rating, task_success, feedback_text, created_at) FROM stdin;
\.


--
-- Data for Name: skill_requests; Type: TABLE DATA; Schema: public; Owner: mcp
--

COPY public.skill_requests (id, skills_hash, skills, agent_id, request_count, created_at, last_requested) FROM stdin;
\.


--
-- Data for Name: skills; Type: TABLE DATA; Schema: public; Owner: mcp
--

COPY public.skills (id, name, version, description, category, content, examples, metadata, tags, embedding, reputation_score, usage_count, feedback_count, avg_rating, status, is_system, created_by, created_at, updated_at) FROM stdin;
c62aeb40-7fc9-456a-92ff-8bef9991af65	kubectl	1.0.0	Kubernetes command-line tool for cluster management, pod operations, and debugging	devops	# kubectl - Kubernetes CLI Reference\n\n## Common Commands\n\n### Cluster Info\n```bash\nkubectl cluster-info                    # Display cluster info\nkubectl get nodes                       # List all nodes\nkubectl describe node <node-name>       # Show node details\n```\n\n### Pod Operations\n```bash\nkubectl get pods                        # List pods in current namespace\nkubectl get pods -A                     # List all pods in all namespaces\nkubectl get pods -o wide               # Show pod IPs and nodes\nkubectl describe pod <pod-name>        # Show pod details\nkubectl logs <pod-name>                # View pod logs\nkubectl logs -f <pod-name>             # Follow pod logs\nkubectl logs <pod-name> -c <container> # Logs from specific container\nkubectl exec -it <pod-name> -- /bin/sh # Shell into pod\n```\n\n### Deployments\n```bash\nkubectl get deployments                 # List deployments\nkubectl describe deployment <name>     # Deployment details\nkubectl scale deployment <name> --replicas=3  # Scale deployment\nkubectl rollout status deployment/<name>      # Check rollout status\nkubectl rollout restart deployment/<name>     # Restart deployment\nkubectl rollout undo deployment/<name>        # Rollback deployment\n```\n\n### Services & Networking\n```bash\nkubectl get services                   # List services\nkubectl get ingress                    # List ingress resources\nkubectl port-forward svc/<name> 8080:80  # Port forward to service\nkubectl port-forward pod/<name> 8080:80  # Port forward to pod\n```\n\n### Debugging\n```bash\nkubectl get events --sort-by=.metadata.creationTimestamp  # Recent events\nkubectl top pods                       # Pod resource usage\nkubectl top nodes                      # Node resource usage\nkubectl debug pod/<name> -it --image=busybox  # Debug pod\n```\n\n### Config & Context\n```bash\nkubectl config get-contexts            # List contexts\nkubectl config use-context <name>      # Switch context\nkubectl config current-context         # Show current context\n```\n\n### Apply & Delete\n```bash\nkubectl apply -f <file.yaml>           # Apply configuration\nkubectl delete -f <file.yaml>          # Delete resources\nkubectl delete pod <name>              # Delete specific pod\nkubectl delete pod <name> --grace-period=0 --force  # Force delete\n```	[]	{}	{kubernetes,k8s,container,orchestration,devops,cli}	\N	80	0	0	0	active	t	\N	2025-12-16 22:11:25.760287+00	2025-12-16 22:11:25.760287+00
d4082b6b-c808-489a-9514-a2a92b6b508b	docker-cli	1.0.0	Docker command-line tool for container management, images, and networking	devops	# Docker CLI Reference\n\n## Container Operations\n```bash\ndocker ps                              # List running containers\ndocker ps -a                           # List all containers\ndocker run -d --name <name> <image>    # Run container detached\ndocker run -it <image> /bin/sh         # Run interactive shell\ndocker run -p 8080:80 <image>          # Run with port mapping\ndocker run -v /host:/container <image> # Run with volume mount\ndocker exec -it <container> /bin/sh    # Shell into container\ndocker logs <container>                # View container logs\ndocker logs -f <container>             # Follow logs\ndocker stop <container>                # Stop container\ndocker start <container>               # Start container\ndocker restart <container>             # Restart container\ndocker rm <container>                  # Remove container\ndocker rm -f <container>               # Force remove\n```\n\n## Image Operations\n```bash\ndocker images                          # List images\ndocker pull <image>                    # Pull image\ndocker build -t <tag> .                # Build image\ndocker build -t <tag> -f Dockerfile.custom .  # Build with specific file\ndocker push <image>                    # Push to registry\ndocker tag <image> <new-tag>          # Tag image\ndocker rmi <image>                    # Remove image\ndocker image prune                    # Remove unused images\n```\n\n## Docker Compose\n```bash\ndocker compose up                      # Start services\ndocker compose up -d                   # Start detached\ndocker compose up --build              # Rebuild and start\ndocker compose down                    # Stop and remove\ndocker compose logs -f                 # Follow all logs\ndocker compose ps                      # List services\ndocker compose exec <service> sh       # Shell into service\n```\n\n## Networking\n```bash\ndocker network ls                      # List networks\ndocker network create <name>           # Create network\ndocker network inspect <name>          # Network details\ndocker network connect <net> <container>  # Connect container\n```\n\n## Volumes\n```bash\ndocker volume ls                       # List volumes\ndocker volume create <name>            # Create volume\ndocker volume inspect <name>           # Volume details\ndocker volume prune                    # Remove unused volumes\n```\n\n## Cleanup\n```bash\ndocker system prune                    # Remove all unused data\ndocker system prune -a                 # Remove all unused + images\ndocker system df                       # Show disk usage\n```	[]	{}	{docker,container,devops,cli,containerization}	\N	80	0	0	0	active	t	\N	2025-12-16 22:11:25.760287+00	2025-12-16 22:11:25.760287+00
662a8da6-2b85-49df-971c-799df9564867	curl	1.0.0	Command-line tool for transferring data with URLs, HTTP requests, and API testing	cli	# curl - HTTP Client Reference\n\n## Basic Requests\n```bash\ncurl https://api.example.com           # GET request\ncurl -X POST https://api.example.com   # POST request\ncurl -X PUT https://api.example.com    # PUT request\ncurl -X DELETE https://api.example.com # DELETE request\ncurl -X PATCH https://api.example.com  # PATCH request\n```\n\n## Headers\n```bash\ncurl -H "Content-Type: application/json" <url>\ncurl -H "Authorization: Bearer <token>" <url>\ncurl -H "X-Custom-Header: value" <url>\n```\n\n## Data/Body\n```bash\n# JSON body\ncurl -X POST -H "Content-Type: application/json" \\\n  -d '{"key": "value"}' <url>\n\n# Form data\ncurl -X POST -d "name=value&other=data" <url>\n\n# File upload\ncurl -X POST -F "file=@/path/to/file" <url>\n\n# From file\ncurl -X POST -d @data.json <url>\n```\n\n## Authentication\n```bash\ncurl -u username:password <url>        # Basic auth\ncurl -H "Authorization: Bearer <token>" <url>  # Bearer token\ncurl --oauth2-bearer <token> <url>     # OAuth2\n```\n\n## Output Options\n```bash\ncurl -o output.json <url>              # Save to file\ncurl -O <url>                          # Save with original name\ncurl -s <url>                          # Silent mode\ncurl -v <url>                          # Verbose output\ncurl -i <url>                          # Include headers in output\ncurl -w "%{http_code}" <url>          # Show status code\n```\n\n## Common Patterns\n```bash\n# GET JSON and parse with jq\ncurl -s <url> | jq .\n\n# POST JSON and check status\ncurl -s -w "%{http_code}" -o /dev/null -X POST \\\n  -H "Content-Type: application/json" \\\n  -d '{"data": "value"}' <url>\n\n# Download with progress\ncurl -# -O <url>\n\n# Follow redirects\ncurl -L <url>\n\n# With timeout\ncurl --connect-timeout 5 --max-time 10 <url>\n\n# Retry on failure\ncurl --retry 3 --retry-delay 2 <url>\n```	[]	{}	{curl,http,api,cli,rest,testing}	\N	75	0	0	0	active	t	\N	2025-12-16 22:11:25.760287+00	2025-12-16 22:11:25.760287+00
47d4c737-336e-47ec-ae40-fb5d82fe4dcf	git	1.0.0	Version control system for tracking changes, branching, and collaboration	devops	# Git Reference\n\n## Basic Commands\n```bash\ngit init                               # Initialize repository\ngit clone <url>                        # Clone repository\ngit status                             # Check status\ngit add .                              # Stage all changes\ngit add <file>                         # Stage specific file\ngit commit -m "message"                # Commit with message\ngit push                               # Push to remote\ngit pull                               # Pull from remote\ngit fetch                              # Fetch without merge\n```\n\n## Branching\n```bash\ngit branch                             # List branches\ngit branch <name>                      # Create branch\ngit checkout <branch>                  # Switch branch\ngit checkout -b <name>                 # Create and switch\ngit merge <branch>                     # Merge branch\ngit branch -d <name>                   # Delete branch\ngit branch -D <name>                   # Force delete\n```\n\n## History & Diff\n```bash\ngit log                                # View history\ngit log --oneline                      # Compact history\ngit log --graph                        # Graph view\ngit diff                               # Unstaged changes\ngit diff --staged                      # Staged changes\ngit diff <branch1>..<branch2>          # Compare branches\ngit show <commit>                      # Show commit details\n```\n\n## Undoing Changes\n```bash\ngit restore <file>                     # Discard changes\ngit restore --staged <file>            # Unstage file\ngit reset HEAD~1                       # Undo last commit (keep changes)\ngit reset --hard HEAD~1                # Undo last commit (discard)\ngit revert <commit>                    # Create reverting commit\ngit stash                              # Stash changes\ngit stash pop                          # Apply and remove stash\n```\n\n## Remote Operations\n```bash\ngit remote -v                          # List remotes\ngit remote add origin <url>            # Add remote\ngit push -u origin <branch>            # Push and set upstream\ngit push --force-with-lease           # Safe force push\n```\n\n## Rebase\n```bash\ngit rebase <branch>                    # Rebase onto branch\ngit rebase -i HEAD~3                   # Interactive rebase\ngit rebase --continue                  # Continue after conflict\ngit rebase --abort                     # Abort rebase\n```	[]	{}	{git,version-control,devops,cli,collaboration}	\N	80	0	0	0	active	t	\N	2025-12-16 22:11:25.760287+00	2025-12-16 22:11:25.760287+00
1596a4f5-e7b5-410b-babf-92bd6f7fa8ae	jq	1.0.0	Command-line JSON processor for parsing, filtering, and transforming JSON data	cli	# jq - JSON Processor Reference\n\n## Basic Usage\n```bash\necho '{"name": "test"}' | jq .        # Pretty print\necho '{"name": "test"}' | jq .name    # Get field\ncat file.json | jq .                   # Parse file\ncurl -s <url> | jq .                   # Parse API response\n```\n\n## Selection\n```bash\njq .key                                # Get object key\njq .[0]                                # Get array element\njq .[]                                 # Iterate array\njq .key1.key2                          # Nested access\njq .key?                               # Optional (no error if missing)\n```\n\n## Filters\n```bash\njq 'select(.age > 30)'                # Filter by condition\njq 'select(.name == "test")'          # Filter by value\njq 'select(.tags | contains(["a"]))'  # Filter by array contains\n```\n\n## Transformation\n```bash\njq '{name, age}'                       # Select specific fields\njq '{newName: .name}'                  # Rename fields\njq '. + {newField: "value"}'           # Add field\njq 'del(.field)'                       # Remove field\njq '.[] | {name, id}'                  # Transform array elements\n```\n\n## Array Operations\n```bash\njq 'length'                            # Array length\njq 'first'                             # First element\njq 'last'                              # Last element\njq 'reverse'                           # Reverse array\njq 'sort'                              # Sort array\njq 'sort_by(.field)'                   # Sort by field\njq 'unique'                            # Remove duplicates\njq 'group_by(.field)'                  # Group by field\njq 'map(.field)'                       # Map to field values\njq '[.[] | select(.x)]'                # Filter and collect\n```\n\n## Output Formats\n```bash\njq -r .name                            # Raw output (no quotes)\njq -c .                                # Compact output\njq -S .                                # Sort keys\njq --tab .                             # Tab indentation\n```\n\n## Common Patterns\n```bash\n# Extract array of names\njq -r '.[].name'\n\n# Create CSV-like output\njq -r '.[] | [.name, .id] | @csv'\n\n# Count items\njq '[.[] | select(.active)] | length'\n\n# Merge objects\njq -s 'add' file1.json file2.json\n```	[]	{}	{jq,json,cli,parsing,data}	\N	70	0	0	0	active	t	\N	2025-12-16 22:11:25.760287+00	2025-12-16 22:11:25.760287+00
b6bab24a-3e7e-42b8-981f-6d7c575bee44	aws-cli	1.0.0	AWS Command Line Interface for managing AWS services	cloud	# AWS CLI Reference\n\n## Configuration\n```bash\naws configure                          # Interactive setup\naws configure list                     # Show config\naws sts get-caller-identity           # Check current identity\n```\n\n## S3\n```bash\naws s3 ls                              # List buckets\naws s3 ls s3://bucket/                 # List bucket contents\naws s3 cp file.txt s3://bucket/       # Upload file\naws s3 cp s3://bucket/file.txt .      # Download file\naws s3 sync ./dir s3://bucket/dir     # Sync directory\naws s3 rm s3://bucket/file.txt        # Delete file\naws s3 rm s3://bucket/ --recursive    # Delete all\naws s3 mb s3://new-bucket             # Create bucket\naws s3 rb s3://bucket                 # Delete bucket\n```\n\n## EC2\n```bash\naws ec2 describe-instances            # List instances\naws ec2 start-instances --instance-ids i-xxx\naws ec2 stop-instances --instance-ids i-xxx\naws ec2 terminate-instances --instance-ids i-xxx\naws ec2 describe-security-groups\naws ec2 describe-vpcs\n```\n\n## Lambda\n```bash\naws lambda list-functions\naws lambda invoke --function-name <name> output.json\naws lambda update-function-code --function-name <name> --zip-file fileb://code.zip\naws lambda get-function --function-name <name>\n```\n\n## ECS\n```bash\naws ecs list-clusters\naws ecs list-services --cluster <name>\naws ecs describe-services --cluster <name> --services <svc>\naws ecs update-service --cluster <name> --service <svc> --force-new-deployment\n```\n\n## CloudWatch\n```bash\naws logs describe-log-groups\naws logs get-log-events --log-group-name <group> --log-stream-name <stream>\naws logs tail <log-group> --follow\n```\n\n## IAM\n```bash\naws iam list-users\naws iam list-roles\naws iam get-user\naws iam list-attached-user-policies --user-name <name>\n```	[]	{}	{aws,cloud,cli,devops,infrastructure}	\N	75	0	0	0	active	t	\N	2025-12-16 22:11:25.760287+00	2025-12-16 22:11:25.760287+00
21fe4fad-91c5-4bf4-aa83-4f8c4b650711	jenkins-ci	1.0.0	Jenkins CI/CD API for querying builds, checking status, fetching logs, and analyzing pipeline stages	devops	# Jenkins CI/CD Query Skill\n\nQuery Jenkins builds and check CI status via the Jenkins API.\n\n## CRITICAL: Environment Variables Don't Work in curl\n\n**NEVER** use environment variable syntax like `$JENKINS_USER` or `${JENKINS_API_TOKEN}` directly in curl commands.\n\n### Step 1: Get the actual credential values first\n```bash\necho "$JENKINS_USER"\necho "$JENKINS_API_TOKEN"\n```\n\n### Step 2: Use those actual values directly in curl commands\n**ALWAYS** hardcode the actual values (from Step 1) into your curl commands.\n\n## URL Patterns\n\n### For Pull Requests\n```\nhttps://JENKINS_URL/job/{ORG}/job/{REPO}/job/PR-{PR_NUMBER}/{BUILD_NUMBER}/\n```\n\n### Common Endpoints\n- `/consoleText` - Full build log\n- `/api/json` - Build status and metadata\n- `/wfapi/describe` - Pipeline stage details\n\n## Basic Query Structure\n\n```bash\ncurl -s "https://JENKINS_URL/job/org/job/repo/job/PR-123/1/consoleText" \\\n  --user "ACTUAL_USER:ACTUAL_TOKEN"\n```\n\n## Common Queries\n\n### Get Console Output (Last 200 Lines)\n```bash\ncurl -s "https://JENKINS_URL/job/org/job/repo/job/PR-123/1/consoleText" \\\n  --user "ACTUAL_USER:ACTUAL_TOKEN" | tail -200\n```\n\n### Check Build Status\n```bash\ncurl -s "https://JENKINS_URL/job/org/job/repo/job/PR-123/1/api/json" \\\n  --user "ACTUAL_USER:ACTUAL_TOKEN" | python3 -c "\nimport sys, json\ndata = json.load(sys.stdin)\nprint(f'Status: {data.get(\\"result\\", \\"RUNNING\\")}')\nprint(f'Building: {data.get(\\"building\\", False)}')\nprint(f'Duration: {data.get(\\"duration\\", 0) / 1000 / 60:.1f} minutes')\n"\n```\n\n### Find What Failed\n```bash\ncurl -s "URL/consoleText" --user "USER:TOKEN" | grep -B 20 "Failed in branch"\n```\n\n### Get Pipeline Stage Timings\n```bash\ncurl -s "URL/wfapi/describe" --user "USER:TOKEN" | python3 -c "\nimport sys, json\ndata = json.load(sys.stdin)\nfor stage in data.get('stages', []):\n    name = stage.get('name', 'Unknown')\n    status = stage.get('status', 'Unknown')\n    duration = stage.get('durationMillis', 0) / 1000 / 60\n    print(f'{name}: {status} ({duration:.1f} min)')\n"\n```\n\n### Get Latest Build Number\n```bash\ncurl -s "https://JENKINS_URL/job/org/job/repo/job/PR-123/api/json" \\\n  --user "USER:TOKEN" | python3 -c "\nimport sys, json\ndata = json.load(sys.stdin)\nlast_build = data.get('lastBuild', {})\nprint(f'Latest build: {last_build.get(\\"number\\", \\"N/A\\")}')\n"\n```\n\n## URL Conversion\n\nBlue Ocean UI: `https://jenkins/blue/organizations/jenkins/org%2Frepo/detail/PR-123/1/pipeline`\nConvert to API: `https://jenkins/job/org/job/repo/job/PR-123/1/consoleText`\n\n## Best Practices\n\n1. Always hardcode credentials in curl commands (don't use env vars directly)\n2. Use `tail -200` for manageable output from large logs\n3. Use `grep -B/-A` with context to find relevant sections\n4. Use `/wfapi/describe` for stage-level timing analysis\n5. Check build status before fetching full logs	null	{}	{jenkins,ci-cd,devops,builds,pipelines,automation}	[-0.0049275397,0.048988912,-0.047414284,0.0011762561,-0.0847403,-0.056164782,-0.056885377,0.020771021,0.042127497,0.041010603,-0.035664223,-0.13714048,0.02167695,-0.024493115,0.03608948,0.0031849192,-0.0127255935,0.04903526,-0.034111604,-0.12313502,0.007282623,0.018971326,0.0030314617,-0.007253853,-0.00044210578,-0.034916077,0.0057240194,0.016509622,-0.026783332,0.05230807,-0.027873458,-0.026243253,-0.033929143,0.004786443,0.048858672,0.063032284,0.053582426,-0.04897621,0.07482114,-0.04092478,0.010413712,-0.0046117483,0.015207193,-0.058078326,-0.01893677,0.005498476,-0.081069395,0.0020215944,-0.102538064,0.0155337,-0.014297232,-0.076622084,0.051211517,-0.004396531,-0.0039005848,-0.0033011439,-0.00966835,0.095993005,0.057841934,-0.064189404,-0.028713701,-0.04480452,0.023004944,0.0196859,-0.09960316,-0.043987498,-0.073939465,0.020986032,0.018235113,-0.005181476,-0.04328477,-0.031542987,-0.07139407,-0.13938153,0.09425423,0.085479386,0.008991805,0.0352524,-0.058795966,-0.070227385,-0.019581936,-0.0073854015,-0.07714093,0.06334282,-0.035345156,0.093763255,0.07833038,0.0627966,0.019038083,0.018287824,-0.0035471965,-0.01471812,-0.110525265,-0.025536202,0.06849281,0.08069726,-0.07762705,-0.045471966,0.009991594,-0.0184247,0.0313051,-0.023597047,0.085283294,-0.023414616,0.0075356085,0.04740423,-0.019664653,-0.0047388393,0.011087058,0.01806375,-0.067574106,-0.036719915,-0.023696333,-0.0045256745,0.072657585,0.081334166,0.058002498,0.012805093,-0.013978129,0.072098285,0.088577606,-0.087342836,-0.05764296,-0.0045833145,0.009615455,0.015075905,0.057586554,2.5695987e-33,0.084731154,0.02175201,0.03554729,0.011014324,0.03072974,0.018610518,0.10952991,-0.013188895,-0.01499596,0.022770258,-0.06992797,0.0716051,-0.05352982,0.022512935,-0.079139665,0.0060456046,0.014329671,-0.011445729,-0.0028657217,0.04365657,0.08248134,-0.08982764,-0.067535,0.036896214,0.07777656,-0.055540975,0.008006032,0.055778172,-0.08440889,0.0005003762,0.006882184,0.0040276116,0.031983238,-0.06249256,0.028165206,-0.0070725773,0.0092564225,0.071506895,-0.066481404,0.057903122,-0.019765196,-0.036154658,0.016709132,0.006930498,0.042821392,-0.09083032,-0.07342387,-0.10490457,0.10004084,0.022514198,0.023063393,-2.5345676e-05,0.06323637,-0.0904545,0.08217754,-0.040089156,-0.01788523,-0.028077526,-0.10598797,-0.08769982,-0.08739557,0.06056213,-0.033830993,-0.032109555,0.0023857583,-0.009428913,-0.0056412583,0.089545004,0.043972597,0.09628654,-0.002463756,0.05835313,-0.000889107,-0.06678114,0.017875206,-0.04729058,0.048199695,0.030602584,0.07166533,0.09711891,0.06887205,0.0017166233,-0.09030082,-0.0024745096,0.0357673,0.013830221,-0.037764005,0.031671636,-0.025743991,-0.016820513,0.023223227,0.003159371,-0.084232494,-0.007173545,-0.054883573,-5.830961e-33,0.07460934,0.0013818108,0.07620327,0.034049027,-0.0077631255,-0.023742117,0.030697312,-0.05305007,0.04356116,-0.08020584,-0.06439398,0.027845882,-0.0589785,0.036515355,0.025962686,-0.030853763,-0.11260599,-0.047886893,0.114649124,0.026902506,-0.02220271,0.00019788313,-0.05350627,0.06968856,-0.009061535,-0.03319681,0.004205621,-0.021807943,-0.0034838354,0.036865894,0.0255073,-0.04529106,-0.078588836,0.1417356,-0.050725505,-0.097586714,-0.013389384,0.12507592,-0.025730059,-0.021103818,0.015400097,-0.040229354,0.008966192,-0.048236962,0.026055157,-0.005617642,0.072962224,-0.0217722,-0.13798943,0.044526286,0.02975977,0.037379555,-0.0892406,0.09193188,0.00034365887,-0.01847823,-0.08032533,0.01895395,-0.09484437,0.045638293,-0.0013414312,0.009579725,0.026779916,0.103315,-0.10594234,-0.058841754,-0.015108914,0.007292985,-0.09856003,-0.0058626076,0.009011585,0.0018945523,0.016367724,-0.040255744,0.02729632,-0.023389261,-0.08308805,-0.004987153,-0.007002434,0.16579342,-0.0085040005,0.052136872,-0.08267302,0.036443923,-0.034942377,0.04216404,-0.030523373,0.027640989,0.038475327,-0.035453692,0.011960781,0.00026291408,-0.02337155,-0.061267883,-0.027141225,-4.7812744e-08,0.05591109,0.022371365,-0.030306619,0.021066437,-0.047715366,0.010371441,-0.0074171536,-0.037762202,-0.040159125,0.029759483,0.0044106683,-0.004611946,0.017835658,0.045423023,-0.033142827,0.015633767,-0.0044209184,-0.011124031,0.052618165,-0.033391833,-0.027814826,-0.0037281658,-0.022694657,-0.023306191,0.028964544,0.019779326,-0.004564008,0.018606007,-0.12472594,0.0052447943,-0.020874342,-0.057298657,-0.037276838,-0.034918226,0.060125772,-0.01689034,0.030111197,-0.022316823,-0.017095424,-0.012539617,0.03073715,0.07570097,-0.0016841812,-0.028871896,-0.012340409,0.005020948,0.094890386,0.037402757,0.003947729,-0.062249415,-0.010926287,-0.037079867,-0.014796669,0.033789255,0.022913003,0.03527739,0.06302942,0.018387148,-0.0018916773,0.004844492,0.035932958,-0.02331575,0.041832987,0.037989803]	50	0	0	0	active	f	api	2025-12-16 22:57:26.886185+00	2025-12-16 22:57:26.886186+00
8507990c-6f68-443e-9f2c-2013b0766d07	datadog-logs	1.0.0	Datadog Logs API for querying application logs, filtering by release/service/status, and debugging production issues	devops	# Datadog Logs Query Skill\n\nQuery Datadog logs using their Logs Search API v2.\n\n## CRITICAL: Environment Variables Don't Work in curl\n\n**NEVER** use environment variable syntax like `$DD_API_KEY` directly in curl commands.\n\n### Step 1: Get the actual credential values first\n```bash\necho "$DD_API_KEY"\necho "$DD_APP_KEY"\n```\n\n### Step 2: Use those actual values directly in curl commands\n\n## API Endpoint\n\n```\nhttps://api.datadoghq.com/api/v2/logs/events/search\n```\n\n## Basic Query Structure\n\n```bash\ncurl -X POST "https://api.datadoghq.com/api/v2/logs/events/search" \\\n  -H "DD-API-KEY: YOUR_API_KEY" \\\n  -H "DD-APPLICATION-KEY: YOUR_APP_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{"filter":{"query":"YOUR_QUERY","from":"now-1h","to":"now"},"sort":"desc","page":{"limit":100}}'\n```\n\n## Time Ranges\n\n- Last hour: `"from":"now-1h","to":"now"`\n- Last 24 hours: `"from":"now-1d","to":"now"`\n- Last 7 days: `"from":"now-7d","to":"now"`\n\n## Query Patterns\n\n### By Release/Build\n```bash\ncurl -X POST "URL" -H "DD-API-KEY: KEY" -H "DD-APPLICATION-KEY: KEY" -H "Content-Type: application/json" \\\n  -d '{"filter":{"query":"@release:your-release-id","from":"now-1h","to":"now"},"sort":"desc","page":{"limit":20}}'\n```\n\n### Filter by Status\n```bash\n# Errors only\n-d '{"filter":{"query":"@release:id status:error","from":"now-1h","to":"now"},"sort":"desc","page":{"limit":50}}'\n\n# Warnings and errors\n-d '{"filter":{"query":"@release:id (status:error OR status:warn)","from":"now-1h","to":"now"},"sort":"desc","page":{"limit":50}}'\n```\n\n### By Service\n```bash\n-d '{"filter":{"query":"service:my-service status:error","from":"now-1h","to":"now"},"sort":"desc","page":{"limit":50}}'\n```\n\n## Common Datadog Query Attributes\n\n- `@release` - Release/build identifier\n- `status` - Log level (info, warn, error, debug)\n- `service` - Service name\n- `source` - Log source (python, nginx, etc.)\n- `message` - Log message text\n- `@http.url` - HTTP request URL\n- `@http.status_code` - HTTP status code\n- `@error.kind` - Error type\n- `@error.stack` - Stack trace\n\n## Processing Results with Python\n\n```bash\ncurl -X POST "URL" ... > /tmp/dd_logs.json\n\npython3 << 'EOF'\nimport json\n\nwith open('/tmp/dd_logs.json') as f:\n    data = json.load(f)\n\nfor log in data.get('data', []):\n    attributes = log.get('attributes', {})\n    print(f"[{attributes.get('timestamp')}] [{attributes.get('status')}] {attributes.get('message')}")\nEOF\n```\n\n## Pagination\n\nMax 1000 results per page. Use cursor from response for next page:\n```bash\n-d '{"filter":{...},"page":{"limit":1000,"cursor":"CURSOR_FROM_RESPONSE"}}'\n```\n\n## Best Practices\n\n1. Always hardcode API keys in curl commands\n2. Use single quotes for JSON body to avoid shell escaping\n3. Start with broad queries, then narrow down\n4. Use appropriate time ranges to limit data volume\n5. Save large result sets to files for processing\n6. Use specific attributes (@release, service) instead of full-text search	null	{}	{datadog,logs,monitoring,observability,debugging,api}	[0.03336572,0.03431606,0.018579584,-0.04795462,-0.030734494,-0.072415255,-0.061025985,0.011036081,0.049898498,0.009247601,0.010581672,-0.087797076,0.061311945,-0.019365638,0.09190402,0.07328993,-0.04858303,0.012885523,-0.03543238,-0.09908172,0.027080717,0.09578303,-0.039166443,-0.0025558355,-0.056013774,-0.0378614,-0.024786105,-0.025074326,-0.012980623,0.021769991,0.0100460425,-0.07163126,-0.09015513,0.06745984,0.06343655,0.02620855,0.0966287,0.03074948,-0.0004930896,0.053446632,0.03877963,0.011653011,0.0040703802,-0.13089482,-0.008178207,-0.03301122,-0.07904183,-0.040298436,-0.09748386,-0.010423964,-0.003160458,-0.027800046,-0.008056671,-0.0049652485,-0.008053616,-0.06347877,-0.08900925,0.10779003,0.023030592,-0.066897124,0.024189971,0.023381287,-0.0066961567,0.007068341,-0.098791786,-0.10880647,-0.062357027,0.005836102,0.08888162,-0.0002150776,-0.047565203,-0.026142579,-0.10723552,-0.11138296,0.021102283,0.028164951,0.0017943386,0.0042519043,0.0007335598,-0.049755137,-0.022505937,-0.014837106,-0.033973057,0.13443369,-0.05494761,0.031875815,-0.01064148,-0.009844166,0.0463527,0.0038689068,0.061730325,-0.07922221,-0.035675786,-0.021818576,0.09455168,0.061230272,-0.06764702,-0.017958803,0.01464854,0.021834984,0.002408835,-0.053489555,-0.058073822,0.014094426,0.045213517,0.018437613,-0.018710725,-0.014401534,-0.010004512,0.08708762,-0.009146873,0.007891122,-0.0074349623,-0.0045702793,0.050344568,0.09532385,-0.0062239124,-0.043796744,-0.053440925,0.035284262,0.096868396,-0.045614757,-0.092158236,0.0076264404,0.040760037,-0.029342247,0.109015785,3.1929946e-33,0.11262735,-0.03874357,0.049143985,-0.027039347,0.024907507,0.059199408,0.08110379,0.014600561,0.023407666,0.059297882,-0.10713659,0.032105055,-0.035251595,-0.069414,-0.055231098,0.033651173,-0.028083866,0.048222538,0.041155268,0.06724559,0.025061198,-0.015776757,0.004408462,0.012062334,0.055109687,-0.015147611,0.01204687,-0.06584803,-0.014398551,0.016373033,0.00091450755,-0.05187834,0.016455647,-0.03672361,0.024962187,-0.029628724,0.0018336553,0.050046068,-0.09219839,0.0519531,0.012897906,0.021303587,-0.06466465,-0.050100464,0.010127312,-0.049974028,-0.07896437,-0.057470437,-0.003063897,-0.03749552,-0.0060462817,0.056953628,0.04359928,-0.13451073,0.09407148,-0.048708413,0.051232036,-0.047454555,-0.046293553,-0.00322493,-0.020670563,0.047342587,0.060454935,-0.08024501,0.08673014,-0.056816638,0.08784985,0.121227816,-0.019208254,0.0065608053,0.029640974,0.017544163,0.038579993,-0.022442618,0.037147373,-0.06604308,0.07116858,0.009735795,0.09578255,0.012794899,0.088166095,-0.06745155,-0.06545179,-0.0051596714,-0.046167392,0.022301294,-0.015929032,-0.01644226,-0.0598204,0.044049043,-0.084683135,0.024237443,-0.032251988,-0.046204254,-0.07931709,-6.64187e-33,-0.002788169,-0.06432776,0.029259693,0.0055634086,0.008112662,-0.0073699607,0.030877074,0.023288514,0.051748704,0.024492336,-0.053653874,0.023812855,-0.014642796,0.035908476,0.0055199987,0.03263752,-0.14160456,0.0065431623,0.023059748,0.017648315,-0.08366328,0.09556907,-0.055373657,0.1036342,-0.054590862,-0.04058824,0.032133047,0.006071408,-0.014354014,0.024577593,0.051241722,-0.0051789535,-0.1007153,0.12213881,-0.10029706,-0.045651358,0.041666575,0.078166164,-0.0052728173,-0.013857266,0.048803598,-0.005996001,-0.010659915,-0.059956595,0.019548362,-0.007724325,0.026196344,0.009357705,0.003801525,-0.046534415,0.02741951,-0.02238142,-0.0021458243,0.107649215,-0.02751515,0.010254104,-0.032843474,0.021260347,-0.1075387,-0.0024053934,-0.010119186,-0.019827802,-0.003427842,0.092235886,-0.069425456,-0.08661168,0.011034474,0.003308786,-0.061187316,-0.082675405,-0.026123736,-0.022989402,0.028871562,-0.0086146435,0.03435597,0.027733238,-0.018502839,-0.064365394,0.039913237,0.12865345,-0.07905743,0.031633437,-0.008763471,0.017472962,0.016508415,0.015942544,-0.048102535,0.079700984,0.035328265,-0.004814091,0.015148614,-0.04131366,-0.050344225,-0.026950365,0.027498048,-4.465017e-08,0.045643542,-0.025489157,-0.040596426,0.064355515,-0.071178794,0.039084066,-0.0144009,0.009730351,-0.0074539063,0.01060809,0.012263488,-0.01698278,-0.033671588,0.036231145,-0.014163472,-0.012623903,-0.04313944,0.00498874,0.043933947,-0.06373165,0.0007863771,0.018528983,0.00016285152,-0.11304818,0.031507272,-0.004785338,-0.02569833,0.026074646,-0.117166735,-0.015759178,0.01136323,0.004357351,-0.006509005,-0.089499064,0.036963243,-0.05006782,0.031779245,-0.020903053,-0.021009209,-0.02162667,0.002445568,0.057121325,0.010298722,-0.01162368,0.0048775887,0.01905554,0.118618004,0.0940241,0.02739076,-0.021358823,-0.021869212,-0.03833786,0.040623013,-0.022914456,0.07110206,0.019171804,-0.0075142374,-0.08394243,-0.0013432046,-0.019470347,0.07465124,-0.014838902,0.020769715,0.03527562]	50	0	0	0	active	f	api	2025-12-16 22:57:37.36725+00	2025-12-16 22:57:37.36725+00
\.


--
-- Name: agents agents_name_key; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.agents
    ADD CONSTRAINT agents_name_key UNIQUE (name);


--
-- Name: agents agents_pkey; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.agents
    ADD CONSTRAINT agents_pkey PRIMARY KEY (id);


--
-- Name: command_feedback command_feedback_pkey; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.command_feedback
    ADD CONSTRAINT command_feedback_pkey PRIMARY KEY (id);


--
-- Name: commands commands_name_key; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.commands
    ADD CONSTRAINT commands_name_key UNIQUE (name);


--
-- Name: commands commands_pkey; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.commands
    ADD CONSTRAINT commands_pkey PRIMARY KEY (id);


--
-- Name: feedback feedback_pkey; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.feedback
    ADD CONSTRAINT feedback_pkey PRIMARY KEY (id);


--
-- Name: governance_actions governance_actions_pkey; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.governance_actions
    ADD CONSTRAINT governance_actions_pkey PRIMARY KEY (id);


--
-- Name: reports reports_pkey; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.reports
    ADD CONSTRAINT reports_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: skill_feedback skill_feedback_pkey; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.skill_feedback
    ADD CONSTRAINT skill_feedback_pkey PRIMARY KEY (id);


--
-- Name: skill_requests skill_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.skill_requests
    ADD CONSTRAINT skill_requests_pkey PRIMARY KEY (id);


--
-- Name: skill_requests skill_requests_skills_hash_key; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.skill_requests
    ADD CONSTRAINT skill_requests_skills_hash_key UNIQUE (skills_hash);


--
-- Name: skills skills_name_key; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.skills
    ADD CONSTRAINT skills_name_key UNIQUE (name);


--
-- Name: skills skills_pkey; Type: CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.skills
    ADD CONSTRAINT skills_pkey PRIMARY KEY (id);


--
-- Name: idx_agents_embedding; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_agents_embedding ON public.agents USING ivfflat (embedding public.vector_cosine_ops) WITH (lists='100');


--
-- Name: idx_agents_name; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_agents_name ON public.agents USING btree (name);


--
-- Name: idx_agents_reputation; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_agents_reputation ON public.agents USING btree (reputation_score DESC);


--
-- Name: idx_agents_skills; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_agents_skills ON public.agents USING gin (skills);


--
-- Name: idx_agents_status; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_agents_status ON public.agents USING btree (status);


--
-- Name: idx_command_feedback_command; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_command_feedback_command ON public.command_feedback USING btree (command_id);


--
-- Name: idx_commands_category; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_commands_category ON public.commands USING btree (category);


--
-- Name: idx_commands_embedding; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_commands_embedding ON public.commands USING ivfflat (embedding public.vector_cosine_ops) WITH (lists='100');


--
-- Name: idx_commands_name; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_commands_name ON public.commands USING btree (name);


--
-- Name: idx_commands_status; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_commands_status ON public.commands USING btree (status);


--
-- Name: idx_commands_tags; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_commands_tags ON public.commands USING gin (tags);


--
-- Name: idx_feedback_agent; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_feedback_agent ON public.feedback USING btree (agent_id);


--
-- Name: idx_feedback_created; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_feedback_created ON public.feedback USING btree (created_at DESC);


--
-- Name: idx_feedback_rating; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_feedback_rating ON public.feedback USING btree (rating);


--
-- Name: idx_governance_action_type; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_governance_action_type ON public.governance_actions USING btree (action_type);


--
-- Name: idx_governance_agent; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_governance_agent ON public.governance_actions USING btree (agent_id);


--
-- Name: idx_governance_created; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_governance_created ON public.governance_actions USING btree (created_at DESC);


--
-- Name: idx_reports_agent; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_reports_agent ON public.reports USING btree (agent_id);


--
-- Name: idx_reports_severity; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_reports_severity ON public.reports USING btree (severity);


--
-- Name: idx_reports_status; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_reports_status ON public.reports USING btree (status);


--
-- Name: idx_skill_feedback_skill; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_skill_feedback_skill ON public.skill_feedback USING btree (skill_id);


--
-- Name: idx_skill_requests_count; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_skill_requests_count ON public.skill_requests USING btree (request_count DESC);


--
-- Name: idx_skill_requests_hash; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_skill_requests_hash ON public.skill_requests USING btree (skills_hash);


--
-- Name: idx_skills_category; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_skills_category ON public.skills USING btree (category);


--
-- Name: idx_skills_embedding; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_skills_embedding ON public.skills USING ivfflat (embedding public.vector_cosine_ops) WITH (lists='100');


--
-- Name: idx_skills_name; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_skills_name ON public.skills USING btree (name);


--
-- Name: idx_skills_status; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_skills_status ON public.skills USING btree (status);


--
-- Name: idx_skills_tags; Type: INDEX; Schema: public; Owner: mcp
--

CREATE INDEX idx_skills_tags ON public.skills USING gin (tags);


--
-- Name: agents agents_updated_at; Type: TRIGGER; Schema: public; Owner: mcp
--

CREATE TRIGGER agents_updated_at BEFORE UPDATE ON public.agents FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();


--
-- Name: commands commands_updated_at; Type: TRIGGER; Schema: public; Owner: mcp
--

CREATE TRIGGER commands_updated_at BEFORE UPDATE ON public.commands FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();


--
-- Name: skills skills_updated_at; Type: TRIGGER; Schema: public; Owner: mcp
--

CREATE TRIGGER skills_updated_at BEFORE UPDATE ON public.skills FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();


--
-- Name: command_feedback command_feedback_command_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.command_feedback
    ADD CONSTRAINT command_feedback_command_id_fkey FOREIGN KEY (command_id) REFERENCES public.commands(id) ON DELETE CASCADE;


--
-- Name: feedback feedback_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.feedback
    ADD CONSTRAINT feedback_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE;


--
-- Name: governance_actions governance_actions_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.governance_actions
    ADD CONSTRAINT governance_actions_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE;


--
-- Name: governance_actions governance_actions_report_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.governance_actions
    ADD CONSTRAINT governance_actions_report_id_fkey FOREIGN KEY (report_id) REFERENCES public.reports(id);


--
-- Name: reports reports_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.reports
    ADD CONSTRAINT reports_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.agents(id) ON DELETE CASCADE;


--
-- Name: skill_feedback skill_feedback_skill_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.skill_feedback
    ADD CONSTRAINT skill_feedback_skill_id_fkey FOREIGN KEY (skill_id) REFERENCES public.skills(id) ON DELETE CASCADE;


--
-- Name: skill_requests skill_requests_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: mcp
--

ALTER TABLE ONLY public.skill_requests
    ADD CONSTRAINT skill_requests_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.agents(id);


--
-- PostgreSQL database dump complete
--

\unrestrict YdmBIxMQBSzdc9vtxKEViygDFoO9RbtQGVqUdNrdnkYszYMBAjRxLRRhOYPQUZd

