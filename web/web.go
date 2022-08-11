package web

import (
	"fmt"
	"net/http"
	"io/fs"
	"path"
	 "io/ioutil"
	 "strings"
	"sync"
	 "embed"
	"github.com/hound-search/hound/api"
	"github.com/hound-search/hound/config"
	"github.com/hound-search/hound/searcher"
	"github.com/hound-search/hound/ui"
)
//go:embed prism
var  embededFiles embed.FS
// Server is an HTTP server that handles all
// http traffic for hound. It is able to serve
// some traffic before indexes are built and
// then transition to all traffic afterwards.
type Server struct {
	cfg *config.Config
	dev bool
	ch  chan error

	mux *http.ServeMux
	lck sync.RWMutex
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == s.cfg.HealthCheckURI {
		fmt.Fprintln(w, "👍")
		return
	}

	s.lck.RLock()
	defer s.lck.RUnlock()
	if m := s.mux; m != nil {
		m.ServeHTTP(w, r)
	} else {
		http.Error(w,
			"Hound is not ready.",
			http.StatusServiceUnavailable)
	}
}

func (s *Server) serveWith(m *http.ServeMux) {
	s.lck.Lock()
	defer s.lck.Unlock()
	s.mux = m
}

// Start creates a new server that will immediately start handling HTTP traffic.
// The HTTP server will return 200 on the health check, but a 503 on every other
// request until ServeWithIndex is called to begin serving search traffic with
// the given searchers.
func Start(cfg *config.Config, addr string, dev bool) *Server {
	ch := make(chan error)

	s := &Server{
		cfg: cfg,
		dev: dev,
		ch:  ch,
	}

	go func() {
		ch <- http.ListenAndServe(addr, s)
	}()

	return s
}
type  PrismFile struct {
	 *strings.Reader
	 f http.File
	 totallen int64
}
const gpre string="<html><head><link href=\"/prism/prism.css\" rel=\"stylesheet\"/></head><body class=\"line-numbers\"><pre><code class=\"language-%s\">"
const gpost string="</code></pre><script src=\"/prism/prism.js\"></script></body></html>"
func NewPrismFile(f http.File) (pr *PrismFile) {
	finfo,_:=f.Stat()
	if finfo.IsDir() {
	pr=&PrismFile {
	     f: f,
	     Reader: nil,
     }
     } else {
	     bs,_:=ioutil.ReadAll(f)
	     suff:=path.Ext(finfo.Name())
	     if len(suff)>1 {
		     suff=suff[1:]
	     }
	     tpre:=fmt.Sprintf(gpre,suff)
	     ss:=tpre+strings.ReplaceAll(strings.ReplaceAll(string(bs),"<","&lt;"),">","&gt;")+gpost
	     /*ssarray:=strings.Split(string(bs),"\n")
	     var  bb strings.Builder;
	     for i,s := range ssarray {
		     fmt.Fprintf(&bb,"<li id=\"L%d\"/>%s\n",i+1,s)
	     }
	     */

	pr=&PrismFile {
		f: f,
		totallen: int64(len(ss)),
		Reader: strings.NewReader(ss),
	}
     }
     return 

}
func (pp *PrismFile) Close()(err error) {
	 pp.f.Close()
	  err=nil
	  return
	}

func (pp *PrismFile) Readdir(n int)(fis []fs.FileInfo, err error) {
	  fis,err=pp.f.Readdir(n)
	  return
}
type  PrismFileInfo struct  {
	fs.FileInfo
	totallen int64
}
func  (info PrismFileInfo) Name() string {
	 return "tmp.html"
}
func  (info PrismFileInfo) Size() int64 {
	return info.totallen
}
func (pp *PrismFile) Stat()(fs.FileInfo, error) {
	     fis,err:=pp.f.Stat()
	     return  PrismFileInfo{FileInfo:fis, totallen:pp.totallen},err
}
func (pp PrismFile) Read(p []byte)(n int,err error) {
	n,err=pp.Reader.Read(p)
	return
}
func (pp PrismFile) Seek(offset int64,whence int) (n int64,err error) {
	n,err=pp.Reader.Seek(offset,whence)
	return
}
func (pp PrismFile) ReadAt(b []byte,off int64) (n int,err error) {
	n,err=pp.Reader.ReadAt(b,off)
	return
}
type  PrismFileSystem  struct {
	http.FileSystem
}
func (fsys  *PrismFileSystem)  Open(name string) (http.File, error)  {

		 file,err :=fsys.FileSystem.Open(name)
		 if err != nil {
			 return nil,err
		 }
		 return  NewPrismFile(file),err
}

// ServeWithIndex allow the server to start offering the search UI and the
// search APIs operating on the given indexes.
func (s *Server) ServeWithIndex(idx map[string]*searcher.Searcher) error {
	h, err := ui.Content(s.dev, s.cfg)
	if err != nil {
		return err
	}
	m := http.NewServeMux()
	m.Handle("/", h)
	api.Setup(m, idx)
	embed1,err :=fs.Sub(embededFiles,"prism")
	if err != nil {
		return err
	}
	m.Handle("/prism/",http.StripPrefix("/prism/",http.FileServer(http.FS(embed1))))
	for k,v :=range idx {
		if v.Path!="" {
		m.Handle("/nonvcs/"+k+"/",http.StripPrefix("/nonvcs/"+k+"/",http.FileServer(&PrismFileSystem {http.Dir(v.Path)})))
		}
	}

	s.serveWith(m)

	return <-s.ch
}
