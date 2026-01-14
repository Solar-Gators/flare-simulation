#!/usr/bin/env bash
#before running script use chmod to make the file executable
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}/frontend"

PORT="${PORT:-5173}"

echo "Starting Vite dev server at http://localhost:${PORT}"
echo "Press Ctrl+C to stop."
npm run dev -- --host --port "${PORT}" --open
