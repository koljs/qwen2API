#!/system/bin/sh
# qwen2API post-fs-data script
# Runs after the filesystem is mounted, before boot completes.
# Fixes executable permissions lost during Magisk module installation.

MODDIR=${0%/*}
chmod 755 "$MODDIR/qwen2api"
