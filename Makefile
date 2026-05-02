.PHONY: help run build test clean migrate seed docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Run the application in development mode
	@echo "🚀 Starting server..."
	go run main.go

build: ## Build the application
	@echo "🔨 Building application..."
	go build -o photobooth-backend main.go
	@echo "✅ Build complete: ./photobooth-backend"

test: ## Run tests
	@echo "🧪 Running tests..."
	go test -v ./...

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	rm -f photobooth-backend
	rm -rf uploads/photos/* uploads/processed/* uploads/thumbnails/* uploads/strips/*
	@echo "✅ Clean complete"

install: ## Install dependencies
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed"

migrate: ## Run database migrations
	@echo "🗄️  Running migrations..."
	go run cmd/migrate/main.go
	@echo "✅ Migrations complete"

seed: ## Seed database with sample data
	@echo "🌱 Seeding database..."
	go run cmd/seed/main.go
	@echo "✅ Seeding complete"

dev: ## Run in development mode with hot reload (requires air)
	@echo "🔥 Starting with hot reload..."
	air

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t photobooth-backend:latest .
	@echo "✅ Docker image built"

docker-up: ## Start Docker containers
	@echo "🐳 Starting Docker containers..."
	docker-compose up -d
	@echo "✅ Containers started"

docker-down: ## Stop Docker containers
	@echo "🐳 Stopping Docker containers..."
	docker-compose down
	@echo "✅ Containers stopped"

docker-logs: ## View Docker logs
	docker-compose logs -f backend

setup: install ## Setup project (install deps + create directories)
	@echo "📁 Creating directories..."
	mkdir -p uploads/{photos,templates,processed,thumbnails,strips,qris}
	@echo "📝 Creating .env file..."
	@if [ ! -f .env ]; then cp .env.example .env; echo "✅ .env created (please update credentials)"; else echo "⚠️  .env already exists"; fi
	@echo "✅ Setup complete"
	@echo ""
	@echo "Next steps:"
	@echo "1. Update .env file with your credentials"
	@echo "2. Create database: make db-create"
	@echo "3. Run migration: make migrate"
	@echo "4. Seed data: make seed"
	@echo "5. Start server: make run"

lint: ## Run linter
	@echo "🔍 Running linter..."
	golangci-lint run
	@echo "✅ Linting complete"

format: ## Format code
	@echo "✨ Formatting code..."
	go fmt ./...
	@echo "✅ Formatting complete"

prod-build: ## Build for production
	@echo "🏭 Building for production..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o photobooth-backend main.go
	@echo "✅ Production build complete"

backup-db: ## Backup database
	@echo "💾 Backing up database..."
	pg_dump -U postgres photobooth > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "✅ Backup complete"

restore-db: ## Restore database from backup (usage: make restore-db FILE=backup.sql)
	@echo "📥 Restoring database..."
	psql -U postgres photobooth < $(FILE)
	@echo "✅ Restore complete"

db-create: ## Create database
	@echo "🗄️  Creating database..."
	createdb photobooth
	@echo "✅ Database 'photobooth' created!"

db-drop: ## Drop database
	@echo "⚠️  Dropping database..."
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		dropdb photobooth; \
		echo "✅ Database dropped!"; \
	else \
		echo "❌ Cancelled"; \
	fi

db-reset: db-drop db-create migrate seed ## Reset database (drop + create + migrate + seed)
	@echo "✅ Database reset complete!"

db-status: ## Check database connection
	@echo "🔍 Checking database connection..."
	@psql -U postgres -d photobooth -c "SELECT 'Database is accessible!' as status;" || echo "❌ Cannot connect to database"

quick-start: setup db-create migrate seed ## Quick start (setup + create db + migrate + seed)
	@echo ""
	@echo "🎉 Quick start complete!"
	@echo ""
	@echo "You can now run: make run"

check: ## Check if all required tools are installed
	@echo "🔍 Checking requirements..."
	@command -v go >/dev/null 2>&1 || { echo "❌ Go is not installed"; exit 1; }
	@command -v psql >/dev/null 2>&1 || { echo "❌ PostgreSQL is not installed"; exit 1; }
	@echo "✅ All requirements met"

.DEFAULT_GOAL := help
