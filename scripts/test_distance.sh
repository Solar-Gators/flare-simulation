#!/usr/bin/env bash
set -euo pipefail

curl -X POST http://localhost:8080/distance \
  -H "Content-Type: application/json" \
  -d '{
    "v": 20,
    "batteryWh": 5000,
    "solarWhPerMin": 5,
    "etaDrive": 0.9,
    "raceDayMin": 480,
    "rWheel": 0.2792,
    "tMax": 45,
    "pMax": 10000,
    "m": 285,
    "g": 9.81,
    "cRr": 0.0015,
    "rho": 1.225,
    "cD": 0.21,
    "a": 0.456,
    "theta": 0
  }'
