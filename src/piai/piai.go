package piai

import (
	l4g "code.google.com/p/log4go"
	"github.com/NoahShen/go-cache"
	"github.com/NoahShen/go-simsimi"
	"time"
)

type PiAi struct {
	sessions *cache.Cache
}

func NewPiAi(sessionTimeout int64) *PiAi {
	ai := &PiAi{}
	ai.sessions = cache.New(time.Duration(sessionTimeout)*time.Second, 60*time.Second)
	return ai
}

func (self *PiAi) Talk(username, message string) (string, error) {
	var session *simsimi.SimSimiSession
	s, found := self.sessions.Get(username)
	if found {
		session = s.(*simsimi.SimSimiSession)
		self.sessions.Replace(username, session, 0) //reset the expiration
		l4g.Debug("Session found: %s", session.Id)
	} else {
		newSession, createErr := simsimi.CreateSimSimiSession(username)
		if createErr != nil {
			return "", createErr
		}
		l4g.Debug("Session created: %s", newSession.Id)
		self.sessions.Set(username, newSession, 0)
		session = newSession
	}
	return session.Talk(message)
}
