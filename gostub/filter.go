// This file is derived from http://golang.org/src/pkg/go/ast/filter.go
// that came with the following notice: http://golang.org/LICENSE.

package main

import . "go/ast"

func myFilterIdentList(list []*Ident, f Filter) []*Ident {
	j := 0
	for _, x := range list {
		if f(x.Name) {
			list[j] = x
			j++
		}
	}
	return list[0:j]
}

// myFieldName assumes that x is the type of an anonymous field and
// returns the corresponding field name. If x is not an acceptable
// anonymous field, the result is nil.
//
func myFieldName(x Expr) *Ident {
	switch t := x.(type) {
	case *Ident:
		return t
	case *SelectorExpr:
		if _, ok := t.X.(*Ident); ok {
			return t.Sel
		}
	case *StarExpr:
		return myFieldName(t.X)
	}
	return nil
}

func myFilterFieldList(fields *FieldList, filter Filter, export bool) (removedFields bool) {
	if fields == nil {
		return false
	}
	list := fields.List
	j := 0
	for _, f := range list {
		keepField := false
		if len(f.Names) == 0 {
			// anonymous field
			name := myFieldName(f.Type)
			keepField = name != nil && filter(name.Name)
		} else {
			n := len(f.Names)
			f.Names = myFilterIdentList(f.Names, filter)
			if len(f.Names) < n {
				removedFields = true
			}
			keepField = len(f.Names) > 0
		}
		if keepField {
			if export {
				myFilterType(f.Type, filter, export)
			}
			list[j] = f
			j++
		}
	}
	if j < len(list) {
		removedFields = true
	}
	fields.List = list[0:j]
	return
}

func myFilterParamList(fields *FieldList, filter Filter, export bool) bool {
	if fields == nil {
		return false
	}
	var b bool
	for _, f := range fields.List {
		if myFilterType(f.Type, filter, export) {
			b = true
		}
	}
	return b
}

func myFilterType(typ Expr, f Filter, export bool) bool {
	switch t := typ.(type) {
	case *Ident:
		return f(t.Name)
	case *ParenExpr:
		return myFilterType(t.X, f, export)
	case *ArrayType:
		return myFilterType(t.Elt, f, export)
	case *StructType:
		if myFilterFieldList(t.Fields, f, export) {
			t.Incomplete = true
		}
		return len(t.Fields.List) > 0
	case *FuncType:
		b1 := myFilterParamList(t.Params, f, export)
		b2 := myFilterParamList(t.Results, f, export)
		return b1 || b2
	case *InterfaceType:
		if myFilterFieldList(t.Methods, f, export) {
			t.Incomplete = true
		}
		return len(t.Methods.List) > 0
	case *MapType:
		b1 := myFilterType(t.Key, f, export)
		b2 := myFilterType(t.Value, f, export)
		return b1 || b2
	case *ChanType:
		return myFilterType(t.Value, f, export)
	}
	return false
}

func myFilterSpec(spec Spec, f Filter, export bool) bool {
	switch s := spec.(type) {
	case *ValueSpec:
		s.Names = myFilterIdentList(s.Names, f)
		if len(s.Names) > 0 {
			if export {
				myFilterType(s.Type, f, export)
			}
			return true
		}
	case *TypeSpec:
		if f(s.Name.Name) {
			if export {
				myFilterType(s.Type, f, export)
			}
			return true
		}
		if !export {
			// For general filtering (not just exports),
			// filter type even if name is not filtered
			// out.
			// If the type contains filtered elements,
			// keep the declaration.
			return myFilterType(s.Type, f, export)
		}
	}
	return false
}

func myFilterSpecList(list []Spec, f Filter, export bool) []Spec {
	j := 0
	for _, s := range list {
		if myFilterSpec(s, f, export) {
			list[j] = s
			j++
		}
	}
	return list[0:j]
}

func myFilterDecl(decl Decl, f Filter, export bool) bool {
	switch d := decl.(type) {
	case *GenDecl:
		d.Specs = myFilterSpecList(d.Specs, f, export)
		return len(d.Specs) > 0
	case *FuncDecl:
		return f(d.Name.Name)
	}
	return false
}

// myExportFilter is a special filter function to extract exported nodes.
func myExportFilter(name string) bool {
	return IsExported(name)
}

func myFilterFile(src *File, f Filter, export bool) bool {
	j := 0
	for _, d := range src.Decls {
		if myFilterDecl(d, f, export) {
			src.Decls[j] = d
			j++
		}
	}
	src.Decls = src.Decls[0:j]
	return j > 0
}
