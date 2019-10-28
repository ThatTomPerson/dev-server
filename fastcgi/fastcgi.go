package fastcgi

import (
	"net"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/eternnoir/gncp"
)

type handler struct {
	root string
	pool gncp.ConnPool
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.pool.Get()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
	}
	_ = conn

	if err := conn.Close(); err != nil {
		log.Error(err)
	}
}

func NewHandler(root, fastcgiAddress string) (http.Handler, error) {
	pool, err := NewPool(fastcgiAddress)
	if err != nil {
		return nil, err
	}
	return &handler{
		root: root,
		pool: pool,
	}, nil
}

func NewPool(fastcgiAddress string) (gncp.ConnPool, error) {
	return gncp.NewPool(1, 10, NewConnFactory(fastcgiAddress))
}

func NewConnFactory(fastcgiAddress string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		return net.Dial("tcp", fastcgiAddress)
	}
}
