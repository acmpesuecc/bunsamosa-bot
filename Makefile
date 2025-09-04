all:
	@echo "\nUsing go run with BUNSAMOSA_DEV_MODE=1"
	BUNSAMOSA_DEV_MODE=1 JSON_LOG_DIR="logs" go run .

ci:
	@echo "\nCleaning db's"
	rm -rf test.db
	@echo "\nInitialising schema: Using go run with BUNSAMOSA_DEV_MODE=1"
	BUNSAMOSA_DEV_MODE=1 JSON_LOG_DIR="logs" timeout 5 go run . || true
	@echo "\nPopulating DB with maintainer and participant info from dev/init.sql"
	make populate-db
	@echo "\nUsing go run with BUNSAMOSA_DEV_MODE=1"
	make

setup-schema:
	@echo "\nCleaning db's"
	rm -rf test.db
	@echo "\nInitialising schema: Using go run with BUNSAMOSA_DEV_MODE=1"
	BUNSAMOSA_DEV_MODE=1 JSON_LOG_DIR="logs" go run .

populate-db:
	cat dev/init.sql | sqlite3 test.db

clean:
	@echo "\nCleaning db's"
	rm -rf test.db

deploy:
	GOOS=linux GOARCH=amd64 go build
	JSON_LOG_DIR="logs" ./bunsamosa-bot
