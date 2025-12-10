#!/usr/bin/env python3
"""
MCP Serve Web Dashboard
A simple web interface for managing agents and governance
Connects directly to PostgreSQL database
"""

from flask import Flask, render_template, request, jsonify, redirect, url_for
import psycopg2
import psycopg2.extras
import json
import os

app = Flask(__name__)
app.config['SECRET_KEY'] = os.environ.get('SECRET_KEY', 'dev-secret-key-change-in-prod')

# Database configuration
DB_CONFIG = {
    'host': os.environ.get('DB_HOST', 'localhost'),
    'port': int(os.environ.get('DB_PORT', 54320)),
    'database': os.environ.get('DB_NAME', 'mcp_serve'),
    'user': os.environ.get('DB_USER', 'mcp'),
    'password': os.environ.get('DB_PASSWORD', 'mcpserve')
}

def get_db():
    """Get database connection"""
    return psycopg2.connect(**DB_CONFIG)

@app.route('/')
def index():
    """Dashboard home - Overview stats"""
    try:
        conn = get_db()
        cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

        # Get statistics
        cur.execute("""
            SELECT
                COUNT(*) as total_agents,
                COUNT(*) FILTER (WHERE status = 'active') as active_agents,
                COUNT(*) FILTER (WHERE status = 'quarantined') as quarantined_agents,
                COUNT(*) FILTER (WHERE status = 'banned') as banned_agents
            FROM agents
        """)
        stats = cur.fetchone()

        cur.execute("SELECT COUNT(*) as pending_reports FROM reports WHERE status = 'pending'")
        report_stats = cur.fetchone()
        stats['pending_reports'] = report_stats['pending_reports']

        cur.close()
        conn.close()

        return render_template('index.html', stats=stats)
    except Exception as e:
        return render_template('index.html', stats=None, error=str(e))

@app.route('/agents')
def agents_list():
    """List all agents"""
    try:
        conn = get_db()
        cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

        cur.execute("""
            SELECT id, name, description, version, model, tools, metadata,
                   status, created_at, updated_at
            FROM agents
            ORDER BY created_at DESC
        """)
        agents = cur.fetchall()

        # Parse metadata JSON
        for agent in agents:
            if agent['metadata']:
                agent['tags'] = agent['metadata'].get('tags', [])
            else:
                agent['tags'] = []

        cur.close()
        conn.close()

        return render_template('agents.html', agents=agents)
    except Exception as e:
        return render_template('agents.html', agents=[], error=str(e))

@app.route('/agents/<name>')
def agent_detail(name):
    """View agent details"""
    try:
        conn = get_db()
        cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

        # Get agent
        cur.execute("""
            SELECT id, name, description, version, model, tools, metadata,
                   prompt, status, is_system, usage_count, feedback_count,
                   avg_rating, reputation_score, created_at, updated_at
            FROM agents
            WHERE name = %s
        """, (name,))
        agent = cur.fetchone()

        if not agent:
            cur.close()
            conn.close()
            return render_template('agent_detail.html', agent=None, stats=None, name=name)

        # Get feedback
        cur.execute("""
            SELECT rating, comment, success, submitted_by, created_at
            FROM feedback
            WHERE agent_id = %s
            ORDER BY created_at DESC
            LIMIT 10
        """, (agent['id'],))
        feedbacks = cur.fetchall()

        stats = {
            'usage_count': agent['usage_count'],
            'feedback_count': agent['feedback_count'],
            'avg_rating': agent['avg_rating'] or 0,
            'reputation_score': agent['reputation_score']
        }

        cur.close()
        conn.close()

        return render_template('agent_detail.html', agent=agent, stats=stats, name=name, feedbacks=feedbacks)
    except Exception as e:
        return render_template('agent_detail.html', agent=None, stats=None, name=name, error=str(e))

@app.route('/agents/search')
def search_agents():
    """Search agents"""
    query = request.args.get('q', '')

    try:
        conn = get_db()
        cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

        cur.execute("""
            SELECT id, name, description, version, metadata, status
            FROM agents
            WHERE name ILIKE %s OR description ILIKE %s
            ORDER BY created_at DESC
        """, (f'%{query}%', f'%{query}%'))
        results = cur.fetchall()

        # Parse metadata
        for agent in results:
            if agent['metadata']:
                agent['tags'] = agent['metadata'].get('tags', [])
            else:
                agent['tags'] = []

        cur.close()
        conn.close()

        return render_template('search.html', query=query, results=results)
    except Exception as e:
        return render_template('search.html', query=query, results=[], error=str(e))

