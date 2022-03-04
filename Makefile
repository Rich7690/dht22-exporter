build:
	GOARCH=arm GOARM=5 go build -o main.armv5 && GOARCH=arm go build -o main.arm