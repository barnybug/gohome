// Service to detect bluetooth LE beacons.
package ble

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

const interval = 30 * time.Second

// Service ble
type Service struct {
}

func (self *Service) ID() string {
	return "ble"
}

func emit(mac string) {
	fields := pubsub.Fields{
		"mac": mac,
	}
	ev := pubsub.NewEvent("beacon", fields)
	services.Publisher.Emit(ev)
}

type Hcitool struct {
	listeners map[string]bool
	stderr    bytes.Buffer
	cmd       *exec.Cmd
}

func (h *Hcitool) launch() {
	h.cmd = exec.Command("sudo", "stdbuf", "-oL", "hcitool", "lescan", "--passive", "--duplicates")
	stdout, err := h.cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to start hcitool: %s", err)
		return
	}
	stderr, err := h.cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to start hcitool: %s", err)
		return
	}
	if err := h.cmd.Start(); err != nil {
		log.Fatalf("Failed to start hcitool: %s", err)
		return
	}

	go io.Copy(&h.stderr, stderr)
	go h.scan(stdout)
}

func (h *Hcitool) terminate() {
	h.cmd.Wait()
	log.Println("Terminated hcitool")
}

func (h *Hcitool) scan(stdout io.ReadCloser) {
	// read stdout by line, send an event for each line
	scanner := bufio.NewScanner(stdout)
	// drop first line
	scanner.Scan()
	for scanner.Scan() {
		line := scanner.Text()
		ps := strings.SplitN(line, " ", 2)
		mac := ps[0]
		if _, ok := h.listeners[mac]; ok {
			emit(mac)
		}
	}

	stderr := h.stderr.Bytes()
	if len(stderr) > 0 {
		log.Printf("hcitool error: %s", string(stderr))
	}

	if err := scanner.Err(); err != nil {
		log.Printf("hcitool failed: %s", err)
	} else {
		log.Printf("hcitool exited: bluetooth monitoring disabled")
	}
}

func (self *Service) shutdown() {
	log.Println("Shutting down...")
	// Send INT to whole process group (pid=0)
	// Note: the only clean way of stopping hcitool is a SIGINT, any other signals
	// result in an unusable hci device requiring a down/up to reset.
	// Must sudo to kill the sudo'ed processes
	cmd := exec.Command("sudo", "kill", "-INT", "0")
	cmd.Run()
	log.Println("Shut down complete")
}

func (self *Service) Run() error {
	hcitool := &Hcitool{
		listeners: map[string]bool{},
	}
	for mac, _ := range services.Config.Protocols["ble"] {
		log.Printf("Scanning bluetooth %s (passive)", mac)
		hcitool.listeners[mac] = true
	}
	hcitool.launch()

	// Gracefully handle signals
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)

L:
	for {
		select {
		case <-sigC:
			break L
		}
	}

	self.shutdown()
	hcitool.terminate()
	return nil
}
