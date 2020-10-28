# password-hash-service
Hash and Encode a Password String

### Build and run

```
$ go build
$ ./password-hash-service
```

### Usage

When the server runs, it listens to port 8080 by default. You can customize the address by specifying the "addr" command line parameter:

```
  -addr string
        HTTP listen address (default ":8080")
```

Adding a password:

```
$ curl --data "password=angryMonkey" -i http://localhost:8080/hash
HTTP/1.1 201 Created
Content-Type: application/json
Location: /hash/1
Date: Wed, 28 Oct 2020 06:02:06 GMT
Content-Length: 9

{"id":1}
```

Retrieving a password hash:

```
$ curl -i http://localhost:8080/hash/1
HTTP/1.1 200 OK
Content-Type: application/json
Date: Wed, 28 Oct 2020 06:10:25 GMT
Content-Length: 100

{"hash":"ZEHhWB65gUlzdVwtDQArEyx+KVLzp/aTaRaPlBzYRIFj6vjFdqEb0Q5B8zVKCZ0vKbZPZklJz0Fd7su2A+gf7Q=="}
```

Getting statistics:

```
$ curl -i http://localhost:8080/stats
HTTP/1.1 200 OK
Content-Type: application/json
Date: Wed, 28 Oct 2020 06:14:47 GMT
Content-Length: 26

{"total":1,"average":972}
```

Shutting down gracefully:

```
$ curl -i -X POST http://localhost:8080/shutdown
HTTP/1.1 200 OK
Date: Wed, 28 Oct 2020 06:20:49 GMT
Content-Length: 2
Content-Type: text/plain; charset=utf-8

OK
```
