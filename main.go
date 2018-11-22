package main // import "ttp.sh/dev-server"

import (
	"flag"
	"fmt"
)

var cert = flag.String("cert", "rootCA.pem", "path to the root CA cert")
var key = flag.String("key", "rootCA-key.pem", "path to the root CA key")

var port = flag.String("port", "2000", "port to listen on")

func main() {
	flag.Parse()

	s := NewServer(*cert, *key)
	s.Listen(fmt.Sprintf(":%s", *port))
}
