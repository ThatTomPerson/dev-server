package main // import "ttp.sh/dev-server"

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yookoala/gofast"
	"ttp.sh/dev-server/mserver"

	"github.com/rakyll/autopprof"
)

var cert = flag.String("cert", "rootCA.pem", "path to the root CA cert")
var key = flag.String("key", "rootCA-key.pem", "path to the root CA key")

var 

var port = flag.String("port", "2000", "port to listen on")

func Exists(name string) bool {
	s, err := os.Stat(name)
	return !os.IsNotExist(err) && !s.IsDir()
}

func main() {
	flag.Parse()

	autopprof.Capture(autopprof.CPUProfile{
		Duration: 15 * time.Second,
	})

	connFactory := gofast.SimpleConnFactory("unix", "/Users/thomasalbrighton/.config/valet/valet.sock")
	clientFactory := gofast.SimpleClientFactory(connFactory, 0)

	pwd, _ := os.Getwd()

	// if file exists and ext not .php
	// 		echo file
	// else
	// 		pass to fpm

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		parts := strings.Split(r.TLS.ServerName, ".")
		site := parts[len(parts)-2]

		root := filepath.Join(pwd, "sites", site, "public")

		uri := filepath.Join(root, r.RequestURI)

		if filepath.Ext(uri) != ".php" && Exists(uri) {
			http.ServeFile(w, r, uri)
			return
		}

		gofast.NewHandler(
			gofast.Chain(
				// FileFSMiddleware(pwd),
				gofast.NewFileEndpoint(filepath.Join(root, "index.php")),
			)(gofast.BasicSession),
			clientFactory,
		).ServeHTTP(w, r)
	})

	address := fmt.Sprintf(":%s", *port)
	logrus.Fatal(mserver.ListenAndServeTLS(address, *cert, *key, nil))
}

func FileFSMiddleware(root string) gofast.Middleware {
	return nil
}
