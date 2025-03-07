# Example Security

## Intro

It's my personal preference to add Token field in HTTP request header if it's necessary.
So I added it to my web framework.

I invented it myself, just like many other developers did.

It is simple thus efficient, but require HTTPS only.

It is similar to [Bearer](https://datatracker.ietf.org/doc/html/rfc6750).

## Tools

`AttachToken` would be invoked automatically to extract possible Token field into `ctx`.
Use `DetachToken` in HandleFunc later to fetch it back.

Use `ParseEmpty` if there is actually nothing to parse in request.

Use `FormatEmpty` if there is actually nothing to format in response.

`NewCodedErrorf` and `NewCodedError` are helpers to generate *CodedError, whose code would be used in HTTP Response.
The difference between them is identical to that between `fmt.Printf` and `fmt.Print`.

## Usage

```shell
go run main.go
```

Under curl verbose output mode, you could find the HTTP Status code varies.

```shell
curl --verbose -X POST localhost:8080/v1/vital
```

```shell
curl --verbose -X POST localhost:8080/v1/vital -H "Token: 123456"
```

```shell
curl --verbose -X POST localhost:8080/v1/vital -H "Token: top_secret"
```