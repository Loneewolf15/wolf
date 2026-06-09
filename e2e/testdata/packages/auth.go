package packages

import "C"

//export Auth_Verify
func Auth_Verify() *C.char {
	return C.CString("hello from go interop")
}
