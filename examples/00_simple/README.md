# Example Simple

## Usage

Start service. Shall block, not return with error messages such as port already in use.

```shell
go run main.go
```

Yes, there is no limitation on GET with request body.
[ref](https://stackoverflow.com/questions/978061/http-get-with-request-body)

```shell
curl -X GET localhost:8080/echo -d 'this_is_payload'
```