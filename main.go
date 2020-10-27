package main

import (
	"flag"
)

var httpAddr = flag.String("addr", ":8080", "HTTP listen address")

func main() {
	flag.Parse()

	svc := NewHashService(httpAddr)

	svc.Run()
}
