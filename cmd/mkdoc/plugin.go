package main

import (
	_ "github.com/thewinds/mkdoc/generator/docsify"
	_ "github.com/thewinds/mkdoc/generator/insomnia"
	_ "github.com/thewinds/mkdoc/generator/markdown"
	_ "github.com/thewinds/mkdoc/objloader/gapiloader"
	_ "github.com/thewinds/mkdoc/objloader/goloader"
	_ "github.com/thewinds/mkdoc/scanner/docdef"
	_ "github.com/thewinds/mkdoc/scanner/gofunc"
	_ "github.com/thewinds/mkdoc/scanner/gqlboss"
)
