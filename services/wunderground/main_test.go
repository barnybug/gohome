package wunderground

import (
	"fmt"
	"time"

	"github.com/barnybug/gohome/config"
)

func ExampleRequest() {
	conf := config.WundergroundConf{
		Url:      "http://weatherstation.wunderground.com/weatherstation/updateweatherstation.php",
		Id:       "ID",
		Password: "pass",
	}
	w := NewWunderground(conf)
	w.Add(Metrics{"temp": 15.0})
	now := time.Date(2014, 1, 2, 3, 4, 5, 0, time.UTC)
	uri := w.RequestUri(now)
	fmt.Println(uri)
	// Output:
	// http://weatherstation.wunderground.com/weatherstation/updateweatherstation.php?ID=ID&PASSWORD=pass&action=updateraw&dateutc=2014-01-02+03%3A04%3A05&softwaretype=gohome&temp=15
}
