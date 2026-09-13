package tty

import (
	"fmt"
	"strings"
	"testing"
)

func TestZZUpNext(t *testing.T) {
	b := withBurst(t, bcWith(t, card(t, "a", "Oceanside, CA 92057")), "Severe Thunderstorm Warning · Harper, KS")
	b.width, b.height, b.ascii = 150, 74, true
	for _, l := range strings.Split(stripANSITest(strings.Join(b.readPair(), "\n")), "\n") {
		fmt.Println("|" + l + "|")
	}
}
