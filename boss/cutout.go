package boss

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// Cutout downloads a chunk of blosc-compressed data from the Boss, decompresses it, and returns the raw data in a byte array
func (b Info) Cutout(collection string, experiment string, channel string, xrng [2]int, yrng [2]int, zrng [2]int, resolution int) ([]byte, error) {
	client := &http.Client{}

	cutoutArgsStr := fmt.Sprintf("%d/%d:%d/%d:%d/%d:%d/", resolution, xrng[0], xrng[1], yrng[0], yrng[1], zrng[0], zrng[1])

	serverURL := b.getServerURL()
	URL := fmt.Sprintf("%s/cutout/%s/%s/%s/%s", serverURL, collection, experiment, channel, cutoutArgsStr)
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return []byte{}, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Token %s", b.Token))
	req.Header.Add("Accept", "application/blosc")
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	// TODO(adb): include this error check in the other URL calls
	if resp.StatusCode != http.StatusOK {
		errorMsgBytes, _ := ioutil.ReadAll(resp.Body)
		errorMsg := string(errorMsgBytes)
		return []byte{}, fmt.Errorf("cutout: http request returned error (%d):\n%s\nURL Requested: %s", resp.StatusCode, errorMsg, URL)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	ret, err := Decompress(body)
	if err != nil {
		return []byte{}, err
	}
	return ret, nil
}
