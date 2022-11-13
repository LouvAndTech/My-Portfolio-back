Build:
	@echo "Building binaries for the host system..."
	@go build -o ./site-backend.go -v
	@echo "Done."

Serve:
	@echo "Starting the server..."
	@go run ./site-backend.go serve
	@echo "Done."
