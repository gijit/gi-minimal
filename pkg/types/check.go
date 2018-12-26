// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements the Check function, which drives type-checking.

package types

import (
	"github.com/gijit/gi/pkg/ast"
	"github.com/gijit/gi/pkg/constant"
	"github.com/gijit/gi/pkg/token"
)

// debugging/development support
const (
	debug = false // leave on during development
	trace = false // turn on for detailed type resolution traces
)

// If Strict is set, the type-checker enforces additional
// rules not specified by the Go 1 spec, but which will
// catch guaranteed run-time errors if the respective
// code is executed. In other words, programs passing in
// Strict mode are Go 1 compliant, but not all Go 1 programs
// will pass in Strict mode. The additional rules are:
//
// - A type assertion x.(T) where T is an interface type
//   is invalid if any (statically known) method that exists
//   for both x and T have different signatures.
//
const strict = false

// exprInfo stores information about an untyped expression.
type exprInfo struct {
	isLhs bool // expression is lhs operand of a shift with delayed type-check
	mode  operandMode
	typ   *Basic
	val   constant.Value // constant value; or nil (if not a constant)
}

// funcInfo stores the information required for type-checking a function.
type funcInfo struct {
	name string    // for debugging/tracing only
	decl *DeclInfo // for cycle detection
	sig  *Signature
	body *ast.BlockStmt
}

// A context represents the context within which an object is type-checked.
type context struct {
	decl          *DeclInfo      // package-level declaration whose init expression/function body is checked
	scope         *Scope         // top-most scope for lookups
	iota          constant.Value // value of iota in a constant declaration; nil otherwise
	sig           *Signature     // function signature if inside a function; nil otherwise
	hasLabel      bool           // set if a function makes use of labels (only ~1% of functions); unused outside functions
	hasCallOrRecv bool           // set if an expression contains a function call or channel receive operation
}

// An importKey identifies an imported package by import path and source directory
// (directory containing the file containing the import). In practice, the directory
// may always be the same, or may not matter. Given an (import path, directory), an
// importer must always return the same package (but given two different import paths,
// an importer may still return the same package by mapping them to the same package
// paths).
type importKey struct {
	path, dir string
}

// A Checker maintains the state of the type checker.
// It must be created with NewChecker.
type Checker struct {
	// package information
	// (initialized by NewChecker, valid for the life-time of checker)
	conf *Config
	fset *token.FileSet
	pkg  *Package
	*Info
	ObjMap map[Object]*DeclInfo   // maps package-level object to declaration info
	impMap map[importKey]*Package // maps (import path, source directory) to (complete or fake) package

	// information collected during type-checking of a set of package files
	// (initialized by Files, valid only for the duration of check.Files;
	// maps and lists are allocated on demand)
	files            []*ast.File                       // package files
	unusedDotImports map[*Scope]map[*Package]token.Pos // positions of unused dot-imported packages for each file scope

	firstErr error                 // first error encountered
	Methods  map[string][]*Func    // maps type names to associated methods
	untyped  map[ast.Expr]exprInfo // map of expressions without final type
	funcs    []funcInfo            // list of functions to type-check
	delayed  []func()              // delayed checks requiring fully setup types

	// context within which the current object is type-checked
	// (valid only for the duration of type-checking a specific object)
	context
	pos token.Pos // if valid, identifiers are looked up as if at position pos (used by Eval)

	// debugging
	indent int // indentation for tracing
}

// addUnusedImport adds the position of a dot-imported package
// pkg to the map of dot imports for the given file scope.
func (check *Checker) addUnusedDotImport(scope *Scope, pkg *Package, pos token.Pos) {
	mm := check.unusedDotImports
	if mm == nil {
		mm = make(map[*Scope]map[*Package]token.Pos)
		check.unusedDotImports = mm
	}
	m := mm[scope]
	if m == nil {
		m = make(map[*Package]token.Pos)
		mm[scope] = m
	}
	m[pkg] = pos
}

// addDeclDep adds the dependency edge (check.decl -> to) if check.decl exists
func (check *Checker) addDeclDep(to Object) {
	from := check.decl
	if from == nil {
		return // not in a package-level init expression
	}
	if _, found := check.ObjMap[to]; !found {
		return // to is not a package-level object
	}
	from.addDep(to)
}

func (check *Checker) assocMethod(tname string, meth *Func) {
	m := check.Methods
	if m == nil {
		m = make(map[string][]*Func)
		check.Methods = m
	}
	m[tname] = append(m[tname], meth)
}

func (check *Checker) rememberUntyped(e ast.Expr, lhs bool, mode operandMode, typ *Basic, val constant.Value) {
	m := check.untyped
	if m == nil {
		m = make(map[ast.Expr]exprInfo)
		check.untyped = m
	}
	m[e] = exprInfo{lhs, mode, typ, val}
}

