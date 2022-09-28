package datatrack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	jvbuster "github.com/Connect-Club/connectclub-jvbuster-client"
	"github.com/Connect-Club/connectclub-mobile-common/volatile"
	log "github.com/sirupsen/logrus"
	"nhooyr.io/websocket"
	"reflect"
	"sync"
	"time"
)

type ConnectionState int32

const (
	ConnectionStateUndefined ConnectionState = iota
	ConnectionStateConnecting
	ConnectionStateConnected
	ConnectionStateClosed
)

func (s ConnectionState) String() string {
	return [...]string{
		"Undefined",
		"Connecting",
		"Connected",
		"Closed",
	}[s]
}

type Connection struct {
	url                           string
	roomId, roomPass, accessToken string
	conn                          *websocket.Conn
	connWriteMu                   sync.Mutex
	connectTask                   *jvbuster.Task
	cancelPing                    context.CancelFunc
	state                         *volatile.Value[ConnectionState]
	UserId                        string
	onUserId                      func(userId string, userType ClientType)
	onState                       func(newState ConnectionState)
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 5 * time.Second

	// Send pings to peer with this period.
	pingInterval = 10 * time.Second

	pingLatency = 2 * time.Second
)

var UnknownMsgTypeErr = errors.New("unknown message type")
var Closed = errors.New("datatrack connection is closed")
var NotConnected = errors.New("datatrack connection is not connected")

func NewConnection(url string, roomId, roomPass, accessToken string, onUserId func(userId string, userType ClientType), onState func(newState ConnectionState)) *Connection {
	c := &Connection{
		url:         url,
		roomId:      roomId,
		roomPass:    roomPass,
		accessToken: accessToken,
		connectTask: nil, //init later
		cancelPing:  func() {},
		state:       volatile.NewValue(ConnectionStateUndefined),
		onUserId:    onUserId,
		onState:     onState,
	}
	c.connectTask = jvbuster.CreateTask(c.connect, 0, false)
	if err := c.connectTask.SyncRun(time.Second * 10); err != nil {
		panic(err.Error())
	}
	return c
}

func (c *Connection) callOnUserId(userId string, userType ClientType) {
	if c.onUserId != nil {
		go c.onUserId(userId, userType)
	}
}

func (c *Connection) callOnState(newState ConnectionState) {
	if c.onState != nil {
		go c.onState(newState)
	}
}

func (c *Connection) Read() (typedMessage interface{}, rawMessage []byte, preparsedMessage map[string]json.RawMessage, err error) {
	_, rawMessage, err = c.read()
	if err != nil {
		return
	}
	typedMessage, preparsedMessage, err = extractPayload(rawMessage)
	return
}

func (c *Connection) SendPath(fromX float64, fromY float64, toX float64, toY float64, pathType string) {
	err := c.writeMsg(&PathRequestMessage{
		CurrentX: fromX,
		CurrentY: fromY,
		NextX:    toX,
		NextY:    toY,
		Type:     pathType,
	})
	if err != nil {
		log.WithError(err).Warn("can not send PathRequestMessage")
	}
}

func (c *Connection) SendState(isVideoEnabled bool, isAudioEnabled bool, isOnPhoneCall bool) {
	err := c.writeMsg(&StateRequestMessage{
		Video:     isVideoEnabled,
		Audio:     isAudioEnabled,
		PhoneCall: isOnPhoneCall,
		Jitsi:     true, //todo: remove when fix datatrack
	})
	if err != nil {
		log.WithError(err).Warn("can not send StateRequestMessage")
	}
}

func (c *Connection) SendAudioLevel(value int) {
	err := c.writeMsg(&AudioLevelRequestMessage{Value: value})
	if err != nil {
		log.WithError(err).Warn("can not send AudioLevelRequestMessage")
	}
}

func (c *Connection) SendMessage(message string) {
	err := c.write(websocket.MessageText, []byte(message))
	if err != nil {
		log.WithError(err).Warn("can not send message")
	}
}

func (c *Connection) internalClose() {
	c.cancelPing()
	if c.conn != nil {
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
	}
}

func (c *Connection) Close() {
	log.Info("⤵")
	defer log.Info("⤴")

	if err := c.connectTask.Stop(time.Second * 10); err != nil {
		log.WithError(err).Error("cannot stop connectTask")
	}
	if c.state.Swap(ConnectionStateClosed) == ConnectionStateClosed {
		return
	}

	c.internalClose()

	c.callOnState(ConnectionStateClosed)
}

func (c *Connection) writeMsg(msg RequestMessage) error {
	data, err := json.Marshal(&messageToWrite{Type: msg.typ(), Payload: msg})
	if err != nil {
		return fmt.Errorf("can not marshal messageToWrite, err = %w", err)
	}
	return c.write(websocket.MessageText, data)
}

