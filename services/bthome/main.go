// Service to receive for bthome bluetooth LE devices.
package bthome

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

// Service bthome
type Service struct {
}

func (self *Service) ID() string {
	return "bthome"
}

type Scanner struct {
	listeners   map[string]bool
	stderr      bytes.Buffer
	cmd         *exec.Cmd
	terminating bool
	done        chan struct{}
}

func bthomeExe() string {
	ex, err := os.Executable()
	if err != nil {
		log.Fatalln("Couldn't get path of executable:", err)
	}
	return filepath.Join(filepath.Dir(ex), "bthome")
}

func (h *Scanner) run() {
	log.Println("Starting bthome scanner...")
	h.cmd = exec.Command("sudo", bthomeExe())
	stdout, err := h.cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to start bthome: %s", err)
	}
	stderr, err := h.cmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to start bthome: %s", err)
	}
	if err := h.cmd.Start(); err != nil {
		log.Fatalf("Failed to start bthome: %s", err)
	}

	go io.Copy(&h.stderr, stderr)
	h.scan(stdout)
	h.cmd.Wait()
	h.done <- struct{}{}
}

func (h *Scanner) launch() {
	go h.run()
}

func (h *Scanner) terminate() {
	// Send INT to whole process group (pid=0)
	// Note: the only clean way of stopping bthome is a SIGINT, any other signals
	// result in an unusabthome hci device requiring a down/up to reset.
	// Must sudo to kill the sudo'ed processes
	h.terminating = true
	cmd := exec.Command("sudo", "kill", "-INT", "0")
	cmd.Run()
}

func (h *Scanner) scan(stdout io.ReadCloser) {
	// read stdout by line, send an event for each line
	scanner := bufio.NewScanner(stdout)
	// drop first line
	scanner.Scan()
	for scanner.Scan() {
		data := scanner.Bytes()
		log.Printf("data: %s", string(data))
		if data[0] != '{' {
			log.Printf("bthome: %s", string(data))
			continue
		}
		var fields map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &fields); err != nil {
			log.Printf("Error parsing %s: %s", string(scanner.Bytes()), err)
			continue
		}
		topic := fields["topic"].(string)
		ev := pubsub.NewEvent(topic, fields)
		services.Config.AddDeviceToEvent(ev)
		services.Publisher.Emit(ev)
	}

	stderr := h.stderr.Bytes()
	if len(stderr) > 0 {
		log.Printf("bthome error: %s", string(stderr))
	}

	if err := scanner.Err(); err != nil {
		log.Printf("bthome failed: %s", err)
	} else {
		log.Printf("bthome exited")
	}
}

func (self *Service) Run() error {
	scanner := &Scanner{
		listeners: map[string]bool{},
		done:      make(chan struct{}),
	}
	scanner.launch()

	// Gracefully handle signals
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)

L:
	for {
		select {
		case <-sigC:
			break L
		case <-scanner.done:
			break L
		}
	}

	log.Println("Shutting down...")
	scanner.terminate()
	return nil
}
