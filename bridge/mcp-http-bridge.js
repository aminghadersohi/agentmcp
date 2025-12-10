#!/usr/bin/env node
/**
 * MCP HTTP Bridge
 * Exposes the Docker-based MCP server via HTTP/SSE for Claude Desktop
 */

const http = require('http');
const { spawn } = require('child_process');

const PORT = process.env.BRIDGE_PORT || 18888;

class MCPBridge {
  constructor() {
    this.mcpProcess = null;
  }

  startMCPProcess() {
    console.log('[Bridge] Starting MCP process via Docker...');

    this.mcpProcess = spawn('docker', [
      'exec', '-i', 'mcp-serve',
      '/app/mcp-serve', '-transport', 'stdio'
    ]);

    this.mcpProcess.on('error', (err) => {
      console.error('[Bridge] MCP process error:', err);
    });

    this.mcpProcess.stderr.on('data', (data) => {
      // Log stderr but don't crash
      console.error('[MCP stderr]', data.toString());
    });

    console.log('[Bridge] MCP process started');
  }

  async sendRequest(request) {
    return new Promise((resolve, reject) => {
      let response = '';

      const timeout = setTimeout(() => {
        reject(new Error('Request timeout'));
      }, 30000);

      const dataHandler = (data) => {
        const chunk = data.toString();
        response += chunk;

        // Try to parse as JSON to see if we have a complete response
        try {
          const parsed = JSON.parse(response);
          clearTimeout(timeout);
          this.mcpProcess.stdout.removeListener('data', dataHandler);
          resolve(parsed);
        } catch (e) {
          // Not complete yet, keep accumulating
        }
      };

      this.mcpProcess.stdout.on('data', dataHandler);

      // Send request
      this.mcpProcess.stdin.write(JSON.stringify(request) + '\n');
    });
  }

  async handleRequest(req, res) {
    // CORS headers
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

    if (req.method === 'OPTIONS') {
      res.writeHead(200);
      res.end();
      return;
    }

    if (req.method === 'GET' && req.url === '/health') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ status: 'ok', bridge: 'mcp-http-bridge' }));
      return;
    }

    if (req.method === 'POST' && req.url === '/message') {
      let body = '';

      req.on('data', chunk => {
        body += chunk.toString();
      });

      req.on('end', async () => {
        try {
          const request = JSON.parse(body);
          console.log('[Bridge] Request:', request.method);

          const response = await this.sendRequest(request);

          res.writeHead(200, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify(response));
        } catch (error) {
          console.error('[Bridge] Error:', error);
          res.writeHead(500, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({
            jsonrpc: '2.0',
            error: { code: -32603, message: error.message }
          }));
        }
      });
      return;
    }

    // SSE endpoint for streaming
    if (req.method === 'GET' && (req.url === '/sse' || req.url === '/')) {
      res.writeHead(200, {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        'Connection': 'keep-alive',
        'Access-Control-Allow-Origin': '*'
      });

      // Send endpoint info message
      const endpointMessage = JSON.stringify({
        jsonrpc: '2.0',
        method: 'endpoint',
        params: {
          endpoint: `http://localhost:${PORT}/message`
        }
      });
      res.write(`data: ${endpointMessage}\n\n`);

      // Keep connection alive
      const keepAlive = setInterval(() => {
        res.write(': keepalive\n\n');
      }, 15000);

      req.on('close', () => {
        clearInterval(keepAlive);
      });
      return;
    }

    res.writeHead(404);
    res.end('Not Found');
  }

  start() {
    this.startMCPProcess();

    const server = http.createServer((req, res) => {
      this.handleRequest(req, res).catch(err => {
        console.error('[Bridge] Handler error:', err);
        res.writeHead(500);
        res.end('Internal Server Error');
      });
    });

    server.listen(PORT, () => {
      console.log(`\n🌉 MCP HTTP Bridge running at http://localhost:${PORT}`);
      console.log(`   Health check: http://localhost:${PORT}/health`);
      console.log(`   MCP endpoint: http://localhost:${PORT}/message\n`);
    });

    // Handle shutdown
    process.on('SIGINT', () => {
      console.log('\n[Bridge] Shutting down...');
      if (this.mcpProcess) {
        this.mcpProcess.kill();
      }
      process.exit(0);
    });
  }
}

const bridge = new MCPBridge();
bridge.start();
