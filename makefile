run:
	go-bindata ./res/references/... 
	go run main.go utils.go bindata.go $(arg1)
	rm ./bindata.go

build:
	go-bindata ./res/references/...
	go build

clean:
	rm ./bindata.go
	rm ./poker-client-pokerstars
	rm ./debug