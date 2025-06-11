# Road Detector Go API Makefile

# Переменные
APP_NAME=road-detector-go
BINARY_NAME=server
DB_COMPOSE_FILE=docker-compose.db.yml

# Цвета для вывода
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help build run db-up db-down db-restart db-status clean rebuild dev logs test migrate migrate-up migrate-down migrate-reset db-setup

# Помощь
help:
	@echo "$(BLUE)Road Detector Go API - Доступные команды:$(NC)"
	@echo ""
	@echo "$(GREEN)База данных:$(NC)"
	@echo "  db-up       - Запустить PostgreSQL базу данных"
	@echo "  db-down     - Остановить базу данных"
	@echo "  db-restart  - Перезапустить базу данных"
	@echo "  db-status   - Проверить статус базы данных"
	@echo "  migrate     - Выполнить миграции базы данных"
	@echo "  migrate-up  - Применить миграции"
	@echo "  migrate-down- Откатить последнюю миграцию"
	@echo "  migrate-reset- Сбросить базу данных и применить миграции"
	@echo ""
	@echo "$(GREEN)Сборка и запуск:$(NC)"
	@echo "  build       - Собрать проект"
	@echo "  run         - Запустить сервер"
	@echo "  dev         - Пересобрать и запустить сервер"
	@echo "  rebuild     - Полная пересборка проекта"
	@echo ""
	@echo "$(GREEN)Утилиты:$(NC)"
	@echo "  clean       - Очистить собранные файлы"
	@echo "  logs        - Показать логи базы данных"
	@echo "  test        - Запустить тесты"

# База данных
db-up:
	@echo "$(YELLOW)Запускаем PostgreSQL базу данных...$(NC)"
	@cd .. && docker-compose -f $(DB_COMPOSE_FILE) up -d
	@echo "$(GREEN)База данных запущена!$(NC)"
	@echo "$(BLUE)Подключение: localhost:5432$(NC)"
	@echo "$(BLUE)База: road_detector, Пользователь: postgres, Пароль: postgres123$(NC)"

db-down:
	@echo "$(YELLOW)Останавливаем базу данных...$(NC)"
	@cd .. && docker-compose -f $(DB_COMPOSE_FILE) down
	@echo "$(GREEN)База данных остановлена!$(NC)"

db-restart:
	@echo "$(YELLOW)Перезапускаем базу данных...$(NC)"
	@cd .. && docker-compose -f $(DB_COMPOSE_FILE) down
	@cd .. && docker-compose -f $(DB_COMPOSE_FILE) up -d
	@echo "$(GREEN)База данных перезапущена!$(NC)"

db-status:
	@echo "$(YELLOW)Проверяем статус базы данных...$(NC)"
	@cd .. && docker-compose -f $(DB_COMPOSE_FILE) ps

# Сборка
build:
	@echo "$(YELLOW)Собираем проект...$(NC)"
	@go mod tidy
	@go build -o bin/$(BINARY_NAME) ./cmd/server
	@echo "$(GREEN)Проект собран: bin/$(BINARY_NAME)$(NC)"

# Запуск
run: build
	@echo "$(YELLOW)Запускаем сервер...$(NC)"
	@echo "$(BLUE)API будет доступно по адресу: http://localhost:8080/api/v1$(NC)"
	@./bin/$(BINARY_NAME)

# Разработка - пересборка и запуск
dev:
	@echo "$(YELLOW)Пересборка и запуск для разработки...$(NC)"
	@make clean
	@make build
	@make run

# Полная пересборка
rebuild: clean build
	@echo "$(GREEN)Полная пересборка завершена!$(NC)"

# Очистка
clean:
	@echo "$(YELLOW)Очищаем собранные файлы...$(NC)"
	@rm -rf bin/
	@mkdir -p bin
	@echo "$(GREEN)Очистка завершена!$(NC)"

# Логи базы данных
logs:
	@echo "$(YELLOW)Показываем логи базы данных...$(NC)"
	@cd .. && docker-compose -f $(DB_COMPOSE_FILE) logs -f postgres

# Тесты
test:
	@echo "$(YELLOW)Запускаем тесты...$(NC)"
	@go test -v ./...
	@echo "$(GREEN)Тесты завершены!$(NC)"

# Миграции базы данных
migrate: migrate-up
	@echo "$(GREEN)Миграции применены!$(NC)"

migrate-up:
	@echo "$(YELLOW)Применяем миграции базы данных...$(NC)"
	@docker exec road-detector-postgres-dev psql -U postgres -d road_detector -f /tmp/001_create_routes_table.sql || true
	@docker cp migrations/001_create_routes_table.sql road-detector-postgres-dev:/tmp/
	@docker exec road-detector-postgres-dev psql -U postgres -d road_detector -f /tmp/001_create_routes_table.sql
	@docker cp migrations/002_create_segments_table.sql road-detector-postgres-dev:/tmp/
	@docker exec road-detector-postgres-dev psql -U postgres -d road_detector -f /tmp/002_create_segments_table.sql
	@echo "$(GREEN)Миграции успешно применены!$(NC)"

migrate-down:
	@echo "$(YELLOW)Откатываем миграции...$(NC)"
	@docker exec road-detector-postgres-dev psql -U postgres -d road_detector -c "DROP TABLE IF EXISTS segments CASCADE;"
	@docker exec road-detector-postgres-dev psql -U postgres -d road_detector -c "DROP TABLE IF EXISTS routes CASCADE;"
	@echo "$(GREEN)Миграции откачены!$(NC)"

migrate-reset: migrate-down migrate-up
	@echo "$(GREEN)База данных сброшена и миграции применены заново!$(NC)"

# Полная установка с миграциями
db-setup: db-up
	@echo "$(YELLOW)Ожидаем запуска базы данных...$(NC)"
	@sleep 5
	@make migrate
	@echo "$(GREEN)База данных готова к работе!$(NC)"

# Установка по умолчанию
all: db-setup build run 