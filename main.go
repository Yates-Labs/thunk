package main

import "fmt"

func main() {
	fmt.Println("Hello World!")

	temp := SampleMethod(5, 3)
	fmt.Println(temp)
}

func SampleMethod(x int, y int) int {
	return x + y
}
