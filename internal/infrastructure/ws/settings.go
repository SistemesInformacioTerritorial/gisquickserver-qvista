package ws

import (
	"bytes"
	"errors"
	"net/http"
	"sync"

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

/* Structure for managing websocket connections for concurrent access */
type websocketsMap struct {
	sync.RWMutex
	name        string
	connections map[string]*websocket.Conn
	// jfs: I think this is a better approach
	writeMutex sync.Mutex
}

func (w *websocketsMap) Set(key string, conn *websocket.Conn) {
	w.Lock()
	defer w.Unlock()
	// TODO: is it better to replace connection or return error?
	if conn == nil {
		delete(w.connections, key)
	} else {
		w.connections[key] = conn
	}
}

func (w *websocketsMap) Get(key string) *websocket.Conn {
	w.RLock()
	defer w.RUnlock()
	return w.connections[key]
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
	dest := w.Get(key)
	if dest != nil {
		w.writeMutex.Lock()
		defer w.writeMutex.Unlock()
		return dest.WriteJSON(message{Type: msgType, Data: data})
	}
	return ErrConnectionNotFound
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
		plugin: &websocketsMap{name: "plugin", connections: make(map[string]*websocket.Conn)},
		webapp: &websocketsMap{name: "webapp", connections: make(map[string]*websocket.Conn)},
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
	src.Set(id, conn)
	s.log.Infow("websocket connection started", "user", id, "channel", src.name)
	if destConn := dest.Get(id); destConn != nil {
		info := map[string]string{"client": r.Header.Get("User-Agent")}
		if err := destConn.WriteJSON(message{Type: "PluginStatus", Status: 200, Data: info}); err != nil {
			s.log.Errorw("failed to send PluginStatus message", "user", id, "channel", dest.name, zap.Error(err))
		}
	}
	for {
		msgType, msg, rerr := conn.ReadMessage()
		if rerr != nil {
			if !websocket.IsCloseError(rerr, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
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
			s.log.Debugw("received TextMessage", "user", id, "channel", src.name, "message", string(msg))
			if destConn := dest.Get(id); destConn != nil {
				//jfs: I think this is a better approach
				s.log.Debugw("forwarding message", "user", id, "channel", dest.name, "message", string(msg))
				dest.writeMutex.Lock()
				s.log.Debugw("writeMutex locked", "user", id, "channel", dest.name)
				//
				if err = destConn.WriteMessage(msgType, msg); err != nil {
					s.log.Errorw("failed to forward message", "user", id, "channel", dest.name, zap.Error(err))
					break
				}

				//jfs: I think this is a better approach
				dest.writeMutex.Unlock()
				s.log.Debugw("writeMutex unlocked", "user", id, "channel", dest.name)
				//
			} else {
				s.log.Warnw("destination connection not found", "user", id, "channel", dest.name)
				conn.WriteJSON(message{Type: "PluginStatus", Status: 503}) // rename to TargetStatus or ReceiverStatus
			}
		} else if msgType == websocket.CloseMessage {
			s.log.Infow("websocket CloseMessage", "user", id, "channel", src.name)
			break
		}
	}
	src.Set(id, nil)
	s.log.Infow("websocket connection closed", "user", id, "channel", src.name)
	if destConn := dest.Get(id); destConn != nil {
		if err := destConn.WriteJSON(message{Type: "PluginStatus", Status: 503}); err != nil {
			s.log.Errorw("failed to send PluginStatus message", "user", id, "channel", dest.name, zap.Error(err))
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
