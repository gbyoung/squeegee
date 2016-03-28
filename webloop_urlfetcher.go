package squeegee

import (
    "github.com/conformal/gotk3/gtk"
    "github.com/sourcegraph/webloop"
)

// WebloopURLFetcher -
type WebloopURLFetcher struct {
}

// Fetch -
func (gr WebloopURLFetcher) Fetch(sf *Fetch) (*[]byte, error) {
    gtk.Init(nil)
    go func() {
        runtime.LockOSThread()
        gtk.Main()
    }()

    ctx := webloop.New()
    view := ctx.NewView()
    defer view.Close()
    view.Open(sf.URL)
    err := view.Wait()
    if err != nil {
        return nil, err
    }
    res, err := view.EvaluateJavaScript("document.title")
    if err != nil {
        return nil, err        
    }
    d := []byte(res.(string))
	return &d, nil
}
