package squeegee

import (
	"errors"
	"fmt"
	// "github.com/davecgh/go-spew/spew"
	"github.com/naoina/toml"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func createWriter(t string) (Writer, error) {
    switch t{
	case ".txt":
		return new(TxtWriter),nil
	default:
		return nil, fmt.Errorf("Invalid Writer Type Specified: %s", t)
	}
}

func createParser(t string) (Parser, error) {
    switch t{
	case "regex":
		return new(RegexParser),nil
	case "xpath":
		return new(XpathParser),nil
	default:
		return nil, fmt.Errorf("Invalid Parser Type Specified: %s", t)
	}
}

// OutputPattern -
type OutputPattern struct {
	Column     string        `toml:"column"`
	ParserType string        `toml:"parser_type"`
	ParserData []interface{} `toml:"parser_data"`
	Parser     Parser
}

// Output -
type Output struct {
	FileName           string          `toml:"file_name"`
	Columns            []string        `toml:"columns"`
	IncludeColumnNames bool            `toml:"include_column_names"`
	RequireAllColumns  bool            `toml:"require_all_columns"`
	LogIfUnfound       bool            `toml:"log_if_unfound"`
	StopIfUnfound      bool            `toml:"stop_if_unfound"`
	OutputPatterns     []OutputPattern `toml:"patterns"`
	outChan            chan *[]string
	outWriter          *Writer
}

// FindURLPattern -
type FindURLPattern struct {
	ParserType    string        `toml:"ParserType"`
	ParserData    []interface{} `toml:"ParserData"`
	Parser        Parser
	LogIfUnfound  bool `toml:"log_if_unfound"`
	StopIfUnfound bool `toml:"stop_if_unfound"`
}

// MatchURL -
type MatchURL struct {
	URLRegex        string `toml:"url_regex"`
	URLRegexComp    *regexp.Regexp
	FindURLPatterns []FindURLPattern `toml:"find_url_patterns"`
	Outputs         []Output         `toml:"outputs"`
}

// Config -
type Config struct {
	LogFile           string     `toml:"log_file"`
	NumConcurrent     int        `toml:"num_concurrent"`
	NumErrRemoveProxy int        `toml:"num_err_remove_proxy"`
	NumErrRemoveURL   int        `toml:"num_err_remove_url"`
	CacheFile         string     `toml:"cache_file"`
	ProxyListFile     string     `toml:"proxy_list_file"`
	ScrapeUrls        []string   `toml:"scrape_urls"`
	Proxies           []string   `toml:"proxies"`
	Useragents        []string   `toml:"useragents"`
	MatchURLs         []MatchURL `toml:"match_urls"`
	OutChans          map[string]chan *[]string
	Writers           map[string]*Writer
	UsingCache        bool
	NumProxies        int
	NumUseragents     int
}

// Configure -
func Configure(confFile string) (Config, error) {
	var config Config
	f, err := os.Open(confFile)
	if err != nil {
		return config, err
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return config, err
	}
	if err := toml.Unmarshal(buf, &config); err != nil {
		return config, err
	}
	if err := config.readProxyList(); err != nil {
		return config, err
	}
	if config.NumErrRemoveProxy == 0 {
		config.NumErrRemoveProxy = 5
	}
	if config.NumErrRemoveURL == 0 {
		config.NumErrRemoveURL = 5
	}
	if config.NumConcurrent == 0 {
		config.NumConcurrent = 1
	}
	if err := config.initAllMatchers(); err != nil {
		return config, err
	}
	if err := config.verifyConfigRequirements(); err != nil {
		return config, err
	}
	if err := config.generateOutputs(); err != nil {
		return config, err
	}
	// spew.Dump(config)
	return config, nil
}

func (sc *Config) generateOutputs() error {
	outChans := make(map[string]chan *[]string)
	outWrtrs := make(map[string]*Writer)
	// Go through our config structure and generate
	// outgoing channels and file writers, one for each filename,
	// and attach these channels and writers to the config on all Output
	// nodes
	for mui, mu := range sc.MatchURLs {
		for opi, op := range mu.Outputs {
			theExt := strings.ToLower(filepath.Ext(op.FileName))
			if _, ok := outChans[op.FileName]; !ok {
				outChans[op.FileName] = make(chan *[]string)
				writer, err := createWriter(theExt)
                if err != nil {
                    return err
                }
			    outWrtrs[op.FileName] = &writer                
                if err := writer.Init(op.FileName, op.Columns); err != nil {
                    return err
                }
			}
			sc.MatchURLs[mui].Outputs[opi].outChan = outChans[op.FileName]
			sc.MatchURLs[mui].Outputs[opi].outWriter = outWrtrs[op.FileName]
		}
	}
	sc.OutChans = outChans
	sc.Writers = outWrtrs
	return nil
}

func (sc *Config) verifyUseragents() {
	sc.NumUseragents = 0
	var uas []string
	nonASCIIRe := regexp.MustCompile(`[^\x00-\x7F]`)
	for _, ua := range sc.Useragents {
		nua := nonASCIIRe.ReplaceAllString(ua, "")
		if nua != "" {
			uas = append(uas, nua)
		}
	}
	sc.Useragents = uas
	sc.NumUseragents = len(sc.Useragents)
	if sc.NumUseragents == 0 {
		sc.Useragents = []string{"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"}
		sc.NumUseragents = 1
	}
}

func (sc *Config) verifyScrapeUrls() error {
	for _, ul := range sc.ScrapeUrls {
		if _, err := url.Parse(ul); err != nil {
			return err
		}
	}
	if len(sc.ScrapeUrls) < 1 {
		return errors.New("You must specify at least one valid scrape_url in your config.")
	}
	return nil
}

func (sc *Config) verifyConfigRequirements() error {
	sc.verifyUseragents()
	if err := sc.verifyScrapeUrls(); err != nil {
		return err
	}
	// TODO  Write complete verification for configuration
	return nil
}

func (sc *Config) initAllMatchers() error {
	for mui, mu := range sc.MatchURLs {
		comp, err := regexp.Compile(mu.URLRegex)
		if err != nil {
			return err
		}
        sc.MatchURLs[mui].URLRegexComp = comp
		for fupi, fup := range mu.FindURLPatterns {
			p, err := createParser(fup.ParserType)
			if err != nil {
				return err
			}
			err = p.Init(fup.ParserData)
			if err != nil {
				return err
			}
            sc.MatchURLs[mui].FindURLPatterns[fupi].Parser = p            
		}
		for opi, op := range mu.Outputs {
			for opai, opat := range op.OutputPatterns {
                p, err := createParser(opat.ParserType)
                if err != nil {
                    return err
                }
				err = p.Init(opat.ParserData)
				if err != nil {
					return err
				}
                sc.MatchURLs[mui].Outputs[opi].OutputPatterns[opai].Parser = p
			}
		}
	}
	return nil
}

func readListFile(fn string, checkRe *regexp.Regexp) ([]string, int, error) {
	lst := []string{}
	if fn == "" {
		return lst, 0, nil
	}
	p, err := ioutil.ReadFile(fn)
	if err != nil {
		return lst, 0, err
	}

	i := 0
	for _, l := range strings.Split(string(p), "\n") {
		tl := strings.Trim(l, "\r\n\t ")
		if tl == "" || !checkRe.MatchString(tl) {
			continue
		}
		lst = append(lst, tl)
		i++
	}
	return lst, i, nil
}

func (sc *Config) readProxyList() error {
	// TODO - Make sure I handle proxies without port numbers (assume 80)
	plRe, err := regexp.Compile(`^(\w+\:\w+\@){0,1}((?:1?\d{1,2}|2[0-4]\d|25[0-5])\.){3}(?:1?\d{1,2}|2[0-4]\d|25[0-5]):\d{2,5}$`)
	if err != nil {
		return err
	}
	pxs, np, err := readListFile(sc.ProxyListFile, plRe)
	if np > 0 {
		sc.Proxies = pxs
		sc.NumProxies = np
	}
	return err
}
