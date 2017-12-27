package boss

/*
#cgo CFLAGS: -O3
#cgo LDFLAGS: -lblosc
#include <blosc.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// Decompress takes a raw, blosc-compressed byte array and decompresses into a raw, uncompressed byte array
func Decompress(compressed []byte) ([]byte, error) {
	_, err := C.blosc_init()
	if err != nil {
		return []byte{}, err
	}
	defer C.blosc_destroy()

	nbytes := C.size_t(0)
	cbytes := C.size_t(0)
	blocksize := C.size_t(0)

	_, err = C.blosc_cbuffer_sizes(unsafe.Pointer(&compressed[0]), &nbytes, &cbytes, &blocksize)
	if err != nil {
		return []byte{}, err
	}

	uncompressed := make([]byte, int(nbytes))

	dsize, err := C.blosc_decompress(unsafe.Pointer(&compressed[0]), unsafe.Pointer(&uncompressed[0]), nbytes)
	if dsize < 0 {
		return []byte{}, fmt.Errorf("blosc: Decompression error with error code %d", dsize)
	}
	if err != nil {
		return []byte{}, err
	}

	return uncompressed, nil
}
