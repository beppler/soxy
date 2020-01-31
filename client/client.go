package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xandout/soxy/proxy"
)

// Start starts a soxy client
func Start(c *cli.Context) error {

	// Otains the websocket URL
	soxyURL, err := url.Parse(c.String("soxy-url"))
	if err != nil {
		log.Errorf("SOXY URL: %v", err.Error())
		return err
	}
	soxyURL.Path = soxyURL.Path + "/" // to keep compability with previous version
	query := soxyURL.Query()
	query.Set("remote", c.String("remote"))
	soxyURL.RawQuery = query.Encode()
	remote := soxyURL.String()

	headers := make(http.Header)
	apiKey := c.String("api-key")
	if apiKey != "" {
		headers.Set("X-Api-Key", c.String("api-key"))
	}

	if c.Bool("insecure") {
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	local := c.String("local")
	l, err := net.Listen("tcp", local)
	if err != nil {
		log.Errorf("TCP LISTENER: %v", err.Error())
		return err
	}
	// Close the listener when the application closes.
	defer l.Close()
	log.Infof("Listening on %v", local)

	for {
		// Listen for an incoming connection.
		tcpConn, err := l.Accept()
		if err != nil {
			log.Errorf("TCP ACCEPT: %v", err.Error())
			return err
		}

		// Handle connections in a new goroutine.
		go handleClient(tcpConn, remote, headers)
	}

}

func handleClient(tcpConn net.Conn, remote string, headers http.Header) {

	log.Infof("Dialing server %v", remote)
	clientWsConn, _, err := websocket.DefaultDialer.Dial(remote, headers)
	if err != nil {
		log.Errorf("DIALER: %v", err.Error())
		tcpConn.Close()
		return
	}
	log.Infof("Start proxying traffic to %v via %v for %v", remote, clientWsConn.RemoteAddr(), tcpConn.RemoteAddr())
	proxy.Copy(clientWsConn, tcpConn)
	log.Infof("End proxying traffic to %v via %v for %v", remote, clientWsConn.RemoteAddr(), tcpConn.RemoteAddr())

}
