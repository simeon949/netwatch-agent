
# NetWatch Agent

A lightweight network monitoring agent for MikroTik, Ubiquiti, and other network devices.

## Quick Start

1. Download the `netwatch-agent.exe` from GitHub Actions
2. Set API key (optional but recommended):
   ```
   set AGENT_API_KEY=YourSecretKey123
   ```
3. Run the agent:
   ```
   netwatch-agent.exe
   ```

The agent will start on port 8080.

## Endpoints

- `GET /ping` - Health check
- `POST /api/ssh/command` - Execute SSH commands
- `POST /api/ping-host` - Ping a host

## Notes

- This Node.js version works for SSH-based device management
- For full MikroTik API support, the Go version is needed (but has dependency issues)
- All major functionality works: device status checks, command execution, system monitoring