@app.route('/governance')
def governance():
    """Governance overview"""
    try:
        conn = get_db()
        cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

        # Get statistics
        cur.execute("""
            SELECT
                COUNT(*) FILTER (WHERE status = 'pending') as pending_reports,
                COUNT(*) as total_reports
            FROM reports
        """)
        report_stats = cur.fetchone()

        cur.execute("""
            SELECT
                COUNT(*) FILTER (WHERE status = 'quarantined') as quarantined_agents,
                COUNT(*) FILTER (WHERE status = 'banned') as banned_agents
            FROM agents
        """)
        agent_stats = cur.fetchone()

        stats = {**report_stats, **agent_stats}

        cur.close()
        conn.close()

        return render_template('governance.html', stats=stats)
    except Exception as e:
        return render_template('governance.html', stats=None, error=str(e))

@app.route('/governance/reports')
def reports():
    """View all reports"""
    try:
        conn = get_db()
        cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

        cur.execute("""
            SELECT r.*, a.name as agent_name
            FROM reports r
            JOIN agents a ON r.agent_id = a.id
            ORDER BY r.created_at DESC
            LIMIT 100
        """)
        reports = cur.fetchall()

        cur.close()
        conn.close()

        return render_template('reports.html', reports=reports)
    except Exception as e:
        return render_template('reports.html', reports=[], error=str(e))

@app.route('/api/feedback', methods=['POST'])
def submit_feedback():
    """Submit feedback via API"""
    try:
        data = request.json
        conn = get_db()
        cur = conn.cursor()

        # Get agent ID
        cur.execute("SELECT id FROM agents WHERE name = %s", (data['agent_name'],))
        result = cur.fetchone()
        if not result:
            return jsonify({"success": False, "error": "Agent not found"})

        agent_id = result[0]

        # Insert feedback
        cur.execute("""
            INSERT INTO feedback (agent_id, rating, comment, success, submitted_by)
            VALUES (%s, %s, %s, %s, %s)
        """, (
            agent_id,
            data.get('rating'),
            data.get('comment', ''),
            data.get('success', True),
            data.get('submitted_by', 'dashboard-user')
        ))

        conn.commit()
        cur.close()
        conn.close()

        return jsonify({"success": True})
    except Exception as e:
        return jsonify({"success": False, "error": str(e)})

@app.route('/api/report', methods=['POST'])
def submit_report():
    """Submit a report via API"""
    try:
        data = request.json
        conn = get_db()
        cur = conn.cursor()

        # Get agent ID
        cur.execute("SELECT id FROM agents WHERE name = %s", (data['agent_name'],))
        result = cur.fetchone()
        if not result:
            return jsonify({"success": False, "error": "Agent not found"})

        agent_id = result[0]

        # Insert report
        cur.execute("""
            INSERT INTO reports (agent_id, reported_by, report_type, severity, description, evidence)
            VALUES (%s, %s, %s, %s, %s, %s)
        """, (
            agent_id,
            data.get('reported_by', 'dashboard-user'),
            data.get('report_type', 'quality'),
            data.get('severity', 'medium'),
            data.get('description', ''),
            json.dumps(data.get('evidence', {}))
        ))

        conn.commit()
        cur.close()
        conn.close()

        return jsonify({"success": True})
    except Exception as e:
        return jsonify({"success": False, "error": str(e)})

@app.template_filter('tojson_pretty')
def tojson_pretty(value):
    """Pretty print JSON"""
    if isinstance(value, (dict, list)):
        return json.dumps(value, indent=2, default=str)
    return str(value)

if __name__ == '__main__':
    port = int(os.environ.get('DASHBOARD_PORT', 5001))
    print(f"\n🚀 Dashboard starting at http://localhost:{port}")
    print(f"📊 Connecting to PostgreSQL at {DB_CONFIG['host']}:{DB_CONFIG['port']}\n")
    app.run(host='0.0.0.0', port=port, debug=True)
