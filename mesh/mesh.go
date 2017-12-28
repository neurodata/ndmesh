package mesh

/*
#include <Mesh/ExtractNeuroglancerMesh.h>
#cgo CFLAGS: -O3 -I/usr/local/include/ndm
#cgo LDFLAGS: -L/usr/local/lib/ndm -lMesh -lneuroglancer_mesh -lboost_iostreams -lboost_system -lz -lglog -lstdc++ -lm -fopenmp
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

// MeshMap defines the mapping from mesh ID -> filename for each chunk processed
type MeshMap map[int]string

var (
	meshSyncMap sync.Map // We use sync.Map to allow for each chunk to maintain its own MeshMap
)

// GetMeshes returns the entries in the globally-scoped meshes map.
func GetMeshes(ctr int) (map[int][]string, error) {
	var meshMap = make(map[int][]string)
	for i := 0; i < ctr; i++ {
		meshMapInterface, ok := meshSyncMap.Load(i)
		if ok == false {
			continue
		}
		meshMapIn, ok := meshMapInterface.(MeshMap)
		if ok == false {
			return map[int][]string{}, fmt.Errorf("mesh: unknown return type for chunk %d", ctr)
		}
		for k, v := range meshMapIn {
			meshMap[k] = append(meshMap[k], v)
		}
	}
	return meshMap, nil
}

//export meshExtractedCallbackGo
func meshExtractedCallbackGo(filename *C.char, id C.int, chunkID C.int) {
	var meshes MeshMap
	_chunkID := int(chunkID)
	meshesInterface, ok := meshSyncMap.Load(_chunkID)
	if ok == false {
		meshes = make(MeshMap)

	} else {
		meshes, ok = meshesInterface.(MeshMap)
		if ok == false {
			panic("fatal: Unable to read MeshMap from synchronized Map for mesh extraction.")
		}
	}
	meshes[int(id)] = C.GoString(filename)
	meshSyncMap.Store(_chunkID, meshes)
}

// ExtractMesh takes an input uint64_t byte array and extracts all mesh objects, saving the resulting meshes in neuroglancer format. Note that the filePathPrefix should include a sequence number if extracting from multiple chunks in the same dataset.
func ExtractMesh(input []byte, chunkID int, filePathPrefix string, xdim int, ydim int, zdim int, xoffset int, yoffset int, zoffset int, xres float32, yres float32, zres float32) (int, error) {

	var length int
	length = xdim * ydim * zdim

	// Convert from 64-bit array to 32-bit array
	convertedInput := make([]byte, xdim*ydim*zdim*4)
	_, err := C.ConvertArray64to32Bit(unsafe.Pointer(&input[0]), unsafe.Pointer(&convertedInput[0]), C.int(length))
	if err != nil {
		return 0, err
	}

	var numMeshes C.int
	_, err = C.ExtractNeuroglancerMeshFromChunk(unsafe.Pointer(&convertedInput[0]), C.CString(filePathPrefix), C.int(xdim), C.int(ydim), C.int(zdim), C.int(xoffset), C.int(yoffset), C.int(zoffset), C.float(xres), C.float(yres), C.float(zres), C.int(chunkID), &numMeshes)
	if err != nil {
		return 0, err
	}

	return int(numMeshes), nil
}
