// 5 july 2014

/*
Package dl implements access to libdl, the library for loading dynamic modules on Unix systems.

It is not intended to provide a way to create dynamic modules in Go itself; it is merely provided to allow loading of pre-existing native modules, such as plugins for multimedia libraries, at runtime.

It is intended to be safe for concurrent use. (This is also why the package exists.)

Only features defined in the Single Unix Specification are supported.

This package cannot be used by itself, as the function pointers it returns are incompatible with Go. You will still need cgo.

Here is an example:

	package main
	import "fmt"
	import "github.com/andlabs/dl"
	// double callsqrt(void *p, double arg)
	// {
	// 	double (*f)(double);
	// 
	// 	*((void **) (&f)) = p;
	// 	return (*f)(arg);
	// }
	import "C"
	func main() {
		d, err := dl.Open("libm.so", dl.Lazy)
		if err != nil { panic(err) }
		defer d.Close()
		s, err := d.Symbol("sqrt")
		if err != nil { panic(err) }
		if s == nil {
			fmt.Println("no sqrt() in libm")
		} else {
			fmt.Println(C.callsqrt(s, 4))
		}
	}

(Note the bizarre cast to set the function pointer; C99 mandates this, and some versions of the libdl man pages also explain this.) See the man page for dlopen() on your local system (or online) for more details.
*/
package dl

import (
	"sync"
	"unsafe"
	"errors"
)

// #cgo LDFLAGS: -ldl
// #include <dlfcn.h>
// #include <stdlib.h>
import "C"

var dllock sync.Mutex

// Module represents a handle to an open library.
type Module uintptr

func dlerror() error {
	return errors.New(C.GoString(C.dlerror()))
}

// Mode represents a mode passed to Open().
type Mode uintptr
const (
	Now Mode = C.RTLD_NOW
	Lazy Mode = C.RTLD_LAZY
	Global Mode = C.RTLD_GLOBAL
	Local Mode = C.RTLD_LOCAL
)

// Note: the SUS does define RTLD_DEFAULT and RTLD_NOW as reserved for future use; while they do work in glibc, you need _GNU_SOURCE defined, so I won't include them.

// Open opens the named library, obeying the system's rule for absolute and relative library lookup.
// If the load fails, 0 is returned.
func Open(name string, mode Mode) (Module, error) {
	dllock.Lock()
	defer dllock.Unlock()

	C.dlerror()		// clear previous error state
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	m := C.dlopen(cname, C.int(mode))
	if m == nil {
		return 0, dlerror()
	}
	return Module(m), nil
}

// OpenSelf opens the current process.
// This is equivalent to calling dlopen() with a NULL filename.
// If the load fails, 0 is returned.
func OpenSelf(mode Mode) (Module, error) {
	dllock.Lock()
	defer dllock.Unlock()

	C.dlerror()		// clear previous error state
	m := C.dlopen(nil, C.int(mode))
	if m == nil {
		return 0, dlerror()
	}
	return Module(m), nil
}

// Close closes the Module.
// Symbols loaded from the Module should not be used after Close is called, even if there are other outstanding referneces to the dynamic library keeping it in memory.
func (m Module) Close() error {
	dllock.Lock()
	defer dllock.Unlock()

	C.dlerror()		// clear previous error state
	if C.dlclose(unsafe.Pointer(m)) != 0 {
		return dlerror()
	}
	return nil
}

// Symbol looks up the given named symbol in the Module.
// Note that the value of Symbol can be nil, so checking symbol for nil will not indicate an error; checking err for nil is.
func (m Module) Symbol(name string) (symbol unsafe.Pointer, err error) {
	dllock.Lock()
	defer dllock.Unlock()

	C.dlerror()		// clear previous error state
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	symbol = C.dlsym(unsafe.Pointer(m), cname)
	if symbol == nil {
		e := C.dlerror()
		if e == nil {		// no error; symbol value is NULL
			return nil, nil
		}
		return nil, errors.New(C.GoString(e))
	}
	return symbol, nil
}
