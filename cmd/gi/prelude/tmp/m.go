package main

import "fmt"

func main() {
	var a [5][5]float64
	a[4][4] = 1.2
	fmt.Printf("a[4][4] = %v\n", a[4][4])
}
