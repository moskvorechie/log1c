package app

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/moskvorechie/logs"
	"gopkg.in/ini.v1"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

type Sender struct {
	wg     *sync.WaitGroup
	cfg    *ini.File
	exit   chan bool
	mess   chan Message
	logger logs.Log
}

type BasicAuthTransport struct {
	*http.Transport
	Username string
	Password string
}

func (t BasicAuthTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.SetBasicAuth(t.Username, t.Password)
	return t.Transport.RoundTrip(r)
}

func (s *Sender) Run(k int) {

	defer func() {
		if rc := recover(); rc != nil {
			s.logger.Error("Fatal stack: \n" + string(debug.Stack()))
			s.logger.FatalF("Recovered Fatal %v", rc)
		}
	}()

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: BasicAuthTransport{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Username: s.cfg.Section("elastic").Key("user").String(),
			Password: s.cfg.Section("elastic").Key("pass").String(),
		},
	}

	var req *http.Request
	var resp *http.Response

	defer s.wg.Done()

	s.logger.InfoF("Sender %d start", k)
	defer s.logger.InfoF("Sender %d stop", k)

	var count int64
	statuses := make(map[int]int64)

	for {
		select {
		case msg, ok := <-s.mess:
			if !ok {
				return
			}

			count++

			// Body
			body, err := json.Marshal(msg)
			if err != nil {
				s.logger.FatalF("%v", err)
			}

			index := "beat_log1c_" + msg.ДатаВремя.Format("2006.01")
			buf := bytes.NewBuffer(body)

			// Max attempt if lose connection
			for attempt := 0; ; attempt++ {

				if s.cfg.Section("main").Key("test").MustBool() {
					resp = &http.Response{
						StatusCode: http.StatusOK,
					}
					break
				}

				// Generate request
				uri := fmt.Sprintf("%s/%s/job/%s", s.cfg.Section("elastic").Key("url").String(), index, msg.ID)
				req, err = http.NewRequest("POST", uri, buf)
				if err != nil {
					s.logger.FatalF("%v", err)
				}
				req.Header.Set("Content-Type", "application/json")

				// Send request
				resp, err = client.Do(req)
				if resp == nil {
					s.logger.WarnF("Retry send: attempt %d | resp nil", attempt)
					time.Sleep(time.Duration(attempt*2) * time.Second)
					continue
				}
				resp.Body.Close()
				if err != nil || resp.StatusCode > 300 {
					if resp.StatusCode > 300 {
						s.logger.WarnF("%v", resp)
					}
					if attempt >= 10 {
						s.logger.ErrorF("Msg %+v", msg)
						s.logger.ErrorF("Uri %+v", uri)
						s.logger.Error("Max attempt to send message")
						break
					} else {
						s.logger.WarnF("Retry send: attempt %d | err %v", attempt, err)
						err = nil
						time.Sleep(time.Duration(attempt*2) * time.Second)
						continue
					}
				}
				break
			}

			s.logger.DebugF("Sent %s %d", msg.ID, resp.StatusCode)

			statuses[resp.StatusCode]++

			if count >= 100 {

				s.logger.InfoF("Sent 100 rows, statuses %v", statuses)

				count = 0
				statuses = make(map[int]int64)
			}
		case <-s.exit:
			return
		}
	}
}
