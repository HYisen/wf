# Example Path

## Intro

In RESTful design, data such as ID is common in URL path.

`ResourceWithID` is provided to work as a helper to match path with content.

`PathIDParser` is provided to work as a helper to extract id from path and stored in req.

Check `simple` for example. If what you need is all, use `comprehensive` as an example.

## Usage

```shell
go run main.go
```

```shell
curl -X DELETE localhost:8080/v1/widgets/1234
```

```shell
curl -X POST localhost:8080/v1/items/1000/content -d 'this_is_body'
```
