# Example JSON

## Intro

JSON is common to store request and response data.

`NewJSONHandler` is provided to work as a helper to reduce boilerplate code.

## Usage

```shell
go run main.go
```

```shell
curl -X POST localhost:8080/v1/whole -d '{"id":1,"name":"Alice"}'
```

```shell
curl -X POST localhost:8080/v1/semi -d '{"id":2,"name":"Bob"}'
```