package squeegee

import (
	"fmt"
	"net/url"
    log "github.com/Sirupsen/logrus"
)

func (s *Squeegee) findUrls(pageURL string, mu MatchURL, data *[]byte, foundURLChan chan *string) {
	for _, fup := range mu.FindURLPatterns {
		found := false
		for _, foundURL := range fup.Parser.Parse(data) {
			if _, err := url.Parse(foundURL); err == nil {
				s.badURLMutex.RLock()
				found = true
				if _, bad := s.badURLs[foundURL]; !bad {
					foundURLChan <- &foundURL
				}
				s.badURLMutex.RUnlock()
			}
		}
		if !found {
			if fup.StopIfUnfound {
				msg := fmt.Sprintf("No URLs found and StopIfUnfound Config option set for page %s\n", pageURL)
				log.Error(msg)
				s.StopChan <- msg

			} else if fup.LogIfUnfound {
				log.Warning(fmt.Sprintf("No URLs found and logIfUnfound Config option set for page %s\n", pageURL))
			}
		}
	}
}

// colsFoundAndFill - Algorithm that duplicates column values if fewer data values for a given column are
// found relative to other columns on the page.  If only a single value is found then
// the value will be repeated for all rows on the page.
// If multiple values are found for a column then a blank will be used for unfound rows.
func colsFoundAndFill(v *map[string][]string) (one bool, all bool, o [][]string) {
    log.Debug("Filling Columns")
	dupChkr := make(map[string]string)
	fndDat := false
	for i := 0; fndDat; i++ {
		fndDat = false
		var row []string
		for c, d := range *v {
			ld := len(d)
			if i == 0 {
				gt0 := ld > 0
				one = one || gt0
				all = all && gt0
			}
			dupv, fndDup := dupChkr[c]
			if i < ld {
				fndDat = true
				row = append(row, d[i])
				if !fndDup {
					dupChkr[c] = d[i]
				} else if dupv != d[i] {
					dupChkr[c] = ""
				}
			} else if fndDup {
				row = append(row, dupv)
			} else {
				row = append(row, "")
			}
		}
		if fndDat {
			o = append(o, row)
		}
	}
    log.Debug("Done Filling Columns")    
	return one, all, o
}

func (s *Squeegee) findData(pageURL string, mu MatchURL, data *[]byte) {
	for _, opt := range mu.Outputs {
		cols := make(map[string][]string)
		for _, op := range opt.OutputPatterns {
			dt := op.Parser.Parse(data)
			if _, ok := cols[op.Column]; (len(dt) > 0) && ok {
				cols[op.Column] = dt
			}
		}
		oneColFound, allColsFound, dat := colsFoundAndFill(&cols)
		if !oneColFound || !(allColsFound && opt.RequireAllColumns) {
			if opt.StopIfUnfound {
				if opt.RequireAllColumns {
					msg := fmt.Sprintf("All Columns for found and StopIfUnfound Config option set for page %s\n", pageURL)
					log.Error(msg)
					s.StopChan <- msg
				}
			} else if opt.LogIfUnfound {
				log.Warning(fmt.Sprintf("All Columns for found and StopIfUnfound Config option set for page %s\n", pageURL))
			}
		} else {
			for _, row := range dat {
				opt.outChan <- &row
			}
		}
	}
}

func (s *Squeegee) parseData(pageURL string, data *[]byte, foundURLChan chan *string) {
	for _, mu := range s.Config.MatchURLs {
		if mu.URLRegexComp.Match([]byte(pageURL)) {
            log.Debug("About to parse and find the URLs on the page")
			s.findUrls(pageURL, mu, data, foundURLChan)
            log.Debug("About to parse and find the data on the page")            
			s.findData(pageURL, mu, data)
            log.Debug("Done Parsing the page:")
            log.Debug(pageURL)
		}
	}
}
