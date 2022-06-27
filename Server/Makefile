all:	quiz

quiz:	dep
	go build -o bin/quiz .

lint:	dep
	go vet ./...

dep:


clean:
	go clean ./...
	rm -f bin/*
