package squeegee

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// RegexParser -
type RegexParser struct {
	regex              string
	regexComp          *regexp.Regexp
	regexMatchGroupNum int
}

// Init -
func (rp *RegexParser) Init(configData []interface{}) error {
	ok := len(configData) > 1
	if !ok {
		return errors.New("match group num (parser_data second param) must be present for regex")
	}
	d := configData[1]
	mgn, ok := d.(string)
	if !ok {
		return errors.New("The second parser_data paramater must be an string")
	}
	i, err := strconv.Atoi(mgn)
	if err != nil {
		return errors.New("The second parser_data paramater must be an string but must be parsable as an integer")
	}
	rp.regexMatchGroupNum = i

	rg := configData[0]
	rgs, ok := rg.(string)
	if !ok {
		return errors.New("regex configuration field must be string")
	}
	rp.regex = rgs

	rgc, err := regexp.Compile(rp.regex)
	if err != nil {
		return fmt.Errorf("regex did not compile starting at %s", rp.regex[:10])
	}
	rp.regexComp = rgc
	return nil
}

// Parse -
func (rp *RegexParser) Parse(d *[]byte) []string {
	return rp.regexComp.FindAllString(string(*d), -1)
}
