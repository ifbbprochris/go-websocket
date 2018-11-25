package impl

import (
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"sync"
)

type Connection struct {
	wsConn *websocket.Conn
	inChan chan []byte
	outChan chan []byte
	closeChan chan []byte
	isClosed bool
	mutex sync.Mutex
}

func InitConnection(wsConn *websocket.Conn) (conn *Connection, err error) {
	conn = &Connection{
		wsConn:wsConn,
		inChan: make(chan []byte, 1000),
		outChan: make(chan []byte, 1000),
		closeChan: make(chan []byte, 1),
	}

	//启动读协程
	go conn.readLoop()
	go conn.writeLoop()
	return
}

func(conn *Connection) ReadMessage() (data []byte, err error) {
	select {
	case data = <- conn.inChan:
		return data ,nil
	case <- conn.closeChan:
		err = errors.New("connection is closed")
	}

	return nil, errors.New("websocket closed")
}

func(conn *Connection) WriteMessage(data []byte) (err error) {
	select {
	case conn.outChan <- data:
	case <- conn.closeChan:
		err = errors.New("connection is closed")
	}
	return nil
}

func(conn *Connection) Close() {
	conn.wsConn.Close()
	conn.mutex.Lock()
	if !conn.isClosed {
		close(conn.closeChan)
		conn.isClosed = true
	}
	conn.mutex.Unlock()
}

func(conn *Connection) readLoop() {
	var (
		data []byte
		err error
	)
	for {
		if _, data, err = conn.wsConn.ReadMessage(); err != nil {
			goto ERR
		}

		select {
		case conn.inChan <- data:
		case <- conn.closeChan:
			goto ERR
		}
		
	}
	ERR:
		conn.Close()
}

func (conn *Connection) writeLoop()  {
	var (
		data []byte
		err error
	)
	for {
		select {
		case data = <- conn.outChan:
		case <- conn.closeChan:
			goto ERR
		}

		if err = conn.wsConn.WriteMessage(websocket.TextMessage,data); err != nil {
			goto ERR
		}
	}
	ERR:
		conn.Close()
}