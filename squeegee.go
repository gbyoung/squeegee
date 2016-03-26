// Package squeegee is a high-performance web scraping library built with golang
package squeegee

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
    log "github.com/Sirupsen/logrus"
	"math/rand"
	"net/url"
	"os" 
	"sync"
	"time"
	//"github.com/davecgh/go-spew/spew"
)

// Fetch - Sent to our parallel fetchers
type Fetch struct {
	URL       string
	Proxy     string
	Useragent string
}

// URLFetcher -
type URLFetcher interface {
	Fetch(sf *Fetch) (*[]byte, error)
}

// Parser -
type Parser interface {
	Init(configData []interface{}) error
	Parse(d *[]byte) []string
}

// Writer -
type Writer interface {
	Init(fileName string, columns []string) error
	Start(outChan chan *[]string, stopChan chan string)
}

// Squeegee -
type Squeegee struct {
	Config           Config
	Db               *leveldb.DB
	URLFetcher       URLFetcher
	numFetchingMutex sync.RWMutex
	numFetching      int
	badProxyMutex    sync.RWMutex
	badProxyCounter  map[string]int
	badProxies       map[string]bool
	badURLMutex      sync.RWMutex
	badURLCounter    map[string]int
	badURLs          map[string]bool
	StopChan         chan string
}

func (s *Squeegee) kickoffFetchers(inURLChan chan *Fetch, foundURLChan chan *string) {
	for i := 0; i < s.Config.NumConcurrent; i++ {
		go s.Fetcher(inURLChan, foundURLChan)
	}
}

func (s *Squeegee) kickoffScrapeUrls(foundURLChan chan *string) {
	for _, surl := range s.Config.ScrapeUrls {
		foundURLChan <- &surl
	}
}

func randStringOrBlank(num int, entries []string) string {
	o := ""
	if num > 0 {
		o = entries[rand.Intn(num)]
	}
	return o
}

// StartFetchersFeedUrls -
func (s *Squeegee) StartFetchersFeedUrls() {
	// Make our FiFos config.NumConcurrent deep so we don't
	// lose any performance waiting
	inURLChan := make(chan *Fetch, s.Config.NumConcurrent)
	foundURLChan := make(chan *string, s.Config.NumConcurrent)
	s.kickoffFetchers(inURLChan, foundURLChan)
	go s.kickoffScrapeUrls(foundURLChan)

	// Main fetch loop.  If we don't see any URLs for 10 seconds or if
	// no fetchers are working and we haven't seen any URLs for 5
	// seconds, then we're done.
	timeoutCond := false
MainLoop:
	for {
		select {
		case furl := <-foundURLChan:
			url, err := url.Parse(*furl)
			log.Debug(url.String())
			if err != nil {
				log.Warning(fmt.Sprintf("Found bad URL:  %s\n", *furl))
				continue
			}
			timeoutCond = false
			proxy := randStringOrBlank(s.Config.NumProxies, s.Config.Proxies)
			useragent := randStringOrBlank(s.Config.NumUseragents, s.Config.Useragents)
			sf := Fetch{
				URL:       *furl,
				Proxy:     proxy,
				Useragent: useragent,
			}
			inURLChan <- &sf
		case <-time.After(time.Second * 5):
			s.numFetchingMutex.RLock()
			numFetching := s.numFetching
			s.numFetchingMutex.RUnlock()
			if (numFetching == 0) || timeoutCond {
				if timeoutCond {
					log.Error(fmt.Sprintf("We had a timeout but all our fetchers don't show as done! %d outstanding.", numFetching))
				}
				break MainLoop
			}
			timeoutCond = true
		}
	}
	log.Debug("Done fetching...")

	//Done.  Feed all the Writers nils so they'll clean up and exit
	for _, oc := range s.Config.OutChans {
		oc <- nil
	}
}

// StartWriters -
func (s *Squeegee) StartWriters() {
	var ostopChans []chan string
	for fn, outChan := range s.Config.OutChans {
		writer := *(s.Config.Writers[fn])
		ostopChan := make(chan string)
		go writer.Start(outChan, ostopChan)
		ostopChans = append(ostopChans, ostopChan)
	}
	msg := ""
	for _, ostopChan := range ostopChans {
		msg = <-ostopChan
	}
	// All of our writers have received nils to terminate them.
	// Send our main thread the final message received from our writers
	// through stopChan to terminate
    log.Debug("Got stop message from the writers.  Stopping...")
	s.StopChan <- msg
}

// Start - Start squeegee fetching, parsing, and writing
func (s *Squeegee) Start() {
	// The StartWriters will send a bool down stop_chan to stop us.
	// This will occur only after all the writers
	// have received nils from our StartFetchersFeedUrls above.
	go s.StartWriters()
	go s.StartFetchersFeedUrls()
	return 
}

// ClearCache - Clear the fetch cache
func (s *Squeegee) ClearCache() error {
	fn := s.Config.CacheFile
	if fn == "" {
		return nil
	}
	if err := os.Remove(fn); err != nil {
		return err
	}
	return s.openDb()
}

func (s *Squeegee) openDb() error {
	db, err := leveldb.OpenFile(s.Config.CacheFile, nil)
	if err != nil {
		return err
	}
	s.Db = db
	return nil
}

func (s *Squeegee) configureLogging(debug bool) error {
	if debug {
        log.SetLevel(log.DebugLevel)
	}
	if s.Config.LogFile != "" {
        fl, err := os.Create(s.Config.LogFile)
        if err != nil {
            return err
        }        
        log.SetOutput(fl)
	}
	return nil
}

// Init -- Initialize Squeegee.  After initialization s.Config is static and can
// be used within our goroutines without issue
// See:  https://groups.google.com/forum/#!msg/golang-nuts/HpLWnGTp-n8/hyUYmnWJqiQJ
// Data outside s.Config can be changed while running and, therefore,
// requires locking
func (s *Squeegee) Init(confFile string, debug bool) error {
	config, err := Configure(confFile)
	if err != nil {
		return err
	}
	s.Config = config

	err = s.configureLogging(debug)
	if err != nil {
		return err
	}

	if config.CacheFile != "" {
		config.UsingCache = true
		if err := s.openDb(); err != nil {
			return err
		}
	}
    s.StopChan = make(chan string)
	s.URLFetcher = GoReqURLFetcher{}
	log.Info("Init:  Successfully Initialized")
	return nil
}
