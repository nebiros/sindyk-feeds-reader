package reader

import (
	"fmt"
)

func ReadFeeds(f []Feed) {
	for _, e := range f {
		fmt.Printf("e, %T, %#v\n", e, e)
	}
}