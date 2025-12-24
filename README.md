# pbot
This is pbot. A Botnet variant for educational purposes. Fully made in Go.

## Features

- **Attack vectors**: Multiple attack vectors to use, including UDP, TCP and ICMP.
- **Fully in Go**: Made in Go, for the best performance.
- **Malware**: Use build.go to build the binaries for the malware. If the malware file is ran, it will connect to the server and be infected.

## Getting Started

### Prerequisitesa

- **Go** (v3.10.0 or later recommended)
- **INFO** Please change the IP in client.go
### Installation

1. **Download Go repositories:**
    ```bash
    go mod download
    ```

2. **Build the files:**
    ```bash
    go run build.go
    ```

### Usage
1. **Run the server**:
    ```bash
    ./server
    ```
- **Now you can infect the bots**

# Also, when you run the client (the malware) you can run it with a parameter as bot group. As an example: ./x86 test, it will be ran as bot group "test".

## Available commands
- **!stats** - Get stats of the botnet | Cores, Ram, etc.
- **!kill [group]**  - Kills the bots of a group.
- **!bots** - Shows all bots
- **?** - Shows all methods and commands.
- **!reboot** - Reboots all bots. 
