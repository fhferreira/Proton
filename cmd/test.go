package main

import (
	"log"
	"time"

	"github.com/mkevac/gopinba"
)

func main() {
	pc, err := gopinba.NewClient("127.0.0.1:30002")
	if err != nil {
		log.Fatalf("NewClient() returned error: %v", err)
	}
	var (
		os      = []string{"Windows", "Linux", "Mac OS"}
		device  = []string{"Mobile", "Desktop", "TV"}
		browser = []string{"Chrome", "FF"}
	)
	for i := 0; i < 2; i++ {
		req := gopinba.Request{
			Tags: map[string]string{
				"OS":      os[i%len(os)],
				"Device":  device[i%len(device)],
				"Browser": browser[i%len(browser)],
			},
		}
		req.Hostname = "hostname"
		req.ServerName = "servername"
		req.ScriptName = "scriptname"
		req.Schema = "https"
		req.Status = 200
		req.RequestCount = 1
		req.RequestTime = 145987 * time.Microsecond
		req.DocumentSize = 1024
		{
			timer := gopinba.TimerStart(map[string]string{"mysql": "server_1", "db_connect": "server_1"})
			time.Sleep(time.Millisecond * 5)
			timer.Stop()
			req.AddTimer(timer)
		}
		{
			timer := gopinba.TimerStart(map[string]string{"mysql": "server_1", "db_connect": "server_1"})
			time.Sleep(time.Millisecond * 5)
			timer.Stop()
			req.AddTimer(timer)
		}
		{
			timer := gopinba.TimerStart(map[string]string{"mysql": "server_1", "postgresql": "server_1"})
			time.Sleep(time.Millisecond * 5)
			timer.Stop()
			req.AddTimer(timer)
		}
		err = pc.SendRequest(&req)
		if err != nil {
			log.Fatalf("SendRequest() returned error: %v", err)
		}
	}
}
