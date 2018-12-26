package compiler

// select_test tests the reflect-based channel
// system.

// For the moment, we'll comment this out while
// we get the fully lua system going.
/*
import (
	"fmt"
	"testing"

	"github.com/gijit/gi/pkg/token"
	"github.com/gijit/gi/pkg/types"
	cv "github.com/glycerine/goconvey/convey"
	"github.com/glycerine/luar"
)

func Test600RecvOnChannel(t *testing.T) {

	cv.Convey(`in Lua, receive an integer on a buffered channel, previously sent by Go`, t, func() {

		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()

		ch := make(chan int, 1)
		ch <- 57

		luar.Register(vm, "", luar.Map{
			"ch": ch,
		})

		// first run instantiates the main package so we can add 'ch' to it.
		code := `b := 3`
		inc := NewIncrState(vm, nil)
		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		pp("translation='%s'", string(translation))
		LuaRunAndReport(vm, string(translation))
		LuaMustInt64(vm, "b", 3)

		// allow ch to type check
		pkg := inc.pkgMap["main"].Arch.Pkg
		scope := pkg.Scope()
		nt64 := types.Typ[types.Int64]
		chVar := types.NewVar(token.NoPos, pkg, "ch", types.NewChan(types.SendRecv, nt64))
		scope.Insert(chVar)

		code = `a := <- ch;`

		translation, err = inc.Tr([]byte(code))
		panicOn(err)

		pp("translation='%s'", string(translation))

		LuaRunAndReport(vm, string(translation))
		LuaMustInt64(vm, "a", 57)
		cv.So(true, cv.ShouldBeTrue)
	})
}

func Test601SendOnChannel(t *testing.T) {

	cv.Convey(`in Lua, send to a buffered channel`, t, func() {

		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()

		ch := make(chan int, 1)

		luar.Register(vm, "", luar.Map{
			"ch": ch,
		})

		// first run instantiates the main package so we can add 'ch' to it.
		code := `b := 3`
		inc := NewIncrState(vm, nil)
		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		pp("translation='%s'", string(translation))
		LuaRunAndReport(vm, string(translation))
		LuaMustInt64(vm, "b", 3)

		// allow ch to type check
		pkg := inc.pkgMap["main"].Arch.Pkg
		scope := pkg.Scope()
		nt64 := types.Typ[types.Int64]
		chVar := types.NewVar(token.NoPos, pkg, "ch", types.NewChan(types.SendRecv, nt64))
		scope.Insert(chVar)

		code = `pre:= true; ch <- 6;`

		translation, err = inc.Tr([]byte(code))
		panicOn(err)

		pp("translation='%s'", string(translation))

		LuaRunAndReport(vm, string(translation))

		a := <-ch
		fmt.Printf("a received! a = %v\n", a)
		cv.So(a, cv.ShouldEqual, 6)
		LuaMustBool(vm, "pre", true)
	})
}

func Test602BlockingSendOnChannel(t *testing.T) {

	cv.Convey(`in Lua, send to an unbuffered channel`, t, func() {

		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()

		ch := make(chan int)

		luar.Register(vm, "", luar.Map{
			"ch": ch,
		})

		// first run instantiates the main package so we can add 'ch' to it.
		code := `b := 3`
		inc := NewIncrState(vm, nil)
		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		pp("translation='%s'", string(translation))
		LuaRunAndReport(vm, string(translation))
		LuaMustInt64(vm, "b", 3)

		// allow ch to type check
		pkg := inc.pkgMap["main"].Arch.Pkg
		scope := pkg.Scope()
		nt64 := types.Typ[types.Int64]
		chVar := types.NewVar(token.NoPos, pkg, "ch", types.NewChan(types.SendRecv, nt64))
		scope.Insert(chVar)

		var a int
		done := make(chan bool)
		go func() {
			a = <-ch
			close(done)
		}()

		code = `pre:= true; ch <- 7;`

		translation, err = inc.Tr([]byte(code))
		panicOn(err)

		pp("translation='%s'", string(translation))

		LuaRunAndReport(vm, string(translation))

		<-done
		fmt.Printf("a received! a = %v\n", a)
		cv.So(a, cv.ShouldEqual, 7)
		LuaMustBool(vm, "pre", true)
	})
}

func Test603RecvOnUnbufferedChannel(t *testing.T) {

	cv.Convey(`in Lua, receive an integer on an unbuffered channel, previously sent by Go`, t, func() {

		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()

		ch := make(chan int)

		luar.Register(vm, "", luar.Map{
			"ch": ch,
		})

		// first run instantiates the main package so we can add 'ch' to it.
		code := `b := 3`
		inc := NewIncrState(vm, nil)
		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		pp("translation='%s'", string(translation))
		LuaRunAndReport(vm, string(translation))
		LuaMustInt64(vm, "b", 3)

		// allow ch to type check
		pkg := inc.pkgMap["main"].Arch.Pkg
		scope := pkg.Scope()
		nt64 := types.Typ[types.Int64]
		chVar := types.NewVar(token.NoPos, pkg, "ch", types.NewChan(types.SendRecv, nt64))
		scope.Insert(chVar)

		code = `a := <- ch;`

		translation, err = inc.Tr([]byte(code))
		panicOn(err)

		pp("translation='%s'", string(translation))

		go func() {
			ch <- 16
		}()
		LuaRunAndReport(vm, string(translation))

		LuaMustInt64(vm, "a", 16)
		cv.So(true, cv.ShouldBeTrue)
	})
}

func Test604Select(t *testing.T) {

	cv.Convey(`in Lua, select on two channels`, t, func() {

		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()

		chInt := make(chan int)
		chStr := make(chan string)

		luar.Register(vm, "", luar.Map{
			"chInt": chInt,
			"chStr": chStr,
		})

		// first run instantiates the main package so we can add 'ch' to it.
		code := `b := 3`
		inc := NewIncrState(vm, nil)
		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		pp("translation='%s'", string(translation))
		LuaRunAndReport(vm, string(translation))
		LuaMustInt64(vm, "b", 3)

		// allow channels to type check
		pkg := inc.pkgMap["main"].Arch.Pkg
		scope := pkg.Scope()

		nt := types.Typ[types.Int]
		chIntVar := types.NewVar(token.NoPos, pkg, "chInt", types.NewChan(types.SendRecv, nt))
		scope.Insert(chIntVar)

		stringType := types.Typ[types.String]
		chStrVar := types.NewVar(token.NoPos, pkg, "chStr", types.NewChan(types.SendRecv, stringType))
		scope.Insert(chStrVar)

		code = `
a := 0
b := ""
for i := 0; i < 2; i++ {
  select {
    case a = <- chInt:
    case b = <- chStr:
  }
}
`
		translation, err = inc.Tr([]byte(code))
		panicOn(err)
		pp("translation='%s'", string(translation))

		go func() {
			chInt <- 43
		}()
		go func() {
			chStr <- "hello select"
		}()
		LuaRunAndReport(vm, string(translation))

		LuaMustString(vm, "b", "hello select")
		LuaMustInt64(vm, "a", 43)
		cv.So(true, cv.ShouldBeTrue)
	})
}

func Test605Select(t *testing.T) {

	cv.Convey(`in Lua, select with send and recv`, t, func() {

		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()

		chInt := make(chan int)
		chStr := make(chan string)

		luar.Register(vm, "", luar.Map{
			"chInt": chInt,
			"chStr": chStr,
		})

		// first run instantiates the main package so we can add 'ch' to it.
		code := `b := 3`
		inc := NewIncrState(vm, nil)
		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		pp("translation='%s'", string(translation))
		LuaRunAndReport(vm, string(translation))
		LuaMustInt64(vm, "b", 3)

		// allow channels to type check
		pkg := inc.pkgMap["main"].Arch.Pkg
		scope := pkg.Scope()

		nt := types.Typ[types.Int]
		chIntVar := types.NewVar(token.NoPos, pkg, "chInt", types.NewChan(types.SendRecv, nt))
		scope.Insert(chIntVar)

		stringType := types.Typ[types.String]
		chStrVar := types.NewVar(token.NoPos, pkg, "chStr", types.NewChan(types.SendRecv, stringType))
		scope.Insert(chStrVar)

		code = `
a := 0
b := 0
c := 0
igot := 0
done := false

for a ==0 || b==0 || c==0 {
  select {
    case igot = <- chInt:
       a=1
    case chStr <- "yumo":
       //send happend
       b=1
    default:
       c=1
  }
}
done = true
`
		translation, err = inc.Tr([]byte(code))
		panicOn(err)
		pp("translation='%s'", string(translation))

		go func() {
			chInt <- 43
		}()
		strReceived := ""
		go func() {
			strReceived = <-chStr
		}()
		LuaRunAndReport(vm, string(translation))

		cv.So(strReceived, cv.ShouldEqual, "yumo")

		LuaMustInt64(vm, "a", 1)
		LuaMustInt64(vm, "b", 1)
		LuaMustInt64(vm, "c", 1)
		LuaMustBool(vm, "done", true)
		LuaMustInt64(vm, "igot", 43)
	})
}

func Test606MakeChannel(t *testing.T) {

	cv.Convey(`in Lua, make a new channel. In Go, extract it and use it to communicate with Lua.`, t, func() {

		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()

		code := `ch := make(chan int, 1); ch <- 23; a := <- ch`
		inc := NewIncrState(vm, nil)
		translation, err := inc.Tr([]byte(code))
		panicOn(err)
		pp("translation='%s'", string(translation))
		LuaRunAndReport(vm, string(translation))

		LuaMustInt64(vm, "a", 23)

		// get the 'ch' channel out and use it in Go.
		iface, err := getChannelFromGlobal(vm, "ch", false)
		panicOn(err)

		var ch chan int = iface.(chan int)

		// verify as buffered
		ch <- 12
		geti := <-ch
		pp("geti = '%v'", geti)
		cv.So(geti, cv.ShouldEqual, 12)

		// from lua do a send, receive in Go.

		code = `ch <- 17;`
		translation, err = inc.Tr([]byte(code))
		panicOn(err)
		pp("translation='%s'", string(translation))
		LuaRunAndReport(vm, string(translation))

		got := <-ch
		cv.So(got, cv.ShouldEqual, 17)

		// send from Go, receive in Lua
		ch <- 9

		code = `nine := <- ch;`
		translation, err = inc.Tr([]byte(code))
		panicOn(err)
		pp("translation='%s'", string(translation))
		LuaRunAndReport(vm, string(translation))
		LuaMustInt64(vm, "nine", 9)

		//note this utility: can be used to generate a chan interface{}
		//luar.MakeChan(L1)
		//var c interface{}
		//_, err := luar.LuaToGo(L1, -1, &c)
	})
}
*/
