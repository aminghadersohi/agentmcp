#!/bin/bash
# Start the MCP HTTP Bridge

cd "$(dirname "$0")"

echo "Starting MCP HTTP Bridge..."
node mcp-http-bridge.js
