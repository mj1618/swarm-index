package parsers

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

func init() {
	Register(&GoParser{})
}

// GoParser extracts symbols from Go source files using the stdlib AST.
type GoParser struct{}

func (g *GoParser) Extensions() []string {
	return []string{".go"}
}

func (g *GoParser) Parse(filePath string, content []byte) ([]Symbol, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	var symbols []Symbol

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			sym := funcSymbol(fset, d)
			symbols = append(symbols, sym)

		case *ast.GenDecl:
			symbols = append(symbols, genDeclSymbols(fset, d)...)
		}
	}

	return symbols, nil
}

func funcSymbol(fset *token.FileSet, d *ast.FuncDecl) Symbol {
	sym := Symbol{
		Name:     d.Name.Name,
		Line:     fset.Position(d.Pos()).Line,
		EndLine:  fset.Position(d.End()).Line,
		Exported: ast.IsExported(d.Name.Name),
	}

	if d.Recv != nil && len(d.Recv.List) > 0 {
		sym.Kind = "method"
		sym.Parent = receiverTypeName(d.Recv.List[0].Type)
	} else {
		sym.Kind = "func"
	}

	sym.Signature = funcSignature(d)
	return sym
}

func genDeclSymbols(fset *token.FileSet, d *ast.GenDecl) []Symbol {
	var symbols []Symbol

	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			sym := Symbol{
				Name:     s.Name.Name,
				Line:     fset.Position(s.Pos()).Line,
				EndLine:  fset.Position(s.End()).Line,
				Exported: ast.IsExported(s.Name.Name),
			}
			switch s.Type.(type) {
			case *ast.StructType:
				sym.Kind = "struct"
			case *ast.InterfaceType:
				sym.Kind = "interface"
			default:
				sym.Kind = "type"
			}
			sym.Signature = "type " + s.Name.Name
			symbols = append(symbols, sym)

		case *ast.ValueSpec:
			kind := "var"
			if d.Tok == token.CONST {
				kind = "const"
			}
			for _, name := range s.Names {
				sym := Symbol{
					Name:     name.Name,
					Kind:     kind,
					Line:     fset.Position(name.Pos()).Line,
					EndLine:  fset.Position(name.End()).Line,
					Exported: ast.IsExported(name.Name),
				}
				sym.Signature = kind + " " + name.Name
				symbols = append(symbols, sym)
			}
		}
	}

	return symbols
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	}
	return ""
}

func funcSignature(d *ast.FuncDecl) string {
	var buf bytes.Buffer
	buf.WriteString("func ")
	if d.Recv != nil && len(d.Recv.List) > 0 {
		buf.WriteString("(")
		buf.WriteString(exprString(d.Recv.List[0].Type))
		buf.WriteString(") ")
	}
	buf.WriteString(d.Name.Name)
	buf.WriteString("(")
	buf.WriteString(fieldListString(d.Type.Params))
	buf.WriteString(")")
	if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
		results := fieldListString(d.Type.Results)
		if len(d.Type.Results.List) > 1 {
			buf.WriteString(" (")
			buf.WriteString(results)
			buf.WriteString(")")
		} else {
			buf.WriteString(" ")
			buf.WriteString(results)
		}
	}
	return buf.String()
}

func fieldListString(fl *ast.FieldList) string {
	if fl == nil {
		return ""
	}
	var parts []string
	for _, f := range fl.List {
		typeStr := exprString(f.Type)
		if len(f.Names) == 0 {
			parts = append(parts, typeStr)
		} else {
			names := make([]string, len(f.Names))
			for i, n := range f.Names {
				names[i] = n.Name
			}
			parts = append(parts, strings.Join(names, ", ")+" "+typeStr)
		}
	}
	return strings.Join(parts, ", ")
}

func exprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprString(t.Elt)
		}
		return "[" + exprString(t.Len) + "]" + exprString(t.Elt)
	case *ast.MapType:
		return "map[" + exprString(t.Key) + "]" + exprString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + exprString(t.Elt)
	case *ast.FuncType:
		var buf bytes.Buffer
		buf.WriteString("func(")
		buf.WriteString(fieldListString(t.Params))
		buf.WriteString(")")
		if t.Results != nil && len(t.Results.List) > 0 {
			results := fieldListString(t.Results)
			if len(t.Results.List) > 1 {
				buf.WriteString(" (")
				buf.WriteString(results)
				buf.WriteString(")")
			} else {
				buf.WriteString(" ")
				buf.WriteString(results)
			}
		}
		return buf.String()
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + exprString(t.Value)
		case ast.RECV:
			return "<-chan " + exprString(t.Value)
		default:
			return "chan " + exprString(t.Value)
		}
	case *ast.BasicLit:
		return t.Value
	case *ast.IndexExpr:
		return exprString(t.X) + "[" + exprString(t.Index) + "]"
	case *ast.IndexListExpr:
		indices := make([]string, len(t.Indices))
		for i, idx := range t.Indices {
			indices[i] = exprString(idx)
		}
		return exprString(t.X) + "[" + strings.Join(indices, ", ") + "]"
	case *ast.ParenExpr:
		return "(" + exprString(t.X) + ")"
	case *ast.StructType:
		return "struct{}"
	}
	return "?"
}
