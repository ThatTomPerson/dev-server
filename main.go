package main // import "ttp.sh/dev-server"

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yookoala/gofast"
	"ttp.sh/dev-server/mserver"

	"github.com/rakyll/autopprof"
)

var version = "master"

var port = flag.String("port", "2000", "port to listen on")
var dev = flag.Bool("dev", false, "enable debugging libraries")
var startFpmFlag = flag.Bool("start-fpm", false, "start fpm")

var printVersion = flag.Bool("version", false, "display version and exit")

var pwd = ""

func init() {
	pwd, _ = os.Getwd()
}

func Exists(name string) bool {
	s, err := os.Stat(name)
	return !os.IsNotExist(err) && !s.IsDir()
}

func startFpm() {
	child := exec.Command("php-fpm", "--nodaemonize")

	child.Stdout = os.Stdout
	child.Stderr = os.Stderr

	logrus.Info("Running php-fpm")
	logrus.Fatal(child.Run())
}

func getSiteRoot(r *http.Request) string {
	parts := strings.Split(r.TLS.ServerName, ".")
	site := parts[len(parts)-2]

	return filepath.Join(pwd, "sites", site, "public")
}

func main() {
	flag.Parse()

	if *printVersion {
		logrus.Printf("dev-server v%s", version)
		return
	}

	if *dev {
		autopprof.Capture(autopprof.CPUProfile{
			Duration: 15 * time.Second,
		})
	}

	if *startFpmFlag {
		go startFpm()
	}

	fcgiAddress := os.Getenv("FASTCGI_ADDR")
	if fcgiAddress == "" {
		fcgiAddress = "127.0.0.1:9000"
	}

	connFactory := gofast.SimpleConnFactory("tcp", fcgiAddress)
	clientFactory := gofast.SimpleClientFactory(connFactory, 0)

	// if file exists and ext not .php
	// 		echo file
	// else
	// 		pass to fpm

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		root := getSiteRoot(r)
		uri := filepath.Join(root, r.RequestURI)

		if filepath.Ext(uri) != ".php" && Exists(uri) {
			http.ServeFile(w, r, uri)
			return
		}

		r.Host = fmt.Sprintf("%s:%s", r.TLS.ServerName, *port)

		gofast.NewHandler(
			gofast.NewFileEndpoint(filepath.Join(root, "index.php"))(gofast.BasicSession),
			clientFactory,
		).ServeHTTP(w, r)
	})
	cert, key := getCACerts()
	address := fmt.Sprintf(":%s", *port)
	logrus.Fatal(mserver.ListenAndServeTLS(address, cert, key, nil))
}

func getCACerts() (string, string) {
	root := getCAROOT()
	return filepath.Join(root, "rootCA.pem"), filepath.Join(root, "rootCA-key.pem")
}

func getCAROOT() string {
	if env := os.Getenv("CAROOT"); env != "" {
		return env
	}

	var dir string
	switch {
	case runtime.GOOS == "windows":
		dir = os.Getenv("LocalAppData")
	case os.Getenv("XDG_DATA_HOME") != "":
		dir = os.Getenv("XDG_DATA_HOME")
	case runtime.GOOS == "darwin":
		dir = os.Getenv("HOME")
		if dir == "" {
			return ""
		}
		dir = filepath.Join(dir, "Library", "Application Support")
	default: // Unix
		dir = os.Getenv("HOME")
		if dir == "" {
			return ""
		}
		dir = filepath.Join(dir, ".local", "share")
	}
	return filepath.Join(dir, "mkcert")
}

func FileFSMiddleware(root string) gofast.Middleware {
	return nil
}