func (check *Checker) later(name string, decl *DeclInfo, sig *Signature, body *ast.BlockStmt) {
	check.funcs = append(check.funcs, funcInfo{name, decl, sig, body})
}

func (check *Checker) delay(f func()) {
	check.delayed = append(check.delayed, f)
}

// NewChecker returns a new Checker instance for a given package.
// Package files may be added incrementally via checker.Files.
func NewChecker(conf *Config, fset *token.FileSet, pkg *Package, info *Info) *Checker {
	// make sure we have a configuration
	if conf == nil {
		conf = new(Config)
	}

	// make sure we have an info struct
	if info == nil {
		info = new(Info)
	}

	return &Checker{
		conf:   conf,
		fset:   fset,
		pkg:    pkg,
		Info:   info,
		ObjMap: make(map[Object]*DeclInfo),
		impMap: make(map[importKey]*Package),
	}
}

// initFiles initializes the files-specific portion of checker.
// The provided files must all belong to the same package.
func (check *Checker) initFiles(files []*ast.File) {
	// start with a clean slate (check.Files may be called multiple times)
	check.files = nil
	check.unusedDotImports = nil

	check.firstErr = nil
	check.Methods = nil
	check.untyped = nil
	check.funcs = nil
	check.delayed = nil

	// determine package name and collect valid files
	pkg := check.pkg
	_ = pkg
	for _, file := range files {
		// jea: turn off package checking for the gijit repl;
		// deactivate this check entirely for now.

		check.files = append(check.files, file)
		/*
			pp("file.Name.Name='%v'", file.Name.Name)
			pp("pkg.name = '%v'", pkg.name)
			switch name := file.Name.Name; pkg.name {
			case "":
				if name != "_" {
					pkg.name = name
				} else {
					check.errorf(file.Name.Pos(), "invalid package name _")
				}
				fallthrough

				// jea add
			case "main":
				if name == "" {
					// jea: our most common case. name == "" and pkg.name = "main"
					check.files = append(check.files, file)
				}
			case name:
				check.files = append(check.files, file)

			default:
				check.errorf(file.Package, "package %s; expected %s", name, pkg.name)
				// ignore this file
			}
		*/
	}
}

// A bailout panic is used for early termination.
type bailout struct{}

func (check *Checker) handleBailout(err *error) {
	switch p := recover().(type) {
	case nil, bailout:
		// normal return or early exit
		*err = check.firstErr
	default:
		// re-panic
		panic(p)
	}
}

// Files checks the provided files as part of the checker's package.
func (check *Checker) Files(files []*ast.File, depth int) error {
	return check.checkFiles(files, depth)
}

func (check *Checker) checkFiles(files []*ast.File, depth int) (err error) {
	defer check.handleBailout(&err)

	check.initFiles(files)
	//pp("past check.initFiles")
	check.collectObjects(depth + 1)
	//pp("past check.collectObjects")

	check.packageObjects(check.resolveOrder())
	//pp("past check.packageObjects")
	check.functionBodies()
	//pp("past check.functionBodies")
	// jea:
	check.initOrder()

	if !check.conf.DisableUnusedImportCheck {
		check.unusedImports()
	}

	// perform delayed checks
	for _, f := range check.delayed {
		f()
	}
	//pp("past delayed checks")
	check.recordUntyped()
	check.pkg.complete = true
	//pp("past recordUntypes; complete = true, err = '%v'", err)
	return
}

func (check *Checker) recordUntyped() {
	if !debug && check.Types == nil {
		return // nothing to do
	}

	for x, info := range check.untyped {
		if debug && isTyped(info.typ) {
			check.dump("%v: %v (type %v) is typed", x.Pos(), x, info.typ)
			unreachable()
		}
		check.recordTypeAndValue(x, info.mode, info.typ, info.val)
	}
}

func (check *Checker) recordTypeAndValue(x ast.Expr, mode operandMode, typ Type, val constant.Value) {
	pp("check.recordTypeAndValue recording x='%s', typ='%s'", x, typ)
	assert(x != nil)
	assert(typ != nil)
	if mode == invalid {
		return // omit
	}
	assert(typ != nil)
	if mode == constant_ {
		assert(val != nil)
		assert(typ == Typ[Invalid] || isConstType(typ))
	}
	if m := check.Types; m != nil {
		m[x] = TypeAndValue{mode, typ, val}
	}
}

func (check *Checker) recordBuiltinType(f ast.Expr, sig *Signature) {
	// f must be a (possibly parenthesized) identifier denoting a built-in
	// (built-ins in package unsafe always produce a constant result and
	// we don't record their signatures, so we don't see qualified idents
	// here): record the signature for f and possible children.
	for {
		check.recordTypeAndValue(f, builtin, sig, nil)
		switch p := f.(type) {
		case *ast.Ident:
			return // we're done
		case *ast.ParenExpr:
			f = p.X
		default:
			unreachable()
		}
	}
}

