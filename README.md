# Fairshares

A [Flexpool](https://www.flexpool.io/) miner monitor and offline notifier.

## Usage

1. Configuration

```
$ cp config/config.sample.toml config/config.toml
```

2. Fulfill the dependency

```
$ go get github.com/mattn/go-sqlite3
$ go get github.com/BurntSushi/toml
$ go get github.com/mailjet/mailjet-apiv3-go
```

3. Run

```
$ go run cmd/main.go
```

## Known Issues

- Unexpected warning from `go-sqlite3`: https://github.com/mattn/go-sqlite3/issues/803
