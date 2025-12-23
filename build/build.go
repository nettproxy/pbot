package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	targets := []struct {
		GOOS   string
		GOARCH string
		Output string
	}{
		{"linux", "386", "./bins/x86"},
	}

	for _, t := range targets {
		fmt.Printf("Building %s (%s/%s)...\n", t.Output, t.GOOS, t.GOARCH)

		cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", t.Output, "client.go")
		env := append(os.Environ(), "GOOS="+t.GOOS, "GOARCH="+t.GOARCH, "CGO_ENABLED=0")
		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Printf("Failed to build %s: %v", t.Output, err)
			continue
		}

		fmt.Printf("Built %s\n", t.Output)

		upxCmd := exec.Command("upx", "--ultra", "--best", t.Output)
		if err := upxCmd.Run(); err == nil {
			fmt.Printf("Compressed %s with UPX\n", t.Output)
		}
	}
	fmt.Println("Building server..")
	cmd := exec.Command("go", "build", "-o", "./server", "server.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Failed %s: %v", "./server", err)
	}
}
