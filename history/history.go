// Package history contains functions for saving and loading historical
// image-dumps. This allows for debugging and testing using prefabricated data.
package history

import (
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/whomever000/poker-client-pokerstars/vision"

	log "github.com/Sirupsen/logrus"
)

// NewImageSource creates a new historical image source given the process ID
// (name of sub-folder).
func NewImageSource(pid int, block bool) vision.ImageSource {
	return &ImageSource{
		dir:      "./dump/" + fmt.Sprintf("%v", pid) + "/",
		lastDate: time.Unix(0, 0),
		block:    block,
	}
}

// ImageSource is an image source based on historical image-dumps.
type ImageSource struct {
	dir      string
	lastDate time.Time
	block    bool
}

// Get returns the next image in the history sequence.
func (is *ImageSource) Get() image.Image {

	var img image.Image

	// Wait for user input
	if is.block {
		var input string
		fmt.Scanln(&input)
	}

	// List files in directory
	ls, err := ioutil.ReadDir(is.dir)
	if err != nil {
		panic(err)
	}

	// Iterate files and find the next in the sequence.
	var fileInfo os.FileInfo
	for _, f := range ls {
		// Check if file is older than last evaluated file
		if fileInfo == nil || f.ModTime().Before(fileInfo.ModTime()) {
			// Check if file is newer than last returned file
			if f.ModTime().After(is.lastDate) {
				fileInfo = f
			}
		}
	}

	// Verify that an image was found
	if fileInfo == nil {
		log.Info("End of history sequence")
		return vision.Image()
	}

	// Update lastDate so that next time, the next image in the sequence will
	// be returned instead.
	is.lastDate = fileInfo.ModTime()

	// Get description from file name
	var descr = ""
	strSplit := strings.Split(fileInfo.Name(), "_")
	if len(strSplit) >= 2 {
		descr = strSplit[1]
	}
	log.Infof("History: %v", descr)

	// Read file
	file, err := os.Open(is.dir + fileInfo.Name())
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Decode image
	img, err = png.Decode(file)
	if err != nil {
		panic(err)
	}

	vision.SetImage(img)
	return img
}

// Save saves a historical image.
func Save(descr string) {
	// Determine directory and file name
	pid := fmt.Sprintf("%v", os.Getpid())
	time := fmt.Sprintf("%v", time.Now().Unix())
	dir := "./dump/" + pid + "/"
	file := time + "_" + descr + ".png"

	// Create dirs
	os.MkdirAll(dir, os.ModePerm)

	// Create and open file
	f, err := os.Create(dir + file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Encode image
	err = png.Encode(f, vision.Image())
	if err != nil {
		panic(err)
	}

	f.Sync()
}
