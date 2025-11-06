all:
	@echo "\nUsing go run with BUNSAMOSA_DEV_MODE=1"
	BUNSAMOSA_DEV_MODE=1 go run .

schema:
	@echo "\nCleaning db's"
	rm -rf test.db
	@echo "\nUsing go run with BUNSAMOSA_DEV_MODE=1"
	BUNSAMOSA_DEV_MODE=1 go run .
	@echo "\nDisplaying DB Schemas"
	echo ".schema" | sqlite3 test.db

clean:
	@echo "\nCleaning db's"
	rm -rf test.db

deploy:
	GOOS=linux GOARCH=amd64 go build
	./bunsamosa-bot
