run:
	go-bindata ./references/... 
	go run main.go vision.go bindata.go
	rm ./bindata.go

build:
	go-bindata ./references/...
	go build

clean:
	rm ./bindata.go
	rm ./poker-client-pokerstars
	rm ./debug