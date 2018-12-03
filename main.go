package main // import "ttp.sh/dev-server"

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yookoala/gofast"
	"ttp.sh/dev-server/devtls"

	reaper "github.com/ramr/go-reaper"

	"gopkg.in/alecthomas/kingpin.v2"
)

var version = "master"

var (
	app = kingpin.New("dev-server", "tls terminator server that generates valid certs based on SNI and forwards requests to a fastcgi")

	startCmd  = app.Command("start", "start the terminator server")
	supervise = startCmd.Flag("supervise", "start and supervise a php-fpm server").Short('s').Bool()
	host      = startCmd.Flag("server", "Server address.").Default("0.0.0.0").Short('h').IP()
	port      = startCmd.Flag("port", "start and supervise a php-fpm server").Short('p').Default("2000").String()
	reap      = startCmd.Flag("init", "watch for and reap zombie processes").Short('i').Default("false").Bool()
	path      = startCmd.Flag("root", "root directory to look for sites").Short('d').Default(".").ExistingDir()

	versionCmd = app.Command("version", "display the version and exit")
)

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

	return filepath.Join(*path, "sites", site, "public")
}

func start(startCmd *kingpin.CmdClause) {
	if *reap {
		go reaper.Reap()
	}

	if *supervise {
		go startFpm()
	}

	if (*path)[0] == '.' {
		*path, _ = filepath.Abs(*path)
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
		logrus.Infof("serving from: %s", root)
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
	address := fmt.Sprintf("%s:%s", *host, *port)
	logrus.Fatal(devtls.ListenAndServeTLS(address, cert, key, nil))
}

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case versionCmd.FullCommand():
		logrus.Printf("dev-server v%s", version)
	case startCmd.FullCommand():
		start(startCmd)
	}
}

func getCACerts() ([]byte, []byte) {
	root := getCAROOT()
	certPath := filepath.Join(root, "rootCA.pem")
	keyPath := filepath.Join(root, "rootCA-key.pem")

	cert, err := ioutil.ReadFile(certPath)
	if err != nil {
		logrus.Fatal("Couldn't find cert: %s", certPath)
	}
	key, err := ioutil.ReadFile(keyPath)
	if err != nil {
		logrus.Fatal("Couldn't find key: %s", keyPath)
	}

	return cert, key
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
