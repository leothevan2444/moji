package indexer

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

type bufferedConnection struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConnection) Read(data []byte) (int, error) { return c.reader.Read(data) }

// The HTTPS proxy and target use separate TLS names/trust. CONNECT addresses
// contain the already validated target IP, so a proxy cannot resolve an
// innocent public hostname again to a private destination.
func dialTLSProxy(ctx context.Context, proxy *url.URL, address string) (net.Conn, error) {
	proxyAddress := proxy.Host
	if proxy.Port() == "" {
		proxyAddress = net.JoinHostPort(proxy.Hostname(), "443")
	}
	connection, err := (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, "tcp", proxyAddress)
	if err != nil {
		return nil, ErrTransport
	}
	good := false
	defer func() {
		if !good {
			connection.Close()
		}
	}()
	secure := tls.Client(connection, &tls.Config{ServerName: proxy.Hostname(), MinVersion: tls.VersionTLS12})
	if err := secure.HandshakeContext(ctx); err != nil {
		return nil, ErrTransport
	}
	stop := context.AfterFunc(ctx, func() { connection.Close() })
	defer stop()
	if deadline, ok := ctx.Deadline(); ok {
		if err := connection.SetDeadline(deadline); err != nil {
			return nil, ErrTransport
		}
	}
	request := &http.Request{Method: http.MethodConnect, URL: &url.URL{Opaque: address}, Host: address, Header: http.Header{}}
	if proxy.User != nil {
		password, _ := proxy.User.Password()
		credentials := proxy.User.Username() + ":" + password
		request.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(credentials)))
	}
	if err := request.Write(secure); err != nil {
		return nil, ErrTransport
	}
	bounded := &io.LimitedReader{R: secure, N: (32 << 10) + 1}
	reader := bufio.NewReaderSize(bounded, 4096)
	response, err := http.ReadResponse(reader, request)
	if err != nil || bounded.N <= 0 {
		return nil, ErrTransport
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, ErrTransport
	}
	if err := connection.SetDeadline(time.Time{}); err != nil {
		return nil, ErrTransport
	}
	bounded.N = 1 << 62 // Only CONNECT headers use the small cap; preserve buffered bytes.
	good = true
	return &bufferedConnection{Conn: secure, reader: reader}, nil
}
