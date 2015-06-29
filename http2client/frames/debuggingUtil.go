package frames

import "fmt"

func printBytes(name string, data []byte) {
	fmt.Printf("%v:", name)
	for _, b := range data {
		fmt.Printf(" %v", b)
	}
	fmt.Printf("\n")
}
