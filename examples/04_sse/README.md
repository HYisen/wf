# Example SSE

## Intro

SSE(Server-Sent Events) is widely use in LLM stream mode to allow output comes out word by word.

To enable user with the best experience, I support it in `wf`.

## Tools

Use `wf.NewServerSentEventsHandler` to creates an SSE handler.

The main path is, how to convert request to an output channel of `wf.MessageEvent`,
which stored in `wf.StreamGenerator` as `wf.HandleFunc` in parameters.

Don't forget to close the output channel when it's done. Handler's callers does not force it.

## Usage

```shell
go run main.go
```

In other TTY, start client to fetch.

```shell
go run client/main.go
```

If you forced the client to stop before server finish its sending (a potential route is by press Ctrl + Z),
the server would notify it through `ctx` in `Handle`, a special log would be printed in server side.

This feature is useful when the client gone situation becomes a notable thing.