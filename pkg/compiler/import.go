package compiler

import (
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/gijit/gi/pkg/importer"
	"github.com/gijit/gi/pkg/token"
	"github.com/gijit/gi/pkg/types"
	"github.com/glycerine/luar"

	golua "github.com/glycerine/golua/lua"

	// shadow_ imports: available inside the REPL

	shadow_bytes "github.com/gijit/gi/pkg/compiler/shadow/bytes"
	shadow_encoding_binary "github.com/gijit/gi/pkg/compiler/shadow/encoding/binary"
	shadow_errors "github.com/gijit/gi/pkg/compiler/shadow/errors"
	shadow_fmt "github.com/gijit/gi/pkg/compiler/shadow/fmt"
	shadow_io "github.com/gijit/gi/pkg/compiler/shadow/io"
	shadow_io_ioutil "github.com/gijit/gi/pkg/compiler/shadow/io/ioutil"
	shadow_math "github.com/gijit/gi/pkg/compiler/shadow/math"
	shadow_math_rand "github.com/gijit/gi/pkg/compiler/shadow/math/rand"
	shadow_os "github.com/gijit/gi/pkg/compiler/shadow/os"
	shadow_reflect "github.com/gijit/gi/pkg/compiler/shadow/reflect"
	shadow_regexp "github.com/gijit/gi/pkg/compiler/shadow/regexp"
	shadow_runtime "github.com/gijit/gi/pkg/compiler/shadow/runtime"
	shadow_runtime_debug "github.com/gijit/gi/pkg/compiler/shadow/runtime/debug"
	shadow_strconv "github.com/gijit/gi/pkg/compiler/shadow/strconv"
	shadow_strings "github.com/gijit/gi/pkg/compiler/shadow/strings"
	shadow_sync "github.com/gijit/gi/pkg/compiler/shadow/sync"
	shadow_sync_atomic "github.com/gijit/gi/pkg/compiler/shadow/sync/atomic"
	shadow_time "github.com/gijit/gi/pkg/compiler/shadow/time"

	// gonum
	shadow_blas "github.com/gijit/gi/pkg/compiler/shadow/gonum.org/v1/gonum/blas"
	shadow_fd "github.com/gijit/gi/pkg/compiler/shadow/gonum.org/v1/gonum/diff/fd"
	shadow_floats "github.com/gijit/gi/pkg/compiler/shadow/gonum.org/v1/gonum/floats"
	shadow_graph "github.com/gijit/gi/pkg/compiler/shadow/gonum.org/v1/gonum/graph"
	shadow_integrate "github.com/gijit/gi/pkg/compiler/shadow/gonum.org/v1/gonum/integrate"
	shadow_lapack "github.com/gijit/gi/pkg/compiler/shadow/gonum.org/v1/gonum/lapack"
	shadow_mat "github.com/gijit/gi/pkg/compiler/shadow/gonum.org/v1/gonum/mat"
	shadow_optimize "github.com/gijit/gi/pkg/compiler/shadow/gonum.org/v1/gonum/optimize"
	shadow_stat "github.com/gijit/gi/pkg/compiler/shadow/gonum.org/v1/gonum/stat"
	shadow_unit "github.com/gijit/gi/pkg/compiler/shadow/gonum.org/v1/gonum/unit"

	// actuals
	"gonum.org/v1/gonum/blas"
	"gonum.org/v1/gonum/diff/fd"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/integrate"
	"gonum.org/v1/gonum/lapack"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/unit"
)

// distinguish binary imports from source imports
var binaryPackage = make(map[string]bool)

func init() {
	a := 1
	b := interface{}(&a)
	_, ok := b.(blas.Complex128)
	_ = ok
}

var _ = ioutil.Discard
var _ = shadow_blas.GijitShadow_InterfaceConvertTo2_Float64
var _ = fd.Backward
var _ = floats.Add
var _ = graph.Copy
var _ = integrate.Trapezoidal
var _ = lapack.None
var _ = mat.Norm
var _ = optimize.ArmijoConditionMet
var _ = stat.CDF
var _ = unit.Atto

func registerLuarReqs(vm *golua.State) {
	// channel ops need reflect, so import it always.

	luar.Register(vm, "reflect", shadow_reflect.Pkg)
	luar.Register(vm, "fmt", shadow_fmt.Pkg)
	//fmt.Printf("reflect/fmt registered\n")

	// give goroutines.lua something to clone
	// to generate select cases.
	refSelCaseVal := reflect.SelectCase{}

	luar.Register(vm, "", luar.Map{
		"__refSelCaseVal": refSelCaseVal,
	})

	registerBasicReflectTypes(vm)

}

func (ic *IncrState) EnableImportsFromLua() {

	// minimize luar stuff for now, focus on pure Lua runtime.

	// running an import means calling the
	// package's __init() function, and entering
	// any Luar bindings into the global namespace.
	goRunImportFromLua := func(path string) {
		// __go_run_import calls here.
		ic.RunTimeGiImportFunc(path, "", 0)
	}

	// compilation allows type checking
	// against the package; it loads an Archive/package
	// into Go memory but does not call __init(). It
	// should not make any changes to the Lua
	// memory or state because no Lua code is actually
	// executed yet.
	goCompileImportFromLua := func(path string) {
		// __go_compile_import calls here.
		ic.CompileTimeGiImportFunc(path, "", 0)
	}

	stacksClosure := func() {
		showLuaStacks(ic.goro.vm)
	}
	luar.Register(ic.goro.vm, "", luar.Map{
		"__go_run_import":     goRunImportFromLua,
		"__go_compile_import": goCompileImportFromLua,
		"__stacks":            stacksClosure,
	})

	// Enable __zygo() calls. Type checking established
	// by getFunFor__callZygo. See incr.go and
	// addPreludeToNewPkg() where that is invoked and
	// the function signature is injected into
	// the toplevel Go scope.
	//
	ic.zlisp = initZygo()
	luar.Register(ic.goro.vm, "", luar.Map{
		"__zygo": func(s string) (interface{}, error) {
			return callZygo(ic.zlisp, s)
		},
	})
}

//////////////
///////////////
//////      ////
//////       ////
//////     ////
///////////////    Import: runtime.
/////////////////
/////       //////      Running an import means calling the
/////        //////  	package's __init() function, and entering
////          //////    any Luar bindings into the global namespace.
////           //////
func (ic *IncrState) RunTimeGiImportFunc(path, pkgDir string, depth int) error {
	pp("RunTimeGiImportFunc called with path = '%s'...", path)

	// if we insist on the eval coroutine, which
	// is already running us (our parent Lua), then the import
	// gets delayed until after the rest of
	// our Lua code finishes. Then that code fails (e.g. test 087)
	// because our import hasn't been done in sequence as
	// programmed. So we must run not on
	// the eval coroutine but immediately.
	useEvalCoroutine := false
	t0 := ic.goro.newTicket("", useEvalCoroutine)

	var srcImport bool
	switch path {
	case "gitesting":
		// test only:
		fmt.Printf("ic.cfg.IsTestMode = %v\n", ic.cfg.IsTestMode)
		if ic.cfg.IsTestMode {
			//fmt.Print("\n registering gitesting.SumArrayInt64! \n")

			t0.regns = "gitesting"
			t0.regmap["SumArrayInt64"] = sumArrayInt64
			t0.regmap["Summer"] = Summer
			t0.regmap["SummerAny"] = SummerAny
			t0.regmap["Incr"] = Incr
			err := t0.Do()
			panicOn(err)
			return err
		}

	case "bytes":
		t0.regmap["bytes"] = shadow_bytes.Pkg
		t0.regmap["__ctor__bytes"] = shadow_bytes.Ctor
		t0.run = append(t0.run, shadow_bytes.InitLua()...)

	case "encoding/binary":
		t0.regmap["binary"] = shadow_encoding_binary.Pkg
		t0.regmap["__ctor__binary"] = shadow_encoding_binary.Ctor
		t0.run = append(t0.run, shadow_encoding_binary.InitLua()...)

	case "errors":
		t0.regmap["errors"] = shadow_errors.Pkg
		t0.regmap["__ctor__errors"] = shadow_errors.Ctor
		t0.run = append(t0.run, shadow_errors.InitLua()...)

	case "fmt":
		pp("RunTimeGiImportFunc sees 'fmt', known and shadowed.")
		t0.regmap["fmt"] = shadow_fmt.Pkg
		t0.regmap["__ctor__fmt"] = shadow_fmt.Ctor
		t0.run = append(t0.run, shadow_fmt.InitLua()...)
	case "io":
		t0.regmap["io"] = shadow_io.Pkg
		t0.regmap["__ctor__io"] = shadow_io.Ctor
		t0.run = append(t0.run, shadow_io.InitLua()...)
	case "math":
		t0.regmap["math"] = shadow_math.Pkg
		t0.regmap["__ctor__math"] = shadow_math.Ctor
		t0.run = append(t0.run, shadow_math.InitLua()...)
	case "math/rand":
		t0.regmap["rand"] = shadow_math_rand.Pkg
		t0.regmap["__ctor__math_rand"] = shadow_math_rand.Ctor
		t0.run = append(t0.run, shadow_math_rand.InitLua()...)
	case "os":
		t0.regmap["os"] = shadow_os.Pkg
		t0.regmap["__ctor__os"] = shadow_os.Ctor
		t0.run = append(t0.run, shadow_os.InitLua()...)

	case "reflect":
		t0.regmap["reflect"] = shadow_reflect.Pkg
		t0.regmap["__ctor__reflect"] = shadow_reflect.Ctor
		t0.run = append(t0.run, shadow_reflect.InitLua()...)

	case "regexp":
		t0.regmap["regexp"] = shadow_regexp.Pkg
		t0.regmap["__ctor__regexp"] = shadow_regexp.Ctor
		t0.run = append(t0.run, shadow_regexp.InitLua()...)

	case "sync":
		t0.regmap["sync"] = shadow_sync.Pkg
		t0.regmap["__ctor__sync"] = shadow_sync.Ctor
		t0.run = append(t0.run, shadow_sync.InitLua()...)

	case "sync/atomic":
		t0.regmap["atomic"] = shadow_sync_atomic.Pkg
		t0.regmap["__ctor__atomic"] = shadow_sync_atomic.Ctor
		t0.run = append(t0.run, shadow_sync_atomic.InitLua()...)

	case "time":
		t0.regmap["time"] = shadow_time.Pkg
		t0.regmap["__ctor__time"] = shadow_time.Ctor
		t0.run = append(t0.run, shadow_time.InitLua()...)

	case "runtime":
		t0.regmap["runtime"] = shadow_runtime.Pkg
		t0.regmap["__ctor__runtime"] = shadow_runtime.Ctor
		t0.run = append(t0.run, shadow_runtime.InitLua()...)

	case "runtime/debug":
		t0.regmap["debug"] = shadow_runtime_debug.Pkg
		t0.regmap["__ctor__debug"] = shadow_runtime_debug.Ctor
		t0.run = append(t0.run, shadow_runtime_debug.InitLua()...)

	case "strconv":
		t0.regmap["strconv"] = shadow_strconv.Pkg
		t0.regmap["__ctor__strconv"] = shadow_strconv.Ctor
		t0.run = append(t0.run, shadow_strconv.InitLua()...)

	case "strings":
		t0.regmap["strings"] = shadow_strings.Pkg
		t0.regmap["__ctor__strings"] = shadow_strings.Ctor
		t0.run = append(t0.run, shadow_strings.InitLua()...)

	case "io/ioutil":
		t0.regmap["ioutil"] = shadow_io_ioutil.Pkg
		t0.regmap["__ctor__ioutil"] = shadow_io_ioutil.Ctor
		t0.run = append(t0.run, shadow_io_ioutil.InitLua()...)

		// gonum:
	case "gonum.org/v1/gonum/blas":
		t0.regmap["blas"] = shadow_blas.Pkg
	case "gonum.org/v1/gonum/fd":
		t0.regmap["fd"] = shadow_fd.Pkg
	case "gonum.org/v1/gonum/floats":
		t0.regmap["floats"] = shadow_floats.Pkg
	case "gonum.org/v1/gonum/graph":
		t0.regmap["graph"] = shadow_graph.Pkg
	case "gonum.org/v1/gonum/integrate":
		t0.regmap["integrate"] = shadow_integrate.Pkg
	case "gonum.org/v1/gonum/lapack":
		t0.regmap["lapack"] = shadow_lapack.Pkg
	case "gonum.org/v1/gonum/mat":
		t0.regmap["mat"] = shadow_mat.Pkg
	case "gonum.org/v1/gonum/optimize":
		t0.regmap["optimize"] = shadow_optimize.Pkg
	case "gonum.org/v1/gonum/stat":
		t0.regmap["stat"] = shadow_stat.Pkg
	case "gonum.org/v1/gonum/unit":
		t0.regmap["unit"] = shadow_unit.Pkg

	default:
		// source import
		srcImport = true
		// don't need to compile again, just call pkg.__init()
		t0.run = []byte(fmt.Sprintf("%s.__init();", omitAnyShadowPathPrefix(path, true)))
	}
	if !srcImport {
		binaryPackage[path] = true
	}
	err := t0.Do()
	pp("RunTimeGiImportFunc executed t0.Do() to run: '%s', got back err='%v'", string(t0.run), err)
	return err
}

///////////////////
///////////////////
//////
//////  Import: compile time
//////
///////////////////
///////////////////
func (ic *IncrState) CompileTimeGiImportFunc(path, pkgDir string, depth int) (*Archive, error) {
	pp("CompileTimeGiImportFunc called with path = '%s'... depth=%v", path, depth)

	// `import "fmt"` means that path == "fmt", for example.

	// check cache first
	arch, ok := ic.Session.Archives[path]
	_, _ = arch, ok
	/*
		if ok {
			pp("ic.CompileTimeGiImportFunc cache hit for path '%s'", path)
			return arch, nil
		}
	*/
	pp("no cache hit for path '%s'", path)

	code := []byte(fmt.Sprintf("\t __go_run_import(\"%[1]s\");\n\t __type__.%[2]s = __type__.%[2]s or {};\n", omitAnyShadowPathPrefix(path, false), omitAnyShadowPathPrefix(path, true)))
	//code := []byte(fmt.Sprintf("\t __go_run_import(\"%[1]s\");\n\t __type__.%[2]s = __type__.%[2]s or {};\n\t local %[2]s = _G.%[2]s;\n", omitAnyShadowPathPrefix(path, false), omitAnyShadowPathPrefix(path, true)))

	switch path {

	// all these are shadowed, see below for the load of type checking info.

	// gen-gijit-shadow outputs to pkg/compiler/shadow/...
	case "bytes":
	case "encoding/binary":
	case "errors":
	case "fmt":
	case "io":
	case "math":
	case "math/rand":
	case "os":
	case "reflect":
	case "regexp":
	case "sync":
	case "sync/atomic":
	case "time":
	case "runtime":
	case "runtime/debug":
	case "strconv":
	case "strings":
	case "io/ioutil":
		// gonum:
	case "gonum.org/v1/gonum/blas":
	case "gonum.org/v1/gonum/fd":
	case "gonum.org/v1/gonum/floats":
	case "gonum.org/v1/gonum/graph":
	case "gonum.org/v1/gonum/integrate":
	case "gonum.org/v1/gonum/lapack":
	case "gonum.org/v1/gonum/mat":
	case "gonum.org/v1/gonum/optimize":
	case "gonum.org/v1/gonum/stat":
	case "gonum.org/v1/gonum/unit":

		// we need to load the type-checking info into arch.Pkg
		// now so that the compile can complete.

	case "gitesting":
		// test only:
		fmt.Printf("ic.cfg.IsTestMode = %v\n", ic.cfg.IsTestMode)
		if ic.cfg.IsTestMode {
			//fmt.Print("\n registering gitesting.SumArrayInt64! \n")
			pkg := types.NewPackage("gitesting", "gitesting")
			pkg.MarkComplete()
			scope := pkg.Scope()

			suma := getFunForSumArrayInt64(pkg)
			scope.Insert(suma)

			summer := getFunForSummer(pkg)
			scope.Insert(summer)

			summerAny := getFunForSummerAny(pkg)
			scope.Insert(summerAny)

			incr := getFunForIncr(pkg)
			scope.Insert(incr)

			a := &Archive{
				SavedArchive: SavedArchive{
					ImportPath: path,
				},
				NewCodeText: [][]byte{code},
				Pkg:         pkg,
			}
			a.Pkg.ClientExtra = a
			ic.CurPkg.importContext.Packages[path] = pkg
			ic.Session.Archives[path] = a
			ic.Session.Archives[pkg.Path()] = a

			pp("a.NewCodeText='%v'", string(a.NewCodeText[0]))
			return a, nil
		}

	default:
		// try a source import?

		p1("should we source import path='%s'? depth=%v", path, depth)
		pp("stack ='%s'\n", stack())

		if depth > 7 {
			// not allowed
			return nil, fmt.Errorf("deep source imports forbidden for performance reasons. problem with import of package '%s' (not shadowed? [1]) depth=%v ... [footnote 1] To shadow it, run gen-gijit-shadow-import on the package, add a case and import above, and recompile gijit.", path, depth)
		}

		archive, err := ic.ImportSourcePackage(path, pkgDir, depth+1)

		pp("CompileTimeGiImportFunc: upon return from ic.ImportSourcePackage(path='%s'), here is the global env:", path)
		//ic.Session.showGlobal()

		if err == nil {
			if archive == nil {
				panic("why was archive nil if err was nil?")
			}
			if archive.Pkg == nil {
				panic("why was archive.Pkg nil if err was nil?")
			}
			// success at source import.

			pp("calling WriteCommandPackage")
			isMain := false
			code, err = ic.Session.WriteCommandPackage(archive, "", isMain)
			p1("back from WriteCommandPackage for path='%s', err='%v', code is\n'%s'", path, err, string(code))
			// fmt is okay here.
			if err != nil {
				return nil, err
			}

			pp("CompileTimeGiImportFunc: upon return from ic.Session.WriteCommandPackage() for path='%s', here is the global env:", path)
			//ic.Session.showGlobal()

			archive.NewCodeText = [][]byte{code}
			archive.Pkg.ClientExtra = archive

			return archive, nil
		}
		// source import failed.
		fmt.Printf("source import of package '%s' failed: '%v'", path, err)

		// need to run gen-gijit-shadow-import
		return nil, fmt.Errorf("error on import: problem with package '%s' (not shadowed? [1]): '%v'. ... [footnote 1] To shadow it, run gen-gijit-shadow-import on the package, add a case and import above, and recompile gijit.", path, err)
	}

	// successfully match path to a shadow package, bring in its
	// type info now.
	shadowPath := "github.com/gijit/gi/pkg/compiler/shadow/" + path
	a, err := ic.ActuallyImportPackage(path, "", shadowPath, depth+1)
	if err != nil {
		return nil, err
	}
	a.NewCodeText = [][]byte{code}
	a.Pkg.ClientExtra = a
	return a, nil
}

func omitAnyShadowPathPrefix(pth string, base bool) string {
	const prefix = "github.com/gijit/gi/pkg/compiler/shadow/"
	if strings.HasPrefix(pth, prefix) {
		if base {
			return path.Base(pth[len(prefix):])
		}
		return pth[len(prefix):]
	}
	if base {
		return path.Base(pth)
	}
	return pth
}

func getFunForSprintf(pkg *types.Package) *types.Func {
	// func Sprintf(format string, a ...interface{}) string
	var recv *types.Var
	var T types.Type = &types.Interface{}
	str := types.Typ[types.String]
	results := types.NewTuple(types.NewVar(token.NoPos, pkg, "", str))
	params := types.NewTuple(types.NewVar(token.NoPos, pkg, "format", str),
		types.NewVar(token.NoPos, pkg, "a", types.NewSlice(T)))
	variadic := true
	sig := types.NewSignature(recv, params, results, variadic)
	fun := types.NewFunc(token.NoPos, pkg, "Sprintf", sig)
	return fun
}

func getFunForPrintf(pkg *types.Package) *types.Func {
	// func Sprintf(format string, a ...interface{}) string
	var recv *types.Var
	var T types.Type = &types.Interface{}
	str := types.Typ[types.String]
	nt := types.Typ[types.Int]
	errt := types.Universe.Lookup("error")
	if errt == nil {
		panic("could not locate error interface in types.Universe")
	}
	results := types.NewTuple(types.NewVar(token.NoPos, pkg, "", nt),
		types.NewVar(token.NoPos, pkg, "", errt.Type()))
	params := types.NewTuple(types.NewVar(token.NoPos, pkg, "format", str),
		types.NewVar(token.NoPos, pkg, "a", types.NewSlice(T)))
	variadic := true
	sig := types.NewSignature(recv, params, results, variadic)
	fun := types.NewFunc(token.NoPos, pkg, "Printf", sig)
	return fun
}

func Summer(a, b int) int {
	return a + b
}

func getFunForSummer(pkg *types.Package) *types.Func {
	// func Summer(a, b int) int
	var recv *types.Var
	nt := types.Typ[types.Int]
	results := types.NewTuple(types.NewVar(token.NoPos, pkg, "", nt))
	params := types.NewTuple(types.NewVar(token.NoPos, pkg, "a", nt),
		types.NewVar(token.NoPos, pkg, "b", nt))
	variadic := false
	sig := types.NewSignature(recv, params, results, variadic)
	fun := types.NewFunc(token.NoPos, pkg, "Summer", sig)
	return fun
}

