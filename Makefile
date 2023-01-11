Build:
	@echo "Building binaries for the host system..."
	@go build -o ./output/backend-exe ./*.go
	@echo "Done."

Serve:
	@echo "Starting the server..."
	@go run ./*.go serve --http 0.0.0.0:8090
	@echo "Done."
