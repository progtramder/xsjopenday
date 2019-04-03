package main

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

func init() {
	go dbRoutine()
}

type chanHandler interface {
	handle()
}

//channel 的缓冲大小直接影响响应性能，可以根据情况调节缓冲大小
var dbChannel = make(chan chanHandler, 20000)

func dbRoutine() {
	for {
		handler := <-dbChannel
		handler.handle()
	}
}

type chanRegister struct {
	event  string
	openId string
	bm     bminfo
}

func register(event, openId, student, gender, phone, category string, session int) {
	dbChannel <- &chanRegister{
		event,
		openId,
		bminfo{student, gender, phone, category, session},
	}
}

func (self *chanRegister) handle() {
	bmEvent := bmEventList.GetEvent(self.event)
	if bmEvent != nil {
		bmEvent.report.register(self.openId,
			self.bm.student, self.bm.gender, self.bm.phone, self.bm.category,
			bmEvent.sessions[self.bm.session].Desc)
	}
}

var client = &http.Client{}

func GetOpenId(code string) (openId string) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		privateData.AppId, privateData.AppSecret, code)
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	type retJson struct {
		OpenId string `json:"openid"`
	}

	rj := retJson{}
	json.Unmarshal(content, &rj)
	openId = rj.OpenId
	return
}

const (
	errSuccess = iota
	errRepeat
	errNotStarted
	errReachLimit
	errInvalidSession
)

func Reason(errCode int) string {
	switch errCode {
	case errSuccess:
		return "报名成功"
	case errRepeat:
		return "重复报名"
	case errNotStarted:
		return "报名未开始"
	case errReachLimit:
		return "已报满"
	case errInvalidSession:
		return "场次错误"
	default:
		return "未知错误"
	}
}

type bminfo struct {
	student  string
	gender   string
	phone    string
	category string
	session  int
}

type BMEvent struct {
	sync.RWMutex
	started  bool
	report   *excel
	name     string
	endTime  time.Time
	webpage  string
	sessions []Session
	bm       map[string]bminfo
}

func (self *BMEvent) put(token, student, gender, phone, category string, session int) int {
	self.Lock()
	defer self.Unlock()
	if !self.started {
		return errNotStarted
	}
	for k, v := range self.bm {
		if k == token || v.phone == phone {
			return errRepeat
		}
	}
	if session >= len(self.sessions) {
		return errInvalidSession
	}
	s := &self.sessions[session]
	if s.number >= s.Limit {
		return errReachLimit
	}

	s.number++
	self.bm[token] = bminfo{student, gender, phone, category, session}
	return errSuccess
}

func (self *BMEvent) has(token string) (bminfo, bool) {
	self.RLock()
	v, ok := self.bm[token]
	self.RUnlock()
	return v, ok
}

type Session struct {
	Desc   string `yaml:"description"`
	Limit  int    `yaml:"limit"`
	Extra  bool   `yaml:"extra"`
	number int
}

type Event struct {
	Event    string    `yaml:"event"`
	EndTime  string    `yaml:"endtime"`
	WebPage  string    `yaml:"webpage"`
	Sessions []Session `yaml:"sessions"`
}

func (self *BMEvent) Expired() bool {
	return time.Now().After(self.endTime)
}

func (self *BMEvent) Init(e Event) error {

	tm, err := parseTime(e.EndTime)
	if err != nil {
		return errors.New(fmt.Sprintf("事件结束时间 %s %s", e.EndTime, err.Error()))
	}

	report, err := InitReport(e.Event)
	if err != nil {
		return err
	}

	self.started = false
	self.name = e.Event
	self.endTime = tm
	self.webpage = e.WebPage
	self.sessions = e.Sessions
	self.report = report
	self.bm = map[string]bminfo{}

	return nil
}

func (self *BMEvent) Start() {
	self.Lock()
	self.started = true
	self.Unlock()
}

type BMEventList struct {
	sync.RWMutex
	events []*BMEvent
}

func (self *BMEventList) Reset() error {
	path := systembasePath + "/event.yaml"
	setting, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	eventList := struct {
		Events []Event `yaml:"events"`
	}{}

	err = yaml.Unmarshal(setting, &eventList)
	if err != nil {
		return err
	}

	self.Lock()
	defer self.Unlock()
	oldEvents := self.events
	self.events = make([]*BMEvent, len(eventList.Events))

	if oldEvents == nil {
		//cold reset
		for i := range self.events {
			bmEvent := &BMEvent{}
			if err := bmEvent.Init(eventList.Events[i]); err != nil {
				return err
			}
			self.events[i] = bmEvent
		}
	} else {
		//hot reset, we reuse the old event object if it is not expired and
		//it's name mathces that in config file
		match := func(name string) int {
			for i, v := range oldEvents {
				if v.name == name && !v.Expired() {
					return i
				}
			}
			return -1
		}

		for i, v := range eventList.Events {
			j := match(v.Event)
			if j == -1 {
				bmEvent := &BMEvent{}
				if err := bmEvent.Init(v); err != nil {
					return err
				}
				self.events[i] = bmEvent
			} else {
				self.events[i] = oldEvents[j]
			}
		}
	}

	return nil
}

func (self *BMEventList) GetEvent(name string) *BMEvent {
	self.RLock()
	defer self.RUnlock()
	for _, v := range self.events {
		if v.name == name {
			return v
		}
	}

	return nil
}

var bmEventList = &BMEventList{}