func (c *Connection) connect(ctx context.Context) {
	log.Info("connect begin")
	defer log.Info("connect end")

	c.state.Store(ConnectionStateConnecting)
	c.callOnState(ConnectionStateConnecting)

	defer func() {
		if ctx.Err() == nil {
			c.state.Store(ConnectionStateConnected)
			c.callOnState(ConnectionStateConnected)
		}
	}()

	c.internalClose()

connectionCycle:
	for firstCycle := true; ; firstCycle = false {
		if !firstCycle {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond * 500):
			}
		}

		log.Info("dial websocket")
		conn, _, err := websocket.Dial(ctx, c.url, nil)
		if err != nil {
			log.WithError(err).Warn("cannot dial websocket")
			continue connectionCycle
		}
		log.Info("websocket connected")
		reqMsg := &RegisterRequestMessage{
			SID:         c.roomId,
			Password:    c.roomPass,
			AccessToken: c.accessToken,
			Version:     5,
		}

		if writer, err := conn.Writer(ctx, websocket.MessageText); err != nil {
			_ = conn.Close(websocket.StatusAbnormalClosure, err.Error())
			log.WithError(err).Warn("cannot get writer")
			continue connectionCycle
		} else if err := json.NewEncoder(writer).Encode(&messageToWrite{Type: reqMsg.typ(), Payload: reqMsg}); err != nil {
			_ = writer.Close()
			_ = conn.Close(websocket.StatusAbnormalClosure, err.Error())
			log.WithError(err).Warn("cannot write(encode) RegisterRequestMessage")
			continue connectionCycle
		} else if err := writer.Close(); err != nil {
			log.WithError(err).Warn("cannot write(close) RegisterRequestMessage")
			continue connectionCycle
		}

		log.Info("websocket RegisterRequestMessage sent")

		var registerResponseMessage *RegisterResponseMessage
		for {
			_, msg, err := conn.Read(ctx)
			log.Info("websocket RegisterRequestMessage read")
			if err != nil {
				log.WithError(err).Warn("cannot read RegisterResponseMessage")
				_ = conn.Close(websocket.StatusAbnormalClosure, err.Error())
				continue connectionCycle
			}

			registerPayload, _, err := extractPayload(msg)
			if err != nil {
				log.WithError(err).Panic("cannot extract payload")
			}

			var ok bool
			registerResponseMessage, ok = registerPayload.(*RegisterResponseMessage)
			if !ok {
				log.Infof("expected RegisterResponseMessage, got %v", reflect.TypeOf(registerPayload))
				continue
			}
			break
		}

		log.Info("websocket RegisterResponseMessage received")

		if c.UserId != registerResponseMessage.ID {
			c.UserId = registerResponseMessage.ID
			c.callOnUserId(registerResponseMessage.ID, registerResponseMessage.Type)
		}

		c.conn = conn
		go c.ping(ctx)
		break
	}
	log.Info("websocket ready")
	defer func() {
		go c.SendPath(0, 0, 0, 0, "spawn")
	}()
}

func (c *Connection) ping(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c.cancelPing = cancel

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if c.state.Load() != ConnectionStateConnected {
				return
			}
			ctx, cancel := context.WithTimeout(ctx, pingLatency)
			if err := c.conn.Ping(ctx); err != nil {
				log.WithError(err).Warn("ping")
			}
			cancel()
		case <-ctx.Done():
			return
		}
	}
}

func (c *Connection) read() (messageType websocket.MessageType, p []byte, err error) {
	state := c.state.Load()
	if state == ConnectionStateClosed {
		return messageType, p, Closed
	} else if state != ConnectionStateConnected {
		return messageType, p, NotConnected
	}

	messageType, p, err = c.conn.Read(context.Background())

	if err != nil {
		log.WithError(err).Info("cannot read message, trying to reconnect")
		c.connectTask.Run()
	}
	return
}

func (c *Connection) write(messageType websocket.MessageType, data []byte) (err error) {
	state := c.state.Load()
	if state == ConnectionStateClosed {
		return Closed
	} else if state != ConnectionStateConnected {
		return NotConnected
	}

	c.connWriteMu.Lock()
	defer c.connWriteMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), writeWait)
	err = c.conn.Write(ctx, messageType, data)
	cancel()

	if err != nil {
		log.WithError(err).Info("cannot write message, trying to reconnect")
		c.connectTask.Run()
	}
	return
}

type messageToWrite struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func extractPayload(msg []byte) (interface{}, map[string]json.RawMessage, error) {
	var jsonMsg map[string]json.RawMessage
	if err := json.Unmarshal(msg, &jsonMsg); err != nil {
		return nil, nil, fmt.Errorf("can not extract payload from not a json message, msg = %v, err = %w", string(msg), err)
	}

	if msgPayload, ok := jsonMsg["payload"]; ok {
		if rawMsgType, ok := jsonMsg["type"]; ok {
			var msgType string
			if err := json.Unmarshal(rawMsgType, &msgType); err != nil {
				return nil, jsonMsg, fmt.Errorf("can not unmarshal type, err = %w", err)
			}
			payload := typeFactory(msgType)
			if payload == nil {
				return nil, jsonMsg, UnknownMsgTypeErr
			}
			if err := json.Unmarshal(msgPayload, &payload); err != nil {
				return nil, jsonMsg, fmt.Errorf("can not unmarshal payload to type = %v, err = %w", string(rawMsgType), err)
			}
			return payload, jsonMsg, nil
		} else {
			return nil, jsonMsg, errors.New("can not extract payload from message without type")
		}
	}
	return nil, jsonMsg, fmt.Errorf("can not extract payload from msg = %v", string(msg))
}

func typeFactory(typ string) (msg interface{}) {
	switch typ {
	case "register":
		msg = &RegisterResponseMessage{}
	case "path":
		msg = &PathResponseMessage{}
	case "speaker":
		msg = &SpeakerResponseMessage{}
	case "radarVolume":
		msg = &RadarVolumeResponseMessage{}
	case "broadcasted":
		msg = &BroadcastedResponseMessage{}
	case "connectionState":
		msg = &ConnectionStateResponseMessage{}
	case "state":
		msg = &StateResponseMessage{}
	default:
		log.Warnf("do not know how to handle datatrack type = '%v'", typ)
	}
	return
}
