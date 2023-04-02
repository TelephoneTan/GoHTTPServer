package base

import "github.com/TelephoneTan/GoHTTPServer/util"

type PathPack struct {
	Path       []string
	PrefixPath []string
	SuffixPath []string
}

func (p *PathPack) Clone() PathPack {
	return PathPack{
		Path:       util.ShallowCloneSlice(p.Path),
		PrefixPath: util.ShallowCloneSlice(p.PrefixPath),
		SuffixPath: util.ShallowCloneSlice(p.SuffixPath),
	}
}
