package util

import (
	"go/ast"
	"go/token"
	"io/ioutil"
	"sync"
)

var loadedFiles sync.Map

// ReadCode read source code from ast.Node
func ReadCode(f *token.FileSet, node ast.Node) string {
	ps := f.Position(node.Pos())
	pe := f.Position(node.End())
	file, ok := loadedFiles.Load(ps.Filename)
	if !ok {
		file, _ = ioutil.ReadFile(ps.Filename)
		loadedFiles.Store(ps.Filename, file)
	}
	return string(file.([]byte)[ps.Offset:pe.Offset])
}
