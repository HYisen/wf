# Web Frame

`wf` is a minimal web framework in Go that helps me evaluate.

## Story

Whenever facing a non-trivial task, it's a best-practice to evaluate candidates before including new dependencies.
Whether to implementing and maintain one by yourself in-place or elsewhere, or introduce gorgeous third-part packages?

`wf`, previous frame.go in my projects, is the gauge that helps me decide whether I shall seek for other's help.

If the task is less important or simpler, I would inline the necessary function as codes around `http.Serve`.

If the task matters or is more complicated, I would introduce a full-fledged web framework such as
[gin](https://github.com/gin-gonic/gin).

What's more, `wf` reminds me to keep app logic away from web framework, pulling and grouping similar things together,
which matters if you shall not put everything inside one huge main function.

## Usage

Inside your project repository root, add `wf` dependency.

```bash
go get -u github.com/hyisen/wf
```

Take a look over /examples to find how to use `wf`.