cd ../
go mod tidy
env GOOS=linux GOARCH=arm64 go build -o build/server_arm
env go build -o build/server_linux
