package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/xandout/soxy/proxy"
)

// Start starts a soxy client
func Start(c *cli.Context) error {

	l, err := net.Listen("tcp", c.String("local"))
	if err != nil {
		log.Errorf("TCP LISTENER: %v", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	log.Infof("Listening on %v", c.String("local"))

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
	log.Infof("Forwarding for %v", soxyURL)

	if c.Bool("insecure") {
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	for {
		// Listen for an incoming connection.
		tcpConn, err := l.Accept()
		if err != nil {
			log.Errorf("TCP ACCEPT: %v", err.Error())
			return err
		}

		headers := make(http.Header)
		apiKey := c.String("api-key")
		if apiKey != "" {
			headers.Set("X-Api-Key", apiKey)
		}

		clientWsConn, _, err := websocket.DefaultDialer.Dial(soxyURL.String(), headers)
		if err != nil {
			log.Errorf("DIALER: %v", err.Error())
			return err
		}
		// Handle connections in a new goroutine.
		log.Infof("Proxying traffic to %v via %v for %v", c.String("remote"), clientWsConn.RemoteAddr(), tcpConn.RemoteAddr())
		go proxy.Copy(clientWsConn, tcpConn)

	}

}
