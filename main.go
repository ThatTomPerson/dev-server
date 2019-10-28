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

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"github.com/yookoala/gofast"
	"ttp.sh/dev-server/devtls"

	reaper "github.com/ramr/go-reaper"

	"flag"
)

var version = "master"

var (
	host      string
	port      string
	reap      bool
	supervise bool
	path      string
)

func init() {
	flag.StringVar(&host, "host", "0.0.0.0", "host to listen on")
	flag.StringVar(&port, "port", "2000", "port to listen on")
	flag.BoolVar(&reap, "init", false, "watch for and reap zombie processes")
	flag.BoolVar(&supervise, "supervise", false, "start and supervise a php-fpm server")
	flag.StringVar(&path, "root", ".", "root directory to look for sites")
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

	return filepath.Join(path, "sites", site, "public")
}

func NewSitesEndpoint() gofast.Middleware {
	return gofast.Chain(
		gofast.BasicParamsMap,
		gofast.MapHeader,
		MapEndpoint,
	)
}

func MapEndpoint(inner gofast.SessionHandler) gofast.SessionHandler {
	return func(client gofast.Client, req *gofast.Request) (*gofast.ResponsePipe, error) {
		r := req.Raw
		dir := getSiteRoot(r)
		webpath := "/index.php"
		fullPath := filepath.Join(dir, webpath)
		req.Params["REQUEST_URI"] = r.URL.RequestURI()
		req.Params["SCRIPT_NAME"] = webpath
		req.Params["SCRIPT_FILENAME"] = fullPath
		req.Params["DOCUMENT_URI"] = r.URL.Path
		req.Params["DOCUMENT_ROOT"] = dir

		spew.Dump(req.Params)

		return inner(client, req)
	}
}

func main() {
	flag.Parse()
	if reap {
		go reaper.Reap()
	}

	if supervise {
		go startFpm()
	}

	if (path)[0] == '.' {
		path, _ = filepath.Abs(path)
	}

	fcgiAddress := os.Getenv("FASTCGI_ADDR")
	if fcgiAddress == "" {
		fcgiAddress = "127.0.0.1:9000"
	}

	logrus.Infof("connecting to FASTCGI on %s", fcgiAddress)

	connFactory := gofast.SimpleConnFactory("tcp", fcgiAddress)
	clientFactory := gofast.SimpleClientFactory(connFactory, 20)
	_ = clientFactory
	logrus.Infof("using pool")
	handler := gofast.NewHandler(
		NewSitesEndpoint()(gofast.BasicSession),
		clientFactory,
	)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		root := getSiteRoot(r)
		if Exists(filepath.Join(root, r.URL.Path)) {
			http.ServeFile(w, r, filepath.Join(root, r.URL.Path))
			return
		}

		handler.ServeHTTP(w, r)
	})

	cert, key := getCACerts()
	address := fmt.Sprintf("%s:%s", host, port)
	logrus.Fatal(devtls.ListenAndServeTLS(address, cert, key, nil))
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
