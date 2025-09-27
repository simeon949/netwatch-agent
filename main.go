package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/go-routeros/routeros"
    "github.com/melbahja/goph"
    "golang.org/x/crypto/ssh"
)

type Device struct {
    Kind     string `json:"Kind"`
    Host     string `json:"Host"`
    Username string `json:"Username"`
    Password string `json:"Password"`
    Port     int    `json:"Port"`
}

type CommandRequest struct {
    Device  Device `json:"device"`
    Command string `json:"command"`
}

type CommandResponse struct {
    Success bool        `json:"success"`
    Result  interface{} `json:"result,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func main() {
    apiKey := os.Getenv("AGENT_API_KEY")
    if apiKey == "" {
        log.Println("Warning: AGENT_API_KEY not set. API will be unprotected.")
    }

    http.HandleFunc("/ping", handlePing)
    http.HandleFunc("/api/ssh/command", authMiddleware(apiKey, handleSSHCommand))
    http.HandleFunc("/api/ping-host", authMiddleware(apiKey, handlePingHost))
    
    fmt.Println("Starting agent on port 8080...")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func authMiddleware(apiKey string, next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if apiKey != "" {
            providedKey := r.Header.Get("X-API-Key")
            if providedKey != apiKey {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
        }
        next(w, r)
    }
}

func handlePing(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": "ok",
        "timestamp": time.Now().Format(time.RFC3339),
    })
}

func handleSSHCommand(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req CommandRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendError(w, "Invalid JSON: "+err.Error())
        return
    }

    var response CommandResponse

    switch req.Device.Kind {
    case "mikrotik":
        response = executeMikrotikCommand(req.Device, req.Command)
    case "ubiquiti", "ssh":
        response = executeSSHCommand(req.Device, req.Command)
    default:
        sendError(w, "Unsupported device kind: "+req.Device.Kind)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func executeMikrotikCommand(device Device, command string) CommandResponse {
    client, err := routeros.Dial(fmt.Sprintf("%s:%d", device.Host, device.Port), device.Username, device.Password)
    if err != nil {
        return CommandResponse{Success: false, Error: "Connection failed: " + err.Error()}
    }
    defer client.Close()

    reply, err := client.Run(command)
    if err != nil {
        return CommandResponse{Success: false, Error: "Command failed: " + err.Error()}
    }

    return CommandResponse{
        Success: true,
        Result: map[string]interface{}{
            "stdout": reply.String(),
            "command": command,
        },
    }
}

func executeSSHCommand(device Device, command string) CommandResponse {
    config := &ssh.ClientConfig{
        User: device.Username,
        Auth: []ssh.AuthMethod{
            ssh.Password(device.Password),
        },
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),
        Timeout:         10 * time.Second,
    }

    client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", device.Host, device.Port), config)
    if err != nil {
        return CommandResponse{Success: false, Error: "SSH connection failed: " + err.Error()}
    }
    defer client.Close()

    session, err := client.NewSession()
    if err != nil {
        return CommandResponse{Success: false, Error: "SSH session failed: " + err.Error()}
    }
    defer session.Close()

    output, err := session.CombinedOutput(command)
    if err != nil {
        return CommandResponse{Success: false, Error: "Command execution failed: " + err.Error()}
    }

    return CommandResponse{
        Success: true,
        Result: map[string]interface{}{
            "stdout": string(output),
            "command": command,
        },
    }
}

func handlePingHost(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Host string `json:"host"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendError(w, "Invalid JSON")
        return
    }

    // Simple ping implementation would go here
    // For now, return success
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "reachable": true,
        "host": req.Host,
    })
}

func sendError(w http.ResponseWriter, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusBadRequest)
    json.NewEncoder(w).Encode(CommandResponse{Success: false, Error: message})
}
