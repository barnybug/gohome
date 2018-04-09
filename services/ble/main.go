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
	listeners   map[string]bool
	stderr      bytes.Buffer
	cmd         *exec.Cmd
	terminating bool
	done        chan struct{}
}

func (h *Hcitool) run() {
	for retries := 0; retries < 3 && !h.terminating; retries += 1 {
		log.Println("Starting hcitool...")
		h.cmd = exec.Command("sudo", "stdbuf", "-oL", "hcitool", "lescan", "--passive", "--duplicates")
		stdout, err := h.cmd.StdoutPipe()
		if err != nil {
			log.Fatalf("Failed to start hcitool: %s", err)
		}
		stderr, err := h.cmd.StderrPipe()
		if err != nil {
			log.Fatalf("Failed to start hcitool: %s", err)
		}
		if err := h.cmd.Start(); err != nil {
			log.Fatalf("Failed to start hcitool: %s", err)
		}

		go io.Copy(&h.stderr, stderr)
		h.scan(stdout)
		h.cmd.Wait()
	}
	h.done <- struct{}{}
}

func (h *Hcitool) launch() {
	go h.run()
}

func (h *Hcitool) terminate() {
	// Send INT to whole process group (pid=0)
	// Note: the only clean way of stopping hcitool is a SIGINT, any other signals
	// result in an unusable hci device requiring a down/up to reset.
	// Must sudo to kill the sudo'ed processes
	h.terminating = true
	cmd := exec.Command("sudo", "kill", "-INT", "0")
	cmd.Run()
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
		log.Printf("hcitool exited")
	}
}

func (self *Service) Run() error {
	hcitool := &Hcitool{
		listeners: map[string]bool{},
		done:      make(chan struct{}),
	}
	for _, dev := range services.Config.DevicesByProtocol("ble") {
		mac := dev.SourceId()
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
		case <-hcitool.done:
			break L
		}
	}

	log.Println("Shutting down...")
	hcitool.terminate()
	return nil
}
