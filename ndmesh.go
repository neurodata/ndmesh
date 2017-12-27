package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	// "github.com/neurodata/ndmesh/boss"
	"./boss"
	// "github.com/neurodata/ndmesh/mesh"
	"./mesh"
)

const (
	bossVersion = "v1"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type meshExtractionInfo struct {
	chunkID   int
	numMeshes int
	err       error
}

type bossExtractionInfo struct {
	collection string
	experiment string
	channel    string
	xres       float32
	yres       float32
	zres       float32
}

type cutoutInfo struct {
	x   [2]int
	y   [2]int
	z   [2]int
	res int
}

func meshExtraction(c chan meshExtractionInfo, boss bossExtractionInfo, cutout cutoutInfo, path string, prefix string, chunkID int, bossServer *boss.Info) {
	var ret meshExtractionInfo

	cutoutData, err := bossServer.Cutout(boss.collection, boss.experiment, boss.channel, cutout.x, cutout.y, cutout.z, cutout.res)
	if err != nil {
		ret.err = err
		c <- ret
		return
	}

	xoffset := cutout.x[0]
	yoffset := cutout.y[0]
	zoffset := cutout.z[0]

	xsize := cutout.x[1] - cutout.x[0]
	ysize := cutout.y[1] - cutout.y[0]
	zsize := cutout.z[1] - cutout.z[0]

	numMeshes, err := mesh.ExtractMesh(cutoutData, chunkID, fmt.Sprintf("%s/%s.%d", path, prefix, chunkID), xsize, ysize, zsize, xoffset, yoffset, zoffset, boss.xres, boss.yres, boss.zres)
	if err != nil {
		ret.err = err
		c <- ret
		return
	}

	ret.numMeshes = numMeshes
	ret.chunkID = chunkID
	c <- ret

	return
}

// NeuroglancerManifest describes the file format expected by Neuroglancer to determine all mesh files that make up a single object
type NeuroglancerManifest struct {
	Fragments []string `json:"fragments"`
}

func exportNeuroglancerManifest(meshes map[int][]string, path string) error {

	/* Neuroglancer Manifest Format:
	 * Filename: <<<Object ID>>>
	 * Body: {"fragments": ["mesh.<<<Chunk ID>>>.<<<Object ID>>>", "mesh.<<<Chunk ID>>>.<<<Object ID>>>"...]}
	 * Note that the path in the body is relative to the root of the webserver that hosts the mesh files. For a S3 bucket, the path should include any folders between the mesh file and the bucket root.

	 */
	for objID, meshFiles := range meshes {
		var manifest NeuroglancerManifest
		for _, fragmentPath := range meshFiles {
			fragmentPathSplit := strings.Split(fragmentPath, "/")
			manifest.Fragments = append(manifest.Fragments, fragmentPathSplit[len(fragmentPathSplit)-1])
		}
		manifestJSON, err := json.Marshal(manifest)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(fmt.Sprintf("%s/%d", path, objID), manifestJSON, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {

	var token = flag.String("token", "", "Boss API token.")
	var path = flag.String("path", "", "Directory to use for output files.")
	var prefix = flag.String("prefix", "mesh", "Prefix for Neuroglancer mesh files.")
	var hostname = flag.String("hostname", "api.boss.neurodata.io", "Boss Server Hostname.")
	var collection = flag.String("collection", "", "Boss collection.")
	var experiment = flag.String("experiment", "", "Boss experiment.")
	var channel = flag.String("channel", "", "Boss channel.")
	var xoffset = flag.Int("xoffset", 0, "The x-offset of the cutout")
	var yoffset = flag.Int("yoffset", 0, "The y-offset of the cutout")
	var zoffset = flag.Int("zoffset", 0, "The z-offset of the cutout")
	var xsize = flag.Int("xsize", 0, "The x-size of the cutout")
	var ysize = flag.Int("ysize", 0, "The y-size of the cutout")
	var zsize = flag.Int("zsize", 0, "The z-size of the cutout")
	var xstride = flag.Int("xstride", 0, "The size of the stride in the x dimension")
	var ystride = flag.Int("ystride", 0, "The size of the stride in the y dimension")
	var zstride = flag.Int("zstride", 0, "The size of the stride in the z dimension")
	var resolution = flag.Int("res", 0, "The resolution of the cutout")
	flag.Parse()

	if len(*token) == 0 {
		panic(errors.New("boss API Token is required"))
	}
	if len(*path) == 0 {
		panic(errors.New("directory path for output files is required"))
	}

	bossServer := boss.Info{*hostname, *token, bossVersion}

	exp, err := bossServer.GetExperimentInfo(*collection, *experiment)
	check(err)
	coord, err := bossServer.GetCoordFrame(exp.CoordFrame)
	check(err)

	// Check dimensions
	if *xoffset < coord.Xstart || (*xoffset+*xsize) > coord.Xstop {
		panic(errors.New("invalid x coordinate range"))
	}
	if *yoffset < coord.Ystart || (*yoffset+*ysize) > coord.Ystop {
		panic(errors.New("invalid y coordinate range"))
	}
	if *zoffset < coord.Zstart || (*zoffset+*zsize) > coord.Zstop {
		panic(errors.New("invalid z coordinate range"))
	}

	bossExtractionInfo := bossExtractionInfo{*collection, *experiment, *channel, coord.Xvoxelsize, coord.Yvoxelsize, coord.Zvoxelsize}
	// TODO(adb): if res > 0 we will need to adjust the voxel res
	if *resolution != 0 {
		panic(errors.New("unable to generate meshes for res > 0"))
	}

	ch := make(chan meshExtractionInfo)

	var ctr int
	ctr = 0
	for x := *xoffset; x < *xoffset+*xsize; x += *xstride {
		for y := *yoffset; y < *yoffset+*ysize; y += *ystride {
			for z := *zoffset; z < *zoffset+*zsize; z += *zstride {
				var xstart, ystart, zstart int
				var xend, yend, zend int

				if (x - 1) < *xoffset {
					xstart = *xoffset
				} else {
					xstart = x - 1
				}
				if (y - 1) < *yoffset {
					ystart = *yoffset
				} else {
					ystart = y - 1
				}
				if (z - 1) < *zoffset {
					zstart = *zoffset
				} else {
					zstart = z - 1
				}

				if (xstart + *xstride) < *xoffset+*xsize {
					xend = xstart + *xstride
				} else {
					xend = *xoffset + *xsize
				}
				if (ystart + *ystride) < *yoffset+*ysize {
					yend = ystart + *ystride
				} else {
					yend = *yoffset + *ysize
				}
				if (zstart + *zstride) < *zoffset+*zsize {
					zend = zstart + *zstride
				} else {
					zend = *zoffset + *zsize
				}

				cutout := cutoutInfo{[2]int{xstart, xend}, [2]int{ystart, yend}, [2]int{zstart, zend}, *resolution}
				fmt.Printf("%d %d %d -> %d %d %d\n", xstart, ystart, zstart, xend, yend, zend)
				go meshExtraction(ch, bossExtractionInfo, cutout, *path, *prefix, ctr, &bossServer)
				ctr++
			}
		}
	}

	fmt.Printf("Started %d threads for extraction.\n", ctr)

	// var globalMeshMap = make(map[int][]string)

	numMeshes := 0
	for i := 0; i < ctr; i++ {
		t := <-ch
		if t.err != nil {
			panic(t.err)
		}
		numMeshes += t.numMeshes
	}
	meshes, err := mesh.GetMeshes(ctr)
	check(err)

	numMeshesTmp := 0
	for _, v := range meshes {
		numMeshesTmp += len(v)
	}
	if numMeshes != numMeshesTmp {
		panic(fmt.Errorf("ndmesh: number of extracted meshes (%d) does not match number of meshes read post extraction (%d). Possible synchronization error", numMeshes, numMeshesTmp))
	}

	err = exportNeuroglancerManifest(meshes, *path)
	check(err)

	fmt.Printf("Done: Extracted %d meshes.\n", numMeshes)

}
