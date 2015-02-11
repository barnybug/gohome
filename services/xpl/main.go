// Service to integrate XPL device on/off messages. Basic XPL support.
//
// For example, with the XPL plugin for Squeezebox this allows a squeezebox
// device to switch on your hifi when it is turned on.
package xpl

import (
	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"
	"log"
	"net"
	"regexp"
	"strings"
)

var re_parts = regexp.MustCompile(`(?s)([A-Za-z.-]+)\n{\n(.+?)\n}\n`)

func PairKeyValues(s string) map[string]string {
	ret := make(map[string]string)
	for _, pair := range strings.Split(s, "\n") {
		kv := strings.SplitN(pair, "=", 2)
		ret[kv[0]] = kv[1]
	}
	return ret
}

// Parse an XPL message.
func Parse(body string) map[string]map[string]string {
	parts := make(map[string]map[string]string)
	for _, m := range re_parts.FindAllStringSubmatch(body, -1) {
		k := m[1]
		v := m[2]
		parts[k] = PairKeyValues(v)
		// parts[k] = dict(l.split('=', 1) for l in  v.split('\n'))
	}
	return parts
}

// Process an XPL message.
func Process(body string) (string, string) {
	res := Parse(body)
	return res["xpl-stat"]["source"], res["audio.basic"]["POWER"]
}

type XplService struct {
}

func (self *XplService) Id() string {
	return "xpl"
}

func (self *XplService) Run() error {
	addr, err := net.ResolveUDPAddr("udp", ":3865")
	if err != nil {
		return err
	}
	sock, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	var buf [1024]byte
	for {
		rlen, _, err := sock.ReadFromUDP(buf[0:])
		if err != nil {
			log.Fatal(err)
			continue
		}
		data := string(buf[:rlen])
		log.Println("Received:", data)
		source, power := Process(data)
		if source != "" && power != "" {
			var command string
			switch power {
			case "1":
				command = "on"
			case "0":
				command = "off"
			}
			fields := map[string]interface{}{
				"origin":  "xpl",
				"command": command,
				"source":  source,
			}
			event := pubsub.NewEvent("xpl", fields)
			services.Publisher.Emit(event)
		}
	}
	return nil
}
