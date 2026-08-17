# cm531_3239

Go project for module `example.com/mestransform`.

## Standard commands

```bash
go build ./...
go test -count=1 ./...
```

## Run

```bash
go run ./cmd/mesctl
```

## Docker validation

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh my-go-task linux/arm64
./build_benzhi_docker.sh my-go-task linux/amd64
docker run -it my-go-task:latest
```

## Known initial failures

See `BUG_REPRO.md` for the exact command and output captured during packaging.
