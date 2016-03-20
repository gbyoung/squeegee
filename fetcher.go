package squeegee

import (
	"sync"
)

func (s *Squeegee) incFetchingCount() {
	s.numFetchingMutex.Lock()
	s.numFetching++
	s.numFetchingMutex.Unlock()
}

func (s *Squeegee) decFetchingCount() {
	s.numFetchingMutex.Lock()
	s.numFetching--
	s.numFetchingMutex.Unlock()
}

func lockAndUpdateCntrs(entry string, mutex *sync.RWMutex, cntr *map[string]int, badOnes *map[string]bool) bool {
	if entry == "" {
		return false
	}
	bad := false
	mutex.Lock()
	(*cntr)[entry]++
	if (*cntr)[entry] > 5 {
		(*badOnes)[entry] = true
		bad = true
	}
	mutex.Unlock()
	return bad
}

func (s *Squeegee) getFromCache(theURL string) *[]byte {
	var data *[]byte
	if s.Config.UsingCache {
		d, err := s.Db.Get([]byte(theURL), nil)
		if err == nil {
			data = &d
		}
	}
	return data
}

func (s *Squeegee) putToCache(theURL string, data *[]byte) {
	if s.Config.UsingCache && len(*data) > 512 {
		err := s.Db.Put([]byte(theURL), *data, nil)
		if err == nil {
			Log.Error("Error writing to cache DB")
			Log.Error(err.Error())
		}
	}
}

func (s *Squeegee) fetchOk(sf *Fetch, err error, foundURLChan chan *string) bool {
	if err == nil {
		return true
	}
	Log.Warning(err.Error())
	if sf.Proxy != "" {
		lockAndUpdateCntrs(sf.Proxy, &s.badProxyMutex, &s.badProxyCounter, &s.badProxies)
	}
	badURL := lockAndUpdateCntrs(sf.URL, &s.badURLMutex, &s.badURLCounter, &s.badURLs)
	if !badURL {
		foundURLChan <- &sf.URL
	}
	return false
}

// Fetcher -
func (s *Squeegee) Fetcher(inURLChan chan *Fetch, foundURLChan chan *string) {
	for {
		sf := <-inURLChan
        Log.Debug("Starting fetch")
		s.incFetchingCount()
		data := s.getFromCache(sf.URL)
		if (data == nil) || len(*data) < 512 {
            Log.Debug("Not found in Cache")
			// Found nothing cached, fetch the page
			d, err := s.URLFetcher.Fetch(sf)
            Log.Debug("Checking if fetch OK")
			if s.fetchOk(sf, err, foundURLChan) {
				data = d
                Log.Debug("Putting data in cache")                
                s.putToCache(sf.URL, data)                
			}
		}
        Log.Debug("About to parse the data")
        s.parseData(sf.URL, data, foundURLChan)
        Log.Debug("Ending fetch")        
		s.decFetchingCount()
	}
}
