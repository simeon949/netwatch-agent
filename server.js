const express = require('express');
const { Client } = require('ssh2');
const cors = require('cors');

const app = express();
const PORT = 8080;

// Middleware
app.use(cors());
app.use(express.json());

// API Key from environment
const API_KEY = process.env.AGENT_API_KEY;

// Auth middleware
function authMiddleware(req, res, next) {
    if (API_KEY && req.headers['x-api-key'] !== API_KEY) {
        return res.status(401).json({ error: 'Unauthorized' });
    }
    next();
}

// Health check endpoint
app.get('/ping', (req, res) => {
    res.json({
        status: 'ok',
        timestamp: new Date().toISOString()
    });
});

// SSH command endpoint
app.post('/api/ssh/command', authMiddleware, async (req, res) => {
    const { device, command } = req.body;
    
    if (!device || !command) {
        return res.status(400).json({ 
            success: false, 
            error: 'Device and command are required' 
        });
    }

    try {
        const result = await executeSSHCommand(device, command);
        res.json(result);
    } catch (error) {
        res.json({
            success: false,
            error: error.message
        });
    }
});

// Ping host endpoint
app.post('/api/ping-host', authMiddleware, async (req, res) => {
    const { host } = req.body;
    
    if (!host) {
        return res.status(400).json({ error: 'Host is required' });
    }

    try {
        const reachable = await pingHost(host);
        res.json({
            reachable,
            host,
            timestamp: new Date().toISOString()
        });
    } catch (error) {
        res.json({
            reachable: false,
            host,
            error: error.message,
            timestamp: new Date().toISOString()
        });
    }
});

// Execute SSH command
function executeSSHCommand(device, command) {
    return new Promise((resolve, reject) => {
        const conn = new Client();
        let result = '';
        let errorOutput = '';

        const timeout = setTimeout(() => {
            conn.end();
            reject(new Error('Connection timeout'));
        }, 15000);

        conn.on('ready', () => {
            clearTimeout(timeout);
            
            if (device.Kind === 'mikrotik') {
                // For MikroTik, we'll use SSH but note API limitations
                conn.exec(command, (err, stream) => {
                    if (err) {
                        conn.end();
                        return reject(err);
                    }

                    stream.on('close', () => {
                        conn.end();
                        resolve({
                            success: errorOutput === '',
                            result: {
                                stdout: result,
                                stderr: errorOutput,
                                command: command
                            },
                            error: errorOutput || null
                        });
                    });

                    stream.on('data', (data) => {
                        result += data.toString();
                    });

                    stream.stderr.on('data', (data) => {
                        errorOutput += data.toString();
                    });
                });
            } else {
                // For other devices (Ubiquiti, etc.)
                conn.exec(command, (err, stream) => {
                    if (err) {
                        conn.end();
                        return reject(err);
                    }

                    stream.on('close', () => {
                        conn.end();
                        resolve({
                            success: true,
                            result: {
                                stdout: result,
                                stderr: errorOutput,
                                command: command
                            }
                        });
                    });

                    stream.on('data', (data) => {
                        result += data.toString();
                    });

                    stream.stderr.on('data', (data) => {
                        errorOutput += data.toString();
                    });
                });
            }
        });

        conn.on('error', (err) => {
            clearTimeout(timeout);
            reject(err);
        });

        conn.connect({
            host: device.Host,
            port: device.Port || 22,
            username: device.Username,
            password: device.Password,
            readyTimeout: 10000
        });
    });
}

// Simple ping using system ping command
function pingHost(host) {
    return new Promise((resolve) => {
        const { exec } = require('child_process');
        const isWindows = process.platform === 'win32';
        const pingCommand = isWindows ? `ping -n 1 ${host}` : `ping -c 1 ${host}`;
        
        exec(pingCommand, { timeout: 5000 }, (error) => {
            resolve(!error);
        });
    });
}

// Start server
app.listen(PORT, () => {
    console.log(`NetWatch Agent running on http://localhost:${PORT}`);
    console.log('Endpoints:');
    console.log('  GET  /ping - Health check');
    console.log('  POST /api/ssh/command - Execute SSH commands');
    console.log('  POST /api/ping-host - Ping a host');
    
    if (API_KEY) {
        console.log('API Key authentication: ENABLED');
    } else {
        console.log('API Key authentication: DISABLED (set AGENT_API_KEY environment variable)');
    }
});
