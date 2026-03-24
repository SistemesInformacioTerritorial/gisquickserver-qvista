package ws

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var (
	ErrConnectionNotFound = errors.New("connection not found")
)

type message struct {
	Type   string      `json:"type"`
	Status int         `json:"status,omitempty"`
	Data   interface{} `json:"data"`
}

type messageEnvelope struct {
	Type string `json:"type"`
}

/* Structure for managing websocket connections for concurrent access */
type websocketsMap struct {
	sync.RWMutex
	name        string
	allowMany   bool
	connections map[string]map[*websocket.Conn]struct{}
	// jfs: I think this is a better approach
	writeMutex sync.Mutex
}

func (w *websocketsMap) Set(key string, conn *websocket.Conn) *websocket.Conn {
	w.Lock()
	defer w.Unlock()
	if conn == nil {
		delete(w.connections, key)
		return nil
	}
	if w.allowMany {
		if w.connections[key] == nil {
			w.connections[key] = make(map[*websocket.Conn]struct{})
		}
		w.connections[key][conn] = struct{}{}
		return nil
	}
	prev := w.firstLocked(key)
	w.connections[key] = map[*websocket.Conn]struct{}{conn: {}}
	return prev
}

func (w *websocketsMap) Get(key string) *websocket.Conn {
	w.RLock()
	defer w.RUnlock()
	return w.firstLocked(key)
}

func (w *websocketsMap) GetAll(key string) []*websocket.Conn {
	w.RLock()
	defer w.RUnlock()
	return w.getAllLocked(key)
}

func (w *websocketsMap) DeleteIfMatch(key string, conn *websocket.Conn) bool {
	w.Lock()
	defer w.Unlock()
	conns := w.connections[key]
	if len(conns) == 0 {
		return false
	}
	if _, ok := conns[conn]; !ok {
		return false
	}
	delete(conns, conn)
	if len(conns) == 0 {
		delete(w.connections, key)
	}
	return true
}

func (w *websocketsMap) firstLocked(key string) *websocket.Conn {
	for conn := range w.connections[key] {
		return conn
	}
	return nil
}

func (w *websocketsMap) getAllLocked(key string) []*websocket.Conn {
	conns := w.connections[key]
	if len(conns) == 0 {
		return nil
	}
	data := make([]*websocket.Conn, 0, len(conns))
	for conn := range conns {
		data = append(data, conn)
	}
	return data
}

// func (w *websocketsMap) Send(key string, msg message) error {
// 	dest := w.Get(key)
// 	if dest != nil {
// 		return dest.WriteJSON(msg)
// 	}
// 	return ErrConnectionNotFound
// }

/* COmmented out the above function and added the below function
// jfs: I think this is a better approach
func (w *websocketsMap) Send(key string, msgType string, data interface{}) error {
	dest := w.Get(key)
	if dest != nil {
		return dest.WriteJSON(message{Type: msgType, Data: data})
	}
	// return ErrConnectionNotFound // probably for MustSend variant
	return nil
}
*/
// / jfs: I think this is a better approach
func (w *websocketsMap) Send(key string, msgType string, data interface{}) error {
	dests := w.GetAll(key)
	if len(dests) > 0 {
		msg := message{Type: msgType, Data: data}
		var sendErr error
		for _, dest := range dests {
			if err := w.writeJSON(dest, msg); err != nil && sendErr == nil {
				sendErr = err
			}
		}
		return sendErr
	}
	return ErrConnectionNotFound
}

func (w *websocketsMap) writeJSON(conn *websocket.Conn, msg message) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(msg)
}

func (w *websocketsMap) writeMessage(conn *websocket.Conn, msgType int, payload []byte) error {
	w.writeMutex.Lock()
	defer w.writeMutex.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteMessage(msgType, payload)
}

func parseMessageType(payload []byte) string {
	var envelope messageEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return ""
	}
	return envelope.Type
}

func isExpectedCloseError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "closed network connection")
}

type SettingsWS struct {
	log      *zap.SugaredLogger
	upgrader websocket.Upgrader
	plugin   *websocketsMap
	webapp   *websocketsMap
}

func NewSettingsWS(log *zap.SugaredLogger) *SettingsWS {
	return &SettingsWS{
		log: log,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		plugin: &websocketsMap{name: "plugin", connections: make(map[string]map[*websocket.Conn]struct{})},
		webapp: &websocketsMap{name: "webapp", allowMany: true, connections: make(map[string]map[*websocket.Conn]struct{})},
	}
}

func (s *SettingsWS) AppChannel() *websocketsMap {
	return s.webapp
}

// func (s *SettingsWS) SendToPlugin(id string, msgType string, data interface{}) error {
// 	dest := s.plugin.Get(id)
// 	if dest != nil {
// 		msg := message{Type: msgType, Data: data}
// 		return dest.WriteJSON(msg)
// 	}
// 	return nil
// }

