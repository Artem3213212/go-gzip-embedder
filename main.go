package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"mime"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/dave/jennifer/jen"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var srcPath = flag.String("src", ".", "Folder with sources to embed")
var dstPath = flag.String("dst", "web_data/handler.go", "Result file path")
var pkgName = flag.String("pkg-name", "web_data", "Name of the generating package")
var rootRoute = flag.String("root-route", "index.html", "Name of file used route for / request")

var (
	stringsContains = Qual("strings", "Contains")
)

var globalConsts = make(map[string]*Statement)

func makeGlobalBinConst(name string, val []byte) {
	globalConsts[name] = Index().Id("byte").ValuesFunc(func(g *Group) {
		for i, b := range val {
			if i%16 == 0 {
				g.Line().Lit(int(b))
			} else {
				g.Lit(int(b))
			}
		}
		g.Line()
	})
}

func makeGlobalGzippedBinConst(name string, val []byte) {
	buf := new(bytes.Buffer)
	gzipWriter, err := gzip.NewWriterLevel(buf, gzip.BestCompression)
	if err != nil {
		panic(err)
	}
	_, err = gzipWriter.Write(val)
	if err != nil {
		_ = gzipWriter.Close()
		panic(err)
	}
	err = gzipWriter.Close()
	if err != nil {
		panic(err)
	}
	makeGlobalBinConst(name, buf.Bytes())
}

var identifierFromFileNameReHelper = regexp.MustCompile(`[^A-Za-z0-9]+`)

func IdentifierFromFileName(name string) string {
	spaced := identifierFromFileNameReHelper.ReplaceAllString(name, " ")
	titleCase := cases.Title(language.Und, cases.NoLower).String(spaced)
	pascalCase := strings.ReplaceAll(titleCase, " ", "")

	if pascalCase == "" {
		return ""
	}

	return strings.ToLower(string(pascalCase[0])) + pascalCase[1:]
}

func genHandlerCall(g *Group, fileName string) {
	constId := IdentifierFromFileName(fileName)

	data, err := os.ReadFile(path.Join(*srcPath, fileName))
	if err != nil {
		panic(err)
	}
	makeGlobalGzippedBinConst(constId, data)

	httpPath := "/" + fileName
	var curr_case *Statement
	if httpPath == *rootRoute {
		curr_case = g.Case(Lit("/"), Lit(httpPath))
	} else {
		curr_case = g.Case(Lit(httpPath))
	}

	curr_case.Block(
		Id("gzipHandler").Call(Id(constId), Lit(mime.TypeByExtension(path.Ext(fileName))), Id("w"), Id("r")),
	)
}

func genRootHandler(f *File) {
	f.Func().Id("Handler").Params(
		Id("w").Qual("net/http", "ResponseWriter"),
		Id("r").Op("*").Qual("net/http", "Request"),
	).Block(
		Switch(Id("r").Dot("URL").Dot("Path")).BlockFunc(func(g *Group) {
			err := filepath.Walk(*srcPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					panic(err)
				}
				relPath, err := filepath.Rel(*srcPath, path)
				if err != nil {
					panic(err)
				}
				if !info.IsDir() {
					genHandlerCall(g, relPath)
				}
				return nil
			})
			if err != nil {
				panic(err)
			}
			g.Default().Block(
				Id("w").Dot("WriteHeader").Call(Lit(404)),
			)
		}),
	)
}

func genGzipHandler(f *File) {
	gzipLit := Lit("gzip")
	acceptEncodingLit := Lit("Accept-Encoding")
	contentEncodingLit := Lit("Content-Encoding")
	gzipReader := Id("gzipReader")

	f.Func().Id("gzipHandler").Params(
		Id("data").Index().Byte(),
		Id("mimeType").String(),
		Id("w").Qual("net/http", "ResponseWriter"),
		Id("r").Op("*").Qual("net/http", "Request"),
	).Block(
		Id("w").Dot("Header").Call().Dot("Set").Call(Lit("Content-Type"), Id("mimeType")),
		Id("w").Dot("Header").Call().Dot("Set").Call(Lit("Cache-Control"), Lit("private, max-age=0")),
		Id("w").Dot("Header").Call().Dot("Set").Call(Lit("Expires"), Lit("-1")),
		Line(),

		If(stringsContains.Call(Id("r").Dot("Header").Dot("Get").Call(acceptEncodingLit), gzipLit)).Block(
			Comment("Give gzip-ed data"),
			Id("w").Dot("Header").Call().Dot("Set").Call(contentEncodingLit, gzipLit),
			Id("w").Dot("WriteHeader").Call(Lit(200)),
			Id("w").Dot("Write").Call(Id("data")),
			Return(),
		),
		Line(),

		Comment("Uncompress and send data"),
		List(gzipReader.Clone(), Err()).Op(":=").Qual("compress/gzip", "NewReader").Call(Qual("bytes", "NewBuffer").Call(Id("data"))),
		If(Err().Op("!=").Nil()).Block(
			Id("w").Dot("WriteHeader").Call(Lit(500)),
			Return(),
		),
		Defer().Add(gzipReader.Clone().Dot("Close").Call()),
		Line(),

		Id("w").Dot("WriteHeader").Call(Lit(200)),
		Qual("io", "Copy").Call(Id("w"), gzipReader.Clone()),
	)
}

func main() {
	flag.Parse()

	f := NewFile(*pkgName)
	c := f.Var()
	genGzipHandler(f)
	f.Line()
	genRootHandler(f)

	c.DefsFunc(func(g *Group) {
		for name, val := range globalConsts {
			g.Id(name).Op("=").Add(val)
		}
	})

	err := os.MkdirAll(filepath.Dir(*dstPath), 0755)
	if err != nil {
		panic(err)
	}
	file, err := os.Create(*dstPath)
	if err != nil {
		panic(err)
	}
	err = f.Render(file)
	if err != nil {
		panic(err)
	}
}
