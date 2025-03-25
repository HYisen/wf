# Example Path

## Intro

In RESTful design, data such as ID is common in URL path.

`ResourceWithID` is provided to work as a helper to match path with content.

`PathIDParser` is provided to work as a helper to extract id from path and stored in req.

`ResourceWithIDs` is provided for more complicated path structures, in its parts format,
use empty string as placeholder of a numeric id. Check its unit test cases for more examples.

Check `simple` for example. If what you need is all, use `comprehensive` as an example.

`HasQuery` is designed to cooperated with other matchers.

`MatchAll` is provided as a helper to combine multiple matchers.

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

```shell
curl -X GET localhost:8080/v1/users/123/items/456
```

```shell
curl -X GET localhost:8080/v1/ask
```

```shell
curl -X GET "localhost:8080/v1/ask?q=this_is_a_question"
```