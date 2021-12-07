package netutils

import (
	"time"
)

func timeIsPast(t time.Time) bool {
	if t.IsZero() {
		return false
	}

	return time.Now().After(t)
}

func timeAfter(t time.Time) <-chan time.Time {
	if t.IsZero() {
		return nil
	}

	return time.After(time.Until(t))
}

func chanClose(c chan interface{}) bool {
	select {
	case <-c:
		return false
	default:
		close(c)
		return true
	}
}
