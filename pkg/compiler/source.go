package compiler

import (
	"fmt"
	//"github.com/gijit/gi/pkg/token"
	//"github.com/gijit/gi/pkg/types"
	//golua "github.com/glycerine/golua/lua"
	//"github.com/glycerine/luar"
)

var _ = fmt.Printf

func (ic *IncrState) ImportSourcePackage(path, pkgDir string, depth int) (res *Archive, err error) {
	pp("IncrState.ImportSourcePackage() top. path='%s', pkgDir='%s'", path, pkgDir)

	//if ic.AllowImportCaching {
	// from ~/go/src/github.com/gopherjs/gopherjs/build/build.go

	if archive, ok := ic.Session.Archives[path]; ok {
		fmt.Printf("using cached copy of package '%s'\n", path)
		return archive, nil
	}

	//}
	_, archive, err := ic.Session.BuildImportPathWithSrcDir(path, pkgDir, depth)
	if err != nil {
		return nil, err
	}
	pp("back from BuildImportWithSrcDir(path='%s'), archive='%#v'", path, archive)
	pp("back from BuildImportWithSrcDir(path='%s'), archive.Pkg='%#v'", path, archive.Pkg) // nil here
	ic.CurPkg.importContext.Packages[archive.ImportPath] = archive.Pkg
	ic.Session.Archives[path] = archive

	// very important, must do this or we won't locate the package!
	ic.CurPkg.importContext.Packages[path] = archive.Pkg

	return archive, nil
}
