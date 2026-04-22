#!/bin/bash
set -e

echo "🚀 Starting Wire Generation..."

# 定義所有需要跑 wire 的路徑
SERVICES=$(find apps -name "wire.go" -exec dirname {} \;)

for SERVICE in $SERVICES; do
    echo "  -> Updating DI in: $SERVICE"
    (cd "$SERVICE" && wire)
done

if [ $? -eq 0 ]; then
  echo "✅ Wire generated successfully!"
else
  echo "❌ Wire generation failed, please check the above errors."
  exit 1
fi