package testplugin

import "C"

//export MyGoFunc
func MyGoFunc(a int, b int) int {
	return a + b + 100
}
