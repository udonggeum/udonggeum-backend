#!/bin/sh

# 모든 xlsx 파일 임포트
DATA_DIR="/app/gold_data"

if [ ! -d "$DATA_DIR" ]; then
  echo "Error: $DATA_DIR directory not found"
  exit 1
fi

echo "Starting bulk import from $DATA_DIR"

for file in "$DATA_DIR"/*.xlsx; do
  if [ -f "$file" ]; then
    filename=$(basename "$file")
    echo "================================================"
    echo "Processing: $filename"
    echo "================================================"
    echo "yes" | /root/seed "$file"

    if [ $? -eq 0 ]; then
      echo "✓ Successfully imported $filename"
    else
      echo "✗ Failed to import $filename"
    fi
    echo ""
  fi
done

echo "All imports completed!"
