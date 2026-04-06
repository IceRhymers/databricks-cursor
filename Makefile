.PHONY: build test clean dist

build:
	go build -o databricks-cursor .

test:
	go test ./... -v

clean:
	rm -f databricks-cursor

dist: clean build
