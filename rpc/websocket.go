package rpc

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"storage-mining/log"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

const (
	wsBuffRead     = 1024
	wsBuffWrite    = 1024
	wsMsgSizeLimit = 8 * 1024 * 1024

	wsHeartbeatInterval = 120 * time.Second
	wsWriteTimeout      = 10 * time.Second
	wsReadTimeout       = 10 * time.Second
)

var wsBufferPool = new(sync.Pool)

func (s *Server) WebsocketHandler(allowedOrigins []string) http.Handler {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  wsBuffRead,
		WriteBufferSize: wsBuffWrite,
		WriteBufferPool: wsBufferPool,
		CheckOrigin:     wsHandshakeValidator(allowedOrigins),
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Debug("WebSocket upgrade failed", "err", err)
			return
		}
		codec := newWebsocketCodec(conn, r.Host, r.Header)
		s.serve(codec)
	})
}

func wsHandshakeValidator(allowedOrigins []string) func(*http.Request) bool {
	origins := mapset.NewSet()
	allowAllOrigins := false

	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAllOrigins = true
		}
		if origin != "" {
			origins.Add(origin)
		}
	}
	// allow localhost if no allowedOrigins are specified.
	if len(origins.ToSlice()) == 0 {
		origins.Add("http://localhost")
		if hostname, err := os.Hostname(); err == nil {
			origins.Add("http://" + hostname)
		}
	}
	log.Debug(fmt.Sprintf("Allowed origin(s) for WS RPC interface %v", origins.ToSlice()))

	f := func(req *http.Request) bool {
		// Skip origin verification if no Origin header is present. The origin check
		// is supposed to protect against browser based attacks. Browsers always set
		// Origin. Non-browser software can put anything in origin and checking it doesn't
		// provide additional security.
		if _, ok := req.Header["Origin"]; !ok {
			return true
		}
		// Verify origin against allow list.
		origin := strings.ToLower(req.Header.Get("Origin"))
		if allowAllOrigins || originIsAllowed(origins, origin) {
			return true
		}
		log.Warn("Rejected WebSocket connection", "origin", origin)
		return false
	}

	return f
}

func originIsAllowed(allowedOrigins mapset.Set, browserOrigin string) bool {
	it := allowedOrigins.Iterator()
	for origin := range it.C {
		if ruleAllowsOrigin(origin.(string), browserOrigin) {
			return true
		}
	}
	return false
}

func ruleAllowsOrigin(allowedOrigin string, browserOrigin string) bool {
	var (
		allowedScheme, allowedHostname, allowedPort string
		browserScheme, browserHostname, browserPort string
		err                                         error
	)
	allowedScheme, allowedHostname, allowedPort, err = parseOriginURL(allowedOrigin)
	if err != nil {
		log.Warn("Error parsing allowed origin specification", "spec", allowedOrigin, "error", err)
		return false
	}
	browserScheme, browserHostname, browserPort, err = parseOriginURL(browserOrigin)
	if err != nil {
		log.Warn("Error parsing browser 'Origin' field", "Origin", browserOrigin, "error", err)
		return false
	}
	if allowedScheme != "" && allowedScheme != browserScheme {
		return false
	}
	if allowedHostname != "" && allowedHostname != browserHostname {
		return false
	}
	if allowedPort != "" && allowedPort != browserPort {
		return false
	}
	return true
}

func parseOriginURL(origin string) (string, string, string, error) {
	parsedURL, err := url.Parse(strings.ToLower(origin))
	if err != nil {
		return "", "", "", err
	}
	var scheme, hostname, port string
	if strings.Contains(origin, "://") {
		scheme = parsedURL.Scheme
		hostname = parsedURL.Hostname()
		port = parsedURL.Port()
	} else {
		scheme = ""
		hostname = parsedURL.Scheme
		port = parsedURL.Opaque
		if hostname == "" {
			hostname = origin
		}
	}
	return scheme, hostname, port, nil
}

func DialWebsocket(ctx context.Context, endpoint, origin string) (*Client, error) {
	dialer := websocket.Dialer{
		ReadBufferSize:  wsBuffRead,
		WriteBufferSize: wsBuffWrite,
		WriteBufferPool: wsBufferPool,
	}
	return DialWebsocketWithDialer(ctx, endpoint, origin, dialer)
}

func wsClientHeaders(endpoint, origin string) (string, http.Header, error) {
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return endpoint, nil, err
	}
	header := make(http.Header)
	if origin != "" {
		header.Add("origin", origin)
	}
	if endpointURL.User != nil {
		b64auth := base64.StdEncoding.EncodeToString([]byte(endpointURL.User.String()))
		header.Add("authorization", "Basic "+b64auth)
		endpointURL.User = nil
	}
	return endpointURL.String(), header, nil
}

func DialWebsocketWithDialer(ctx context.Context, endpoint, origin string, dialer websocket.Dialer) (*Client, error) {
	endpoint, header, err := wsClientHeaders(endpoint, origin)
	if err != nil {
		return nil, err
	}
	conn, _, err := dialer.DialContext(ctx, endpoint, header)
	if err != nil {
		return nil, err
	}
	codec := newWebsocketCodec(conn, endpoint, header)
	return newClient(codec), nil
}

type websocketCodec struct {
	*protoCodec

	mu        sync.Mutex
	wg        sync.WaitGroup
	pingReset chan struct{}
}

func newWebsocketCodec(conn *websocket.Conn, host string, req http.Header) *websocketCodec {
	conn.SetReadLimit(wsMsgSizeLimit)
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Time{})
		return nil
	})
	wc := &websocketCodec{
		protoCodec: &protoCodec{
			conn:     conn,
			closedCh: make(chan struct{}),
		},
		pingReset: make(chan struct{}, 1),
	}

	// Start pinger.
	wc.wg.Add(1)
	go wc.pingLoop()
	return wc
}

func (w *websocketCodec) Close() {
	w.close()
	w.wg.Wait()
}

func (w *websocketCodec) WriteMsg(ctx context.Context, v proto.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(wsWriteTimeout)
	}
	w.getConn().SetWriteDeadline(deadline)
	err := w.write(v)
	if err == nil {
		// Notify pingLoop to delay the next idle ping.
		select {
		case w.pingReset <- struct{}{}:
		default:
		}
	}
	return err
}

func (w *websocketCodec) pingLoop() {
	var timer = time.NewTimer(wsHeartbeatInterval)
	defer w.wg.Done()
	defer timer.Stop()

	for {
		select {
		case <-w.closed():
			return
		case <-w.pingReset:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(wsHeartbeatInterval)
		case <-timer.C:
			w.mu.Lock()
			w.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			w.conn.WriteMessage(websocket.PingMessage, nil)
			w.conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
			w.mu.Unlock()
			timer.Reset(wsHeartbeatInterval)
		}
	}
}
