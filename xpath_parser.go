package squeegee

import (
	"errors"
	"fmt"
	"launchpad.net/xmlpath"
	"strings"
)

// XpathParser -
type XpathParser struct {
	xpath     string
	XpathComp *xmlpath.Path
}

// Init -
func (xp *XpathParser) Init(configData []interface{}) error {
	ok := len(configData) > 0
	if !ok {
		return errors.New("xpath string (parser_data first param) must be present for xpath")
	}
	xpd := configData[0]
	xpth, ok := xpd.(string)
	if !ok {
		return errors.New("xpath configuration field must be a string")
	}
	xp.xpath = xpth

	xpc, err := xmlpath.Compile(xp.xpath)
	if err != nil {
		return fmt.Errorf("xpath did not compile starting at %s", xp.xpath[:10])
	}
	xp.XpathComp = xpc
	return nil
}

// Parse -
func (xp *XpathParser) Parse(d *[]byte) []string {
	o := []string{}
	root, err := xmlpath.Parse(strings.NewReader(string(*d)))
	if err != nil {
		return o
	}
	iter := xp.XpathComp.Iter(root)
	for iter.Next() {
		if s, ok := xp.XpathComp.String(iter.Node()); ok {
			o = append(o, s)
		}
	}
	return o
}
