package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const defaultTimeout = 10 * time.Second

type Dialer struct {
	URL                *url.URL
	InsecureSkipVerify bool
}

func NewDialer(u *url.URL, insecure bool) *Dialer {
	return &Dialer{URL: u, InsecureSkipVerify: insecure}
}

func proxyPort(u *url.URL) string {
	if u.Port() != "" {
		return u.Port()
	}
	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}
	return "80"
}

func (d *Dialer) DialContext(ctx context.Context, network, target string) (net.Conn, error) {
	if d == nil || d.URL == nil {
		nd := net.Dialer{Timeout: defaultTimeout}
		return nd.DialContext(ctx, "tcp", target)
	}
	if strings.EqualFold(d.URL.Scheme, "socks5") {
		return d.dialSOCKS5(target)
	}
	return d.dialConnect(target)
}

func (d *Dialer) dialSOCKS5(target string) (net.Conn, error) {
	proxyAddr := d.URL.Host
	if d.URL.Port() == "" {
		proxyAddr = net.JoinHostPort(d.URL.Hostname(), "1080")
	}
	var auth *proxy.Auth
	if d.URL.User != nil {
		pw, _ := d.URL.User.Password()
		auth = &proxy.Auth{User: d.URL.User.Username(), Password: pw}
	}
	sd, err := proxy.SOCKS5("tcp", proxyAddr, auth, &net.Dialer{Timeout: defaultTimeout})
	if err != nil {
		return nil, fmt.Errorf("proxy: socks5 setup %s: %w", proxyAddr, err)
	}
	conn, err := sd.Dial("tcp", target)
	if err != nil {
		return nil, fmt.Errorf("proxy: socks5 dial %s: %w", target, err)
	}
	return conn, nil
}

func (d *Dialer) dialConnect(target string) (net.Conn, error) {
	proxyAddr := net.JoinHostPort(d.URL.Hostname(), proxyPort(d.URL))

	var conn net.Conn
	var err error
	if strings.EqualFold(d.URL.Scheme, "https") {
		td := &tls.Dialer{
			NetDialer: &net.Dialer{Timeout: defaultTimeout},
			Config:    &tls.Config{ServerName: d.URL.Hostname(), InsecureSkipVerify: d.InsecureSkipVerify},
		}
		conn, err = td.DialContext(context.Background(), "tcp", proxyAddr)
	} else {
		nd := net.Dialer{Timeout: defaultTimeout}
		conn, err = nd.DialContext(context.Background(), "tcp", proxyAddr)
	}
	if err != nil {
		return nil, fmt.Errorf("proxy: dial %s: %w", proxyAddr, err)
	}

	req := strings.Builder{}
	req.WriteString("CONNECT " + target + " HTTP/1.1\r\n")
	req.WriteString("Host: " + target + "\r\n")
	if d.URL.User != nil {
		if up := d.URL.User.String(); up != "" {
			enc := base64.StdEncoding.EncodeToString([]byte(up))
			req.WriteString("Proxy-Authorization: Basic " + enc + "\r\n")
		}
	}
	req.WriteString("\r\n")

	if _, err := conn.Write([]byte(req.String())); err != nil {
		conn.Close()
		return nil, fmt.Errorf("proxy: CONNECT write: %w", err)
	}

	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("proxy: CONNECT read: %w", err)
	}
	fields := strings.Fields(statusLine)
	if len(fields) < 2 {
		conn.Close()
		return nil, fmt.Errorf("proxy: bad CONNECT response %q", statusLine)
	}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("proxy: CONNECT headers: %w", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}
	if fields[1] != "200" {
		conn.Close()
		return nil, fmt.Errorf("proxy: CONNECT refused (%s)", strings.TrimSpace(statusLine))
	}

	return &tunnelConn{conn: conn, br: br}, nil
}

func (d *Dialer) Dial(target string) (net.Conn, error) {
	return d.DialContext(context.Background(), "tcp", target)
}

type tunnelConn struct {
	conn net.Conn
	br   *bufio.Reader
}

func (t *tunnelConn) Read(b []byte) (int, error)  { return t.br.Read(b) }
func (t *tunnelConn) Write(b []byte) (int, error) { return t.conn.Write(b) }
func (t *tunnelConn) Close() error                { return t.conn.Close() }
func (t *tunnelConn) LocalAddr() net.Addr         { return t.conn.LocalAddr() }
func (t *tunnelConn) RemoteAddr() net.Addr        { return t.conn.RemoteAddr() }
func (t *tunnelConn) SetDeadline(tm time.Time) error {
	return t.conn.SetDeadline(tm)
}
func (t *tunnelConn) SetReadDeadline(tm time.Time) error {
	return t.conn.SetReadDeadline(tm)
}
func (t *tunnelConn) SetWriteDeadline(tm time.Time) error {
	return t.conn.SetWriteDeadline(tm)
}

func (d *Dialer) HTTPTransport() *http.Transport {
	return &http.Transport{DialContext: d.DialContext}
}

func (d *Dialer) WSDialer() *websocket.Dialer {
	return &websocket.Dialer{NetDialContext: d.DialContext}
}

func Connect(proxyURL *url.URL, target string) (net.Conn, error) {
	return NewDialer(proxyURL, false).Dial(target)
}
