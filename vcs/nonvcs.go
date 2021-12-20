package vcs

import (
	"encoding/json"
	"log"
	//"os/exec"
	"os"
)



func init() {
	Register(newNonVcs, "nonvcs")
}

type NonVcsDriver struct {
	symbol_link     bool   `json:"symbol"`
}

func newNonVcs(b []byte) (Driver, error) {
	var d NonVcsDriver

	if b != nil {
		if err := json.Unmarshal(b, &d); err != nil {
			return nil, err
		}
	}

	return &d, nil
}

func (g *NonVcsDriver) HeadRev(dir string) (string, error) {
	return "nonvcs",nil
}


func (g *NonVcsDriver) Pull(dir string) (string, error) {
	return g.HeadRev(dir)
}


func (g *NonVcsDriver) Clone(dir, url string) (string, error) {
	os.Symlink(url,dir)
	log.Printf("clone nonvcs")
	//cmd :=exec.Command("cp", "-R",url,dir)
	//cmd.Run()
	return g.Pull(dir)
}

func (g *NonVcsDriver) SpecialFiles() []string {
	return []string{
	}
}