func (check *Checker) recordCommaOkTypes(x ast.Expr, a [2]Type) {
	assert(x != nil)
	if a[0] == nil || a[1] == nil {
		return
	}
	assert(isTyped(a[0]) && isTyped(a[1]) && isBoolean(a[1]))
	if m := check.Types; m != nil {
		for {
			tv := m[x]
			assert(tv.Type != nil) // should have been recorded already
			pos := x.Pos()
			tv.Type = NewTuple(
				NewVar(pos, check.pkg, "", a[0]),
				NewVar(pos, check.pkg, "", a[1]),
			)
			m[x] = tv
			// if x is a parenthesized expression (p.X), update p.X
			p, _ := x.(*ast.ParenExpr)
			if p == nil {
				break
			}
			x = p.X
		}
	}
}

func (check *Checker) recordDefAtScope(id *ast.Ident, obj Object, scope *Scope, node ast.Node) {
	assert(id != nil)
	check.recordDef(id, obj)
	pp("adding NewCode for id='%#v', obj.Name()='%v'", id, obj.Name())
	check.NewCode = append(check.NewCode,
		&NewStuff{
			Obj:        obj,
			Scope:      scope,
			Node:       node,
			IsPkgScope: scope == scope.Topmost(),
		})
}

func (check *Checker) recordDef(id *ast.Ident, obj Object) {
	if obj == nil {
		return
	}
	if check.conf.FullPackage {
		if m := check.Defs; m != nil {
			m[id] = obj
		}
		return
	}

	// obj='&types.Func
	// vs
	// obj='&types.TypeName{
	//pp("check.recordDef for id='%s', obj='%#v'/'%s'. obj.Type()='%#v'", id.Name, obj, obj, obj.Type())
	_, objIsTypeName := obj.(*TypeName)
	//_, objIsFunc := obj.(*Func)

	assert(id != nil)
	//check.scope.Dump()

	// are we replacing an earlier definition?

	// CAREFUL. We have to allow an interface and method names
	// to be the same. The runtime package defines an Error
	// interface, and methods called Error() on a type.
	//
	if check.scope != nil && obj != nil {
		oname := obj.Name()
		prior := check.scope.Lookup(oname)
		if prior != nil {
			//_, priorIsTypeName := obj.(*TypeName)
			//_, priorIsFunc := obj.(*Func)
			//pp("prior found for id='%s', prior='%#v'/ prior.Type()='%#v'\n", id.Name, prior, prior.Type())

			// jea: for the REPL, if this is a type,
			// we will need to delete the old version of the type
			// from the ObjMap, so it doesn't grab new
			// methods that are added.
			//
			switch prior.(type) {
			//case *Var:
			case *TypeName:
				// obj and prior must *both be type names* for us to do this delete.
				// Otherwise we do a wrongful delete when a method name
				// clashes with an interface name.
				if objIsTypeName {
					//pp("deleting prior type!!?! : '%s'", oname)
					check.deleteFromObjMapPriorTypeName(oname)
				}
			}
		} else {
			//pp("prior was nil, for obj.Name()='%s'!, here is stack:\n%s\n", obj.Name(), string(runtimedebug.Stack()))
			//check.scope.DeleteByName(obj.Name())
		}
	}

	if m := check.Defs; m != nil {
		m[id] = obj
	}
}

func (check *Checker) deleteFromObjMapPriorTypeName(name string) {
	m := check.ObjMap
	for k := range m {
		if k.Name() == name {
			delete(m, k)
			return
		}
	}
}

func (check *Checker) recordUse(id *ast.Ident, obj Object) {
	assert(id != nil)
	assert(obj != nil)
	if m := check.Uses; m != nil {
		m[id] = obj
	}
}

func (check *Checker) recordImplicit(node ast.Node, obj Object) {
	assert(node != nil)
	assert(obj != nil)
	if m := check.Implicits; m != nil {
		//fmt.Printf("\n jea debug check.go:367 recordImplicit() writing m[node='%#v'] with obj='%#v'\n", node, obj)
		m[node] = obj
	} else {
		//fmt.Printf("\n jea debug check.go:370 recordImplicit() failing to record anything, since check.Implicits is nil")
	}
}

func (check *Checker) recordSelection(x *ast.SelectorExpr, kind SelectionKind, recv Type, obj Object, index []int, indirect bool) {
	assert(obj != nil && (recv == nil || len(index) > 0))
	check.recordUse(x.Sel, obj)
	if m := check.Selections; m != nil {
		m[x] = &Selection{kind, recv, obj, index, indirect}
	}
}

func (check *Checker) recordScope(node ast.Node, scope *Scope) {
	assert(node != nil)
	assert(scope != nil)
	m := check.Scopes
	if m == nil {
		return
	}
	m[node] = scope
}
