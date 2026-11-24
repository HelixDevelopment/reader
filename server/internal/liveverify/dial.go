package liveverify

import (
	"net"
	"time"
)

func netDialer() net.Dialer { return net.Dialer{Timeout: 5 * time.Second} }