func SummerAny(a ...int) int {
	fmt.Printf("top of SummaryAny, a is len %v\n", len(a))
	tot := 0
	for i := range a {
		tot += a[i]
	}
	fmt.Printf("end of SummaryAny, returning tot=%v\n", tot)
	return tot
}

func getFunForSummerAny(pkg *types.Package) *types.Func {
	// func Summer(a, b int) int
	var recv *types.Var
	nt := types.Typ[types.Int]
	results := types.NewTuple(types.NewVar(token.NoPos, pkg, "", nt))
	params := types.NewTuple(types.NewVar(token.NoPos, pkg, "a", types.NewSlice(nt)))
	variadic := true
	sig := types.NewSignature(recv, params, results, variadic)
	fun := types.NewFunc(token.NoPos, pkg, "SummerAny", sig)
	return fun
}

func getFunForSumArrayInt64(pkg *types.Package) *types.Func {
	// func sumArrayInt64(a [3]int64) (tot int64)
	var recv *types.Var
	nt64 := types.Typ[types.Int64]
	results := types.NewTuple(types.NewVar(token.NoPos, pkg, "tot", nt64))
	params := types.NewTuple(types.NewVar(token.NoPos, pkg, "a", types.NewArray(nt64, 3)))
	variadic := false
	sig := types.NewSignature(recv, params, results, variadic)
	fun := types.NewFunc(token.NoPos, pkg, "SumArrayInt64", sig)
	return fun
}

func getFunFor__tostring(pkg *types.Package) *types.Func {
	// func Tostring(a interface{}) string
	var recv *types.Var
	str := types.Typ[types.String]
	results := types.NewTuple(types.NewVar(token.NoPos, pkg, "", str))
	emptyInterface := types.NewInterface(nil, nil)
	params := types.NewTuple(types.NewVar(token.NoPos, pkg, "a", emptyInterface))
	variadic := false
	sig := types.NewSignature(recv, params, results, variadic)
	fun := types.NewFunc(token.NoPos, pkg, "__tostring", sig)
	return fun
}

// make the lua __st (show table) utility available in Go land.
func getFunFor__st(pkg *types.Package) *types.Func {
	// func __st(a interface{}) string
	var recv *types.Var
	str := types.Typ[types.String]
	results := types.NewTuple(types.NewVar(token.NoPos, pkg, "", str))
	emptyInterface := types.NewInterface(nil, nil)
	params := types.NewTuple(types.NewVar(token.NoPos, pkg, "a", emptyInterface))
	variadic := false
	sig := types.NewSignature(recv, params, results, variadic)
	fun := types.NewFunc(token.NoPos, pkg, "__st", sig)
	return fun
}

// and __ls, __gls, __lst, __glst; all functions with no arguments
// and no results, just side effect of displaying info.
func getFunFor__replUtil(cmd string, pkg *types.Package) *types.Func {
	// func __ls()
	sig := types.NewSignature(nil, nil, nil, false)
	fun := types.NewFunc(token.NoPos, pkg, cmd, sig)
	return fun
}

func Incr(a int) int {
	fmt.Printf("\nYAY Incr(a) called! with a = '%v'\n", a)
	return a + 1
}

func getFunForIncr(pkg *types.Package) *types.Func {
	// func Incr(a int) int
	var recv *types.Var
	nt := types.Typ[types.Int]
	results := types.NewTuple(types.NewVar(token.NoPos, pkg, "", nt))
	params := types.NewTuple(types.NewVar(token.NoPos, pkg, "a", nt))
	variadic := false
	sig := types.NewSignature(recv, params, results, variadic)
	fun := types.NewFunc(token.NoPos, pkg, "Incr", sig)
	return fun
}

