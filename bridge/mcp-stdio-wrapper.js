#!/usr/bin/env node
/**
 * MCP stdio wrapper - Simple passthrough to Docker container
 * This accepts stdio input and forwards it to the Docker container
 */

const { spawn } = require('child_process');

// Find docker executable (handles multiple Docker installation locations)
const fs = require('fs');
const possiblePaths = [
  '/Applications/Docker.app/Contents/Resources/bin/docker',
  '/usr/local/bin/docker',
  '/opt/homebrew/bin/docker'
];
const dockerPath = possiblePaths.find(p => fs.existsSync(p)) || 'docker';

// Spawn the Docker exec process
const mcpProcess = spawn(dockerPath, [
  'exec', '-i', 'agentmcp',
  '/app/agentmcp', '-transport', 'stdio'
], {
  stdio: ['pipe', 'pipe', 'ignore']  // stdin piped, stdout piped, stderr ignored
});

// Pipe stdin to Docker process
process.stdin.pipe(mcpProcess.stdin);

// Pipe Docker output to stdout
mcpProcess.stdout.pipe(process.stdout);

// Handle errors
mcpProcess.on('error', (err) => {
  console.error('MCP process error:', err);
  process.exit(1);
});

mcpProcess.on('exit', (code) => {
  process.exit(code || 0);
});

// Forward signals
process.on('SIGINT', () => mcpProcess.kill('SIGINT'));
process.on('SIGTERM', () => mcpProcess.kill('SIGTERM'));
