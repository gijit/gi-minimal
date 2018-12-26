package compiler

import (
	"testing"

	cv "github.com/glycerine/goconvey/convey"
)

func Test1500CallLuaFromGijit(t *testing.T) {
	cv.Convey(`within gijit code: a, err := __lua("3LL + 4LL"); should return int64(7) and nil error`, t, func() {

		src := `
a, err := __lua("3LL + 4LL");
`

		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()
		inc := NewIncrState(vm, nil)

		translation, err := inc.Tr([]byte(src))
		panicOn(err)
		vv("go:'%s'  -->  '%s' in lua\n", src, string(translation))

		LoadAndRunTestHelper(t, vm, translation)

		LuaMustInt64(vm, "a", 7)
		LuaMustBeNilGolangError(vm, "err")
	})
}

func Test1501CallLuaFromGijitPassStrings(t *testing.T) {
	cv.Convey("within gijit code: a, err := __lua(`\"hello \" .. \"world\"`); should return `hello world` and nil error", t, func() {

		src := "a, err := __lua(`\"hello \" .. \"world\"`);"

		vm, err := NewLuaVmWithPrelude(nil)
		panicOn(err)
		defer vm.Close()
		inc := NewIncrState(vm, nil)

		translation, err := inc.Tr([]byte(src))
		panicOn(err)
		vv("go:'%s'  -->  '%s' in lua\n", src, string(translation))

		LoadAndRunTestHelper(t, vm, translation)

		LuaMustString(vm, "a", "hello world")
		LuaMustBeNilGolangError(vm, "err")
	})
}