// We use the go/importer to load the compiled form of
// the package. This reads from the
// last built binary .a lib on disk. Warning: this might
// be out of date. Later we might read source using the
// go/loader from tools/x, to be most up to date.
// However, the binary loader is *much* faster.
//
// dir provides where to import from, to honor vendored packages.
func (ic *IncrState) ActuallyImportPackage(path, dir, shadowPath string, depth int) (*Archive, error) {
	pp("IncrState.ActuallyImportPackage(path='%s', dir='%s', shadowPath='%s'", path, dir, shadowPath)
	pp("stack='%s'", string(debug.Stack()))
	var pkg *types.Package

	//imp := importer.For("source", nil) // Default()
	// faster than source importing is reading the binary.
	imp := importer.Default()
	imp2, ok := imp.(types.ImporterFrom)
	if !ok {
		panic("importer.ImportFrom not available, vendored packages would be lost")
	}
	var mode types.ImportMode
	var err error
	pkg, err = imp2.ImportFrom(path, dir, mode, depth)

	if err != nil {
		return nil, err
	}

	pkgName := pkg.Name()

	res := &Archive{
		SavedArchive: SavedArchive{
			Name:       pkgName,
			ImportPath: path,
		},
		Pkg: pkg,
	}

	pkg.SetPath(shadowPath)

	// very important, must do this or we won't locate the package!
	ic.CurPkg.importContext.Packages[path] = pkg
	ic.Session.Archives[path] = res
	ic.Session.Archives[pkg.Path()] = res

	return res, nil
}

// __gijit_printQuoted(a ...interface{})
func getFunForGijitPrintQuoted(pkg *types.Package) *types.Func {
	// func __gijit_printQuoted(a ...interface{})
	var recv *types.Var
	var T types.Type = &types.Interface{}
	results := types.NewTuple()
	params := types.NewTuple(types.NewVar(token.NoPos, pkg, "a", types.NewSlice(T)))
	variadic := true
	sig := types.NewSignature(recv, params, results, variadic)
	fun := types.NewFunc(token.NoPos, pkg, "__gijit_printQuoted", sig)
	return fun
}

// __lua: just skip compilation and
//            pass through as compiled code directly
func getFunFor__callLua(pkg *types.Package) *types.Func {
	// func __lua(s string) (interface{}, error)
	var recv *types.Var
	emptyInterface := types.NewInterface(nil, nil)
	errTypeName := types.Universe.Lookup("error").(*types.TypeName)
	results := types.NewTuple(
		types.NewVar(token.NoPos, pkg, "", emptyInterface),
		types.NewVar(token.NoPos, pkg, "", errTypeName.Type()),
	)
	str := types.Typ[types.String]
	params := types.NewTuple(types.NewVar(token.NoPos, pkg, "s", str))
	variadic := false
	sig := types.NewSignature(recv, params, results, variadic)
	fun := types.NewFunc(token.NoPos, pkg, "__lua", sig)
	return fun
}

// __zygo: pass to string zygo, return result and error.
//
func getFunFor__callZygo(pkg *types.Package) *types.Func {
	// func __zygo(s string) (interface{}, error)
	var recv *types.Var
	emptyInterface := types.NewInterface(nil, nil)
	errTypeName := types.Universe.Lookup("error").(*types.TypeName)
	results := types.NewTuple(
		types.NewVar(token.NoPos, pkg, "", emptyInterface),
		types.NewVar(token.NoPos, pkg, "", errTypeName.Type()),
	)
	str := types.Typ[types.String]
	params := types.NewTuple(types.NewVar(token.NoPos, pkg, "s", str))
	variadic := false
	sig := types.NewSignature(recv, params, results, variadic)
	fun := types.NewFunc(token.NoPos, pkg, "__zygo", sig)
	return fun
}

/* time stuff

var $setTimeout = function(f, t) {
  $awakeGoroutines++;
  return setTimeout(function() {
    $awakeGoroutines--;
    f();
  }, t);
};

func Sleep(d Duration) {
	c := make(chan struct{})
	js.Global.Call("$setTimeout", js.InternalObject(func() { close(c) }), int(d/Millisecond))
	<-c
}


func startTimer(t *runtimeTimer) {
	t.active = true
	diff := (t.when - runtimeNano()) / int64(Millisecond)
	if diff > 1<<31-1 { // math.MaxInt32
		return
	}
	if diff < 0 {
		diff = 0
	}
	t.timeout = js.Global.Call("$setTimeout", js.InternalObject(func() {
		t.active = false
		if t.period != 0 {
			t.when += t.period
			startTimer(t)
		}
		go t.f(t.arg, 0)
	}), diff+1)
}

func stopTimer(t *runtimeTimer) bool {
	js.Global.Call("clearTimeout", t.timeout)
	wasActive := t.active
	t.active = false
	return wasActive
}

*/
