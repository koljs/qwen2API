#!/system/bin/sh
# qwen2API Magisk service script
# Runs the backend as a background daemon on boot

MODDIR=${0%/*}
PERSIST_DIR=/data/local/qwen2api
BIN="$MODDIR/qwen2api"
PID_FILE="$PERSIST_DIR/qwen2api.pid"
LOG_FILE="$PERSIST_DIR/logs/service.log"

# Ensure persistent directories exist
mkdir -p "$PERSIST_DIR/data"
mkdir -p "$PERSIST_DIR/logs"

# Wait for boot to complete (network may not be ready immediately)
while [ "$(getprop sys.boot_completed)" != "1" ]; do
    sleep 2
done
sleep 5

# Kill any existing instance
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE" 2>/dev/null)
    if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" 2>/dev/null; then
        kill "$OLD_PID" 2>/dev/null
        sleep 1
    fi
    rm -f "$PID_FILE"
fi

# Start the backend
# BASE_DIR points to the module directory (for frontend dist)
# DATA_DIR and LOGS_DIR point to persistent storage
# ADMIN_KEY must be set by the user in the config file below
CONFIG_FILE="$PERSIST_DIR/config.env"
if [ ! -f "$CONFIG_FILE" ]; then
    # Generate a default config on first run
    cat > "$CONFIG_FILE" << 'DEFCONF'
# qwen2API configuration
# Edit this file and reboot (or restart the service) to apply changes.

# Required: set a strong admin key for WebUI and admin API access
ADMIN_KEY=change-me-to-a-strong-key

# Port the HTTP server listens on (default: 7860)
PORT=7860

# Log level: DEBUG, INFO, WARN, ERROR
LOG_LEVEL=INFO

# Optional: inject Qwen upstream accounts at runtime (not saved to data/accounts.json)
# Format: token;optional-email;optional-password
# QWEN_ACCOUNT_1=your-qwen-token;user@example.com;optional-password

# Optional: inject downstream API keys at runtime (not saved to data/api_keys.json)
# QWEN_API_KEY=sk-your-env-key
DEFCONF
    chmod 644 "$CONFIG_FILE"
fi

# Source user config
. "$CONFIG_FILE"

# Export environment variables for the backend
export BASE_DIR="$MODDIR"
export DATA_DIR="$PERSIST_DIR/data"
export LOGS_DIR="$PERSIST_DIR/logs"
export ADMIN_KEY="${ADMIN_KEY:-}"
export PORT="${PORT:-7860}"
export LOG_LEVEL="${LOG_LEVEL:-INFO}"

# Start the daemon
nohup "$BIN" >> "$LOG_FILE" 2>&1 &
echo $! > "$PID_FILE"

echo "[$(date)] qwen2API started (PID=$(cat "$PID_FILE"), PORT=$PORT)" >> "$LOG_FILE"
