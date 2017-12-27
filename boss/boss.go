package boss

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// Boss is a interface to access Boss APIs
type Boss interface {
	GetExperimentInfo(collection string, experiment string) (e Experiment, err error)
	GetCoordFrame(coordFrameName string) (c CoordinateFrame, err error)
	formatServerUrl() string
	Cutout(collection string, experiment string, channel string, xrng [2]int, yrng [2]int, zrng [2]int, resolution int) ([]byte, error)
}

// Info describes the authentication and access information for a particular boss instance
type Info struct {
	Hostname string
	Token    string
	Version  string
}

// Experiment describes the metadata corresponding to a boss experiment
type Experiment struct {
	Channels           []string `json:"channels"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Collection         string   `json:"collection"`
	CoordFrame         string   `json:"coord_frame"`
	NumHierarchyLevels int      `json:"num_hierarchy_levels"`
	HierarchyMethod    string   `json:"hierarchy_method"`
	NumTimeSamples     int      `json:"num_time_samples"`
	TimeStep           string   `json:"tine_step"`
	TimeStepUnit       string   `json:"time_step_unit"`
	Creator            string   `json:"creator"`
}

func (b Info) getServerURL() string {
	var version string
	if len(b.Version) > 0 {
		version = b.Version
	} else {
		version = "latest"
	}
	return fmt.Sprintf("https://%s/%s", b.Hostname, version)
}

// GetExperimentInfo returns the experiment info for a given experiment nested in a collection
func (b Info) GetExperimentInfo(collection string, experiment string) (e Experiment, err error) {
	client := &http.Client{}

	serverURL := b.getServerURL()
	URL := fmt.Sprintf("%s/collection/%s/experiment/%s/", serverURL, collection, experiment)
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return Experiment{}, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Token %s", b.Token))

	resp, err := client.Do(req)
	if err != nil {
		return Experiment{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Experiment{}, err
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	for {
		if err := dec.Decode(&e); err == io.EOF {
			break
		} else if err != nil {
			return Experiment{}, err
		}
		return e, nil
	}

	return Experiment{}, errors.New("experiment: Unable to parse JSON experiment information")
}

// CoordinateFrame describes the metadata corresponding to a boss coordinate frame
type CoordinateFrame struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Xstart      int     `json:"x_start"`
	Xstop       int     `json:"x_stop"`
	Ystart      int     `json:"y_start"`
	Ystop       int     `json:"y_stop"`
	Zstart      int     `json:"z_start"`
	Zstop       int     `json:"z_stop"`
	Xvoxelsize  float32 `json:"x_voxel_size"`
	Yvoxelsize  float32 `json:"y_voxel_size"`
	Zvoxelsize  float32 `json:"z_voxel_size"`
	VoxelUnit   string  `json:"voxel_unit"`
}

// GetCoordFrame returns the coord frame infromation corresponding to coord_frame
func (b Info) GetCoordFrame(coordFrameName string) (c CoordinateFrame, err error) {
	client := &http.Client{}

	serverURL := b.getServerURL()
	URL := fmt.Sprintf("%s/coord/%s", serverURL, coordFrameName)
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return CoordinateFrame{}, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", b.Token))
	resp, err := client.Do(req)
	if err != nil {
		return CoordinateFrame{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return CoordinateFrame{}, err
	}

	dec := json.NewDecoder(bytes.NewReader(body))
	for {
		if err := dec.Decode(&c); err == io.EOF {
			break
		} else if err != nil {
			return CoordinateFrame{}, err
		}
		return c, nil
	}

	return CoordinateFrame{}, errors.New("coord frame: Unable to parse JSON coordinate frame information")
}
