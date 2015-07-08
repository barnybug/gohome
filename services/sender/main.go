// Service to emit test events received on stdin. Use simply for testing.
package sender

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
)

// Service sender
type Service struct{}

func (self *Service) ID() string {
	return "sender"
}

func (self *Service) Run() error {
	b := bufio.NewScanner(os.Stdin)
	for b.Scan() {
		ev := pubsub.Parse(b.Text())
		if ev != nil {
			fmt.Println(ev)
			services.Publisher.Emit(ev)
		} else {
			fmt.Println("Parse failed")
		}
	}

	// give it time to send
	time.Sleep(time.Duration(500) * time.Millisecond)
	return nil
}
