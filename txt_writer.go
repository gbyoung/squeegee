package squeegee

import (
	"errors"
	"fmt"
	"os"
    log "github.com/Sirupsen/logrus"
)

// TxtWriter --
type TxtWriter struct {
	FileName string
	fp       *os.File
}

// Init --
func (tw *TxtWriter) Init(fileName string, columns []string) error {
	if len(columns) != 1 {
		return errors.New("Tried to initialize a text output with more than one column.")
	}
	file, err := os.Create(fileName)
    tw.FileName = fileName
	tw.fp = file
	return err
}

// Start --
func (tw *TxtWriter) Start(outChan chan *[]string, stopChan chan string) {
	for {
		colData := <-outChan
		if colData == nil {
			break
		}
		ncols := len(*colData)
		if ncols != 1 {
			fmt.Printf("Tried to write %d columns of data to Txt Writer (only 1 is allowed)", ncols)
		}
		tw.fp.WriteString((*colData)[0] + "\n")
	}
    log.Debug("Got the nil on the outChan.  Sending Done to main")
	stopChan <- "Done"
}
