package server

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xandout/soxy/proxy"
)

// Start starts the http server
func Start(c *cli.Context) error {
	port := c.String("port")
	http.Handle("/", &socketHandler{apiKey: c.String("api-key")})
	err := http.ListenAndServe(port, nil)
	log.Errorf("HTTP SERVER: %v", err.Error())
	return err

}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type socketHandler struct {
	apiKey string
}

func (h *socketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.apiKey != "" {
		apiKey := r.Header.Get("X-Api-Key")
		if h.apiKey != apiKey {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Forbidden"))
			log.Errorf("HTTP SERVER: Invalid API Key '%v'", apiKey)
			return
		}
	}
	q := r.URL.Query()
	var useTLS bool
	if q.Get("useTLS") != "" {
		useTLS = true
	}
	remote := q.Get("remote")
	if remote == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("remote not set"))
		log.Errorf("HTTP SERVER: %v", "remote not set")
		return
	}
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Errorf("HTTP SERVER, WS Connection Upgrade: %v", err.Error())
		return
	}
	var remoteTCPConn net.Conn
	if useTLS {
		remoteTCPConn, err = tls.Dial("tcp", remote, &tls.Config{
			InsecureSkipVerify: true,
		})
	} else {
		remoteTCPConn, err = net.Dial("tcp", remote)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Errorf("HTTP SERVER, TCP Write: %v", err.Error())
		return
	}
	go h.handleClient(wsConn, remoteTCPConn)
}

func (h *socketHandler) handleClient(wsConn *websocket.Conn, remoteTCPConn net.Conn) {
	log.Infof("Start proxying traffic to %v on behalf of %v", remoteTCPConn.RemoteAddr(), wsConn.RemoteAddr())
	proxy.Copy(wsConn, remoteTCPConn)
	log.Infof("End proxying traffic to %v on behalf of %v", remoteTCPConn.RemoteAddr(), wsConn.RemoteAddr())
}
