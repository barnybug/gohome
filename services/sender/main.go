// Service to emit test events received on stdin. Use simply for testing.
package sender

import (
	"bufio"
	"fmt"
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"os"
	"time"
)

type SenderService struct{}

func (self *SenderService) Id() string {
	return "sender"
}

func (self *SenderService) Run() error {
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
