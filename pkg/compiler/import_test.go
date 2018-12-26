package compiler

import (
	"fmt"
	"os"
	"testing"

	//"github.com/gijit/gi/pkg/verb"
	cv "github.com/glycerine/goconvey/convey"
)

func Test1000ImportAGoSourcePackage(t *testing.T) {

	cv.Convey(`import a Go source package`, t, func() {

		fishMultipliesBy(2)
		code := `
import "github.com/gijit/gi/pkg/compiler/spkg_tst"
caught := spkg_tst.Fish(2)
`
		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()
		inc := NewIncrState(vm, nil)

		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		fmt.Printf("\n translation='%s'\n", translation)

		// and verify that it happens correctly
		LuaRunAndReport(vm, string(translation))

		LuaMustInt64(vm, "caught", 4)
	})
}

func Test1001NoCachingOfImportsOfGoSourcePackages(t *testing.T) {

	cv.Convey(`since they may be in flux, importing a Go source package must re-read the source every time, and not use a cached version`, t, func() {

		defer fishMultipliesBy(2) // cleanup

		for i := 1; i <= 2; i++ {
			fmt.Printf("\n ... fishing for import caching, which is a no-no on source imports. They may change often. on i=%v\n\n", i)
			fishMultipliesBy(i + 1) // 2, then 3
			code := `
import "github.com/gijit/gi/pkg/compiler/spkg_tst"`
			code2 := `
caught := spkg_tst.Fish(2)
`
			vm, err := NewLuaVmWithPrelude(nil)
			panicOn(err)
			defer vm.Close()
			inc := NewIncrState(vm, nil)

			translation, err := inc.Tr([]byte(code))
			panicOn(err)
			fmt.Printf("\n translation='%s'\n", translation)

			// and verify that it happens correctly
			LuaRunAndReport(vm, string(translation))

			translation2, err := inc.Tr([]byte(code2))
			panicOn(err)
			fmt.Printf("\n translation2='%s'\n", translation2)

			// and verify that it happens correctly
			LuaRunAndReport(vm, string(translation2))

			LuaMustInt64(vm, "caught", int64(2*(i+1)))
			fmt.Printf("\n caught = %v\n", 2*(i+1))
			cv.So(true, cv.ShouldBeTrue)
		}
	})
}

func fishMultipliesBy(i int) {
	f, err := os.Create("spkg_tst/spkg.go")
	panicOn(err)
	fmt.Fprintf(f, `package spkg_tst

type GONZAGA interface {
	error
}

func Fish(numPole int) (fishCaught int) {
	return numPole * %v
}
`, i)
	f.Close()
}

func Test1002ImportSourcePackageThatLoadsRuntime(t *testing.T) {

	cv.Convey(`import a Go source package that imports 'fmt', and so loads 'runtime' in turn by source, rather than by binary import.`, t, func() {

		code := `
import "github.com/gijit/gi/pkg/compiler/spkg_tst2"
chk := spkg_tst2.Verbose
spkg_tst2.Verbose = true
callres := spkg_tst2.ToString("Hello %s", "world")
`
		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()
		inc := NewIncrState(vm, nil)

		translation, err := inc.Tr([]byte(code))
		panicOn(err)

		// and verify that it happens correctly
		//pp("dump gls just before running translation")
		//LuaRunAndReport(vm, "__gls();")
		pp("above is global env just before we run this translation:")
		fmt.Printf("\n translation='%s'\n", translation)
		LuaRunAndReport(vm, string(translation))

		LuaMustBool(vm, "chk", false)
		LuaMustString(vm, "callres", "Verbose=trueHello world")
		cv.So(true, cv.ShouldBeTrue)
	})
}

func Test1003ImportSourcePackageThatLoadsRuntime(t *testing.T) {

	cv.Convey(`import a Go source package mimicing the problem with the 'runtime' package type checking`, t, func() {

		code := `
import "github.com/gijit/gi/pkg/compiler/spkg_tst3"
`
		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()
		inc := NewIncrState(vm, nil)

		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		fmt.Printf("\n translation='%s'\n", translation)

		// and verify that it happens correctly
		LuaRunAndReport(vm, string(translation))

		//LuaMustInt64(vm, "caught", 4)
		cv.So(true, cv.ShouldBeTrue)
	})
}

func Test1004ImportArrayOfSliceOfBytes(t *testing.T) {

	cv.Convey(`rbuf like import revealed an issue, see spkg_tst4 for minimal example`, t, func() {

		code := `
import "github.com/gijit/gi/pkg/compiler/spkg_tst4"
r := spkg_tst4.NewR();
chk := r.Get1();

`
		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()
		inc := NewIncrState(vm, nil)

		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		fmt.Printf("\n translation='%s'\n", translation)

		// and verify that it happens correctly
		LuaRunAndReport(vm, string(translation))

		LuaMustString(vm, "chk", "world")
		cv.So(true, cv.ShouldBeTrue)
	})
}

func Test1005ArrayOfSliceOfBytes(t *testing.T) {

	cv.Convey(`array of slice of bytes, direct, no import`, t, func() {

		code := `
type R struct {
	A [2][]byte
}

func NewR() *R {
	r := &R{}
	r.A[0] = []byte("hello")
	r.A[1] = []byte("world")
	return r
}

func (r *R) Get1() string {
	return string(r.A[1])
}
r := NewR();
chk := r.Get1();

`
		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()
		inc := NewIncrState(vm, nil)

		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		fmt.Printf("\n translation='%s'\n", translation)

		// and verify that it happens correctly
		LuaRunAndReport(vm, string(translation))

		LuaMustString(vm, "chk", "world")
		cv.So(true, cv.ShouldBeTrue)
	})
}

func Test1006DotImport(t *testing.T) {

	cv.Convey(`import . "spkg_tst5" should work; a dot import`, t, func() {

		code := `
import . "github.com/gijit/gi/pkg/compiler/spkg_tst5"
b := Incr(1)
`
		code2 := `c := Incr(4)`
		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()
		inc := NewIncrState(vm, nil)

		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		fmt.Printf("\n translation='%s'\n", translation)

		// we were repeating the whole package import
		// with every command. Ugh. Don't do that.

		// and verify that it happens correctly
		LuaRunAndReport(vm, string(translation))

		LuaMustInt64(vm, "b", 2)
		cv.So(true, cv.ShouldBeTrue)

		translation2, err := inc.Tr([]byte(code2))
		panicOn(err)
		fmt.Printf("\n translation2='%s'\n", translation2)
		cv.So(string(translation2), matchesLuaSrc,
			`  	c = spkg_tst5.Incr(4LL);`)
		LuaRunAndReport(vm, string(translation2))
		LuaMustInt64(vm, "c", 5)
	})
}
