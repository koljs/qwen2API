#!/usr/bin/env bash
# Build qwen2API as a Magisk module zip
# Prerequisites: Go 1.26+, Node.js 20+, npm, zip
#
# Usage:
#   ./build-magisk.sh            # build for arm64 (default)
#   ./build-magisk.sh arm64      # same as above
#   ./build-magisk.sh arm        # 32-bit ARM
#   ./build-magisk.sh all        # build both architectures

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_DIR="$PROJECT_DIR/build/magisk"

# Target architecture
ARCH="${1:-arm64}"

# Map arch to Go GOARCH
case "$ARCH" in
    arm64) GOARCH=arm64 ;;
    arm)   GOARCH=arm ;;
    all)   GOARCH="arm64 arm" ;;
    *)     echo "Unknown arch: $ARCH (use arm64, arm, or all)" >&2; exit 1 ;;
esac

echo "=== qwen2API Magisk Module Builder ==="
echo "Project: $PROJECT_DIR"
echo "Target:  $ARCH"
echo ""

# ---- Step 1: Build frontend ----
echo "[1/3] Building frontend..."
cd "$PROJECT_DIR/frontend"
npm ci --prefer-offline 2>/dev/null || npm ci
npm run build
echo "Frontend built: frontend/dist/"
echo ""

# ---- Step 2: Build backend binary for each arch ----
build_backend() {
    local goarch="$1"
    local bin_name="qwen2api"
    local out_dir="$BUILD_DIR/${goarch}"

    echo "[2/3] Building backend for linux/$goarch..."
    cd "$PROJECT_DIR/backend"

    mkdir -p "$out_dir"
    CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" \
        go build -trimpath -ldflags="-s -w" -o "$out_dir/$bin_name" .

    local size
    size=$(du -h "$out_dir/$bin_name" | cut -f1)
    echo "Backend built: $out_dir/$bin_name ($size)"
    echo ""
}

# ---- Step 3: Assemble Magisk module zip ----
assemble_module() {
    local goarch="$1"
    local out_dir="$BUILD_DIR/${goarch}"
    local module_dir="$BUILD_DIR/module-${goarch}"
    local zip_name="qwen2api-magisk-linux-${goarch}.zip"
    local zip_path="$BUILD_DIR/$zip_name"

    echo "[3/3] Assembling Magisk module for $goarch..."

    # Clean previous build
    rm -rf "$module_dir"
    mkdir -p "$module_dir"

    # Copy module metadata
    cp "$PROJECT_DIR/magisk/module.prop" "$module_dir/"
    cp "$PROJECT_DIR/magisk/service.sh" "$module_dir/"
    cp "$PROJECT_DIR/magisk/post-fs-data.sh" "$module_dir/"
    chmod 755 "$module_dir/service.sh" "$module_dir/post-fs-data.sh"

    # Copy backend binary
    cp "$out_dir/qwen2api" "$module_dir/"
    chmod 755 "$module_dir/qwen2api"

    # Copy frontend dist (served by the backend from BASE_DIR/frontend/dist)
    mkdir -p "$module_dir/frontend/dist"
    cp -r "$PROJECT_DIR/frontend/dist/." "$module_dir/frontend/dist/"

    # Create the zip
    cd "$BUILD_DIR"
    rm -f "$zip_name"
    cd "$module_dir"
    zip -r "$zip_path" . -x "*.DS_Store"

    local zip_size
    zip_size=$(du -h "$zip_path" | cut -f1)
    echo ""
    echo "=== Build complete ==="
    echo "Module: $zip_path ($zip_size)"
    echo ""
    echo "Installation:"
    echo "  1. Transfer $zip_name to your phone"
    echo "  2. Open Magisk -> Modules -> Install from storage"
    echo "  3. Select the zip file and reboot"
    echo "  4. After boot, edit /data/local/qwen2api/config.env to set ADMIN_KEY"
    echo "  5. Restart the service: kill \$(cat /data/local/qwen2api/qwen2api.pid); /data/adb/modules/qwen2api/service.sh"
    echo "  6. Access WebUI at http://127.0.0.1:7860"
    echo ""
}

# Execute builds
for arch in $GOARCH; do
    build_backend "$arch"
    assemble_module "$arch"
done
