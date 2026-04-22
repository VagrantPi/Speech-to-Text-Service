# 1. Run infra first
docker compose -f docker-compose.infra.yml up -d

# 2. Build application images (if needed)
./scripts/build-images.sh

# 3. Run infra + migrations + applications
docker compose -f docker-compose.yml -f docker-compose.infra.yml up -d

# 4. Clean up migration container (optional - it exits automatically)
docker compose -f docker-compose.yml -f docker-compose.infra.yml rm -f infra-migration
