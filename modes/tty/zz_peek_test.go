package tty

import (
	"fmt"
	"testing"
)

func TestZZPeek(t *testing.T) {
	d := openMap(t, Config{MapFeed: viewFeed}, 133, 44)
	fmt.Printf("view %+v given %d\n", d.viewBox(d.mapBodySize()), len(d.mapPane.given))
	for id := range d.mapPane.given {
		fmt.Println(" ", id)
	}
	fmt.Println(boxText(d))
}