func (s *SettingsWS) bridgeHandler(id string, src *websocketsMap, dest *websocketsMap, w http.ResponseWriter, r *http.Request) (err error) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Errorw("failed to upgrade websocket connection", "user", id, "channel", src.name, zap.Error(err))
		return
	}
	if prev := src.Set(id, conn); prev != nil && prev != conn {
		s.log.Infow("closing replaced websocket connection", "user", id, "channel", src.name)
		_ = prev.Close()
	}
	defer func() {
		_ = conn.Close()
		removed := src.DeleteIfMatch(id, conn)
		s.log.Infow("websocket connection closed", "user", id, "channel", src.name, "removed", removed)
		for _, destConn := range dest.GetAll(id) {
			if err := dest.writeJSON(destConn, message{Type: "PluginStatus", Status: 503}); err != nil {
				s.log.Errorw("failed to send PluginStatus message", "user", id, "channel", dest.name, zap.Error(err))
			}
		}
	}()
	s.log.Infow("websocket connection started", "user", id, "channel", src.name)
	if src.name == "plugin" {
		for _, destConn := range dest.GetAll(id) {
			info := map[string]string{"client": r.Header.Get("User-Agent")}
			if err := dest.writeJSON(destConn, message{Type: "PluginStatus", Status: 200, Data: info}); err != nil {
				s.log.Errorw("failed to send PluginStatus message", "user", id, "channel", dest.name, zap.Error(err))
			}
		}
	} else if pluginConn := dest.Get(id); pluginConn != nil {
		info := map[string]string{"client": r.Header.Get("User-Agent")}
		if err := src.writeJSON(conn, message{Type: "PluginStatus", Status: 200, Data: info}); err != nil {
			s.log.Errorw("failed to send PluginStatus message", "user", id, "channel", src.name, zap.Error(err))
		}
	}
	for {
		msgType, msg, rerr := conn.ReadMessage()
		if rerr != nil {
			if !websocket.IsCloseError(rerr, websocket.CloseNormalClosure, websocket.CloseGoingAway) && !isExpectedCloseError(rerr) {
				err = rerr
				s.log.Errorw("websocket error", "user", id, "channel", src.name, zap.Error(rerr))
			}
			break
		}
		if bytes.Compare(msg, []byte("Ping")) == 0 {
			s.log.Debugw("received Ping message", "user", id, "channel", src.name)
			continue
		}

		if msgType == websocket.TextMessage {
			messageType := parseMessageType(msg)
			if src.name == "webapp" && messageType == "PluginStatus" {
				s.log.Debugw("dropping webapp plugin status probe", "user", id)
				continue
			}
			if messageType != "" {
				s.log.Debugw("received TextMessage", "user", id, "channel", src.name, "type", messageType)
			} else {
				s.log.Debugw("received TextMessage", "user", id, "channel", src.name, "message", string(msg))
			}
			if dests := dest.GetAll(id); len(dests) > 0 {
				if messageType != "" {
					s.log.Debugw("forwarding message", "user", id, "channel", dest.name, "type", messageType)
				} else {
					s.log.Debugw("forwarding message", "user", id, "channel", dest.name, "message", string(msg))
				}
				for _, destConn := range dests {
					if err = dest.writeMessage(destConn, msgType, msg); err != nil {
						s.log.Errorw("failed to forward message", "user", id, "channel", dest.name, zap.Error(err))
						break
					}
				}
				if err != nil {
					break
				}
			} else {
				if src.name == "plugin" && messageType == "PluginStatus" {
					s.log.Debugw("dropping plugin status without webapp connection", "user", id)
					continue
				}
				s.log.Warnw("destination connection not found", "user", id, "channel", dest.name)
				if writeErr := src.writeJSON(conn, message{Type: "PluginStatus", Status: 503}); writeErr != nil {
					s.log.Errorw("failed to send PluginStatus message", "user", id, "channel", src.name, zap.Error(writeErr))
				}
			}
		} else if msgType == websocket.CloseMessage {
			s.log.Infow("websocket CloseMessage", "user", id, "channel", src.name)
			break
		}
	}
	return
}

func (s *SettingsWS) WebAppHandler(id string, w http.ResponseWriter, r *http.Request) error {
	return s.bridgeHandler(id, s.webapp, s.plugin, w, r)
}

func (s *SettingsWS) PluginHandler(id string, w http.ResponseWriter, r *http.Request) error {
	return s.bridgeHandler(id, s.plugin, s.webapp, w, r)
}

/*
type BridgeConnection struct {
	id     string
	wsconn *websocket.Conn
	pool   *websocketsMap
}

func (c *BridgeConnection) SendMessage(msgType string, data interface{}) {
	if destConn := c.pool.Get(c.id); destConn != nil {
		destConn.WriteJSON(message{Type: msgType, Data: data, Status: 200})
	} else {
		c.wsconn.WriteJSON(message{Type: "PluginStatus", Status: 503}) // rename to TargetStatus or ReceiverStatus
	}
}

func (c *BridgeConnection) Forward() {
	for {
		// Read message from source connection
		msgType, msg, err := c.wsconn.ReadMessage()
		if err != nil {
			// log.Println(err)
			break
		}
		// msgType == websocket.PingMessage
		if bytes.Compare(msg, []byte("Ping")) == 0 {
			continue
		}

		if msgType == websocket.TextMessage {
			if destConn := c.pool.Get(c.id); destConn != nil {
				if err = destConn.WriteMessage(msgType, msg); err != nil {
					break // or better reply with error message?
				}
			} else {
				c.wsconn.WriteJSON(message{Type: "PluginStatus", Status: 503}) // rename to TargetStatus or ReceiverStatus
			}
		} else if msgType == websocket.CloseMessage {
			break
		}
	}
}
*/
