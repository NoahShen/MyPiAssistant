package xmpp

import (
	"strings"
)

func ToBareJID(jid string) string {
	i := strings.Index(jid, "/")
	if i < 0 {
		return jid
	}
	bareJid := jid[0:i]
	return bareJid
}
