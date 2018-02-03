all: linux windows

linux:
	GOOS=linux go build -o bin/Likegram src/likegram.go

windows:
    GOOS=windows go build -o bin/Likegram.exe src/likegram.go

clean:
	rm -r bin/*