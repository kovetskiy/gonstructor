package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/moznion/gonstructor/internal"
	"github.com/moznion/gonstructor/internal/constructor"
	g "github.com/moznion/gowrtr/generator"
)

const (
	allArgsConstructorType = "allArgs"
	builderConstructorType = "builder"
)

var (
	typeName          = flag.String("type", "", "[mandatory] a type name")
	output            = flag.String("output", "", `[optional] output file name (default "srcdir/<type>_gen.go")`)
	constructorTypes  = flag.String("constructorTypes", allArgsConstructorType, fmt.Sprintf(`[optional] comma-separated list of constructor types; it expects "%s" and "%s"`, allArgsConstructorType, builderConstructorType))
	shouldShowVersion = flag.Bool("version", false, "[optional] show the version information")
)

func main() {
	flag.Parse()

	if *shouldShowVersion {
		internal.ShowVersion()
		return
	}

	if *typeName == "" {
		flag.Usage()
		os.Exit(2)
	}

	constructorTypes, err := getConstructorTypes()
	if err != nil {
		log.Printf("[error] %s", err)
		flag.Usage()
		os.Exit(2)
	}

	args := flag.Args()
	if len(args) <= 0 {
		args = []string{"."}
	}

	pkg, err := internal.ParsePackage(args)
	if err != nil {
		log.Fatal(fmt.Errorf("[error] failed to parse a package: %w", err))
	}

	astFiles, err := internal.ParseFiles(pkg.GoFiles)
	if err != nil {
		log.Fatal(fmt.Errorf("[error] failed to parse a file: %w", err))
	}

	fields, err := constructor.CollectConstructorFieldsFromAST(*typeName, astFiles)
	if err != nil {
		log.Fatal(fmt.Errorf("[error] failed to collect fields from files: %w", err))
	}

	rootStmt := g.NewRoot(
		g.NewComment(fmt.Sprintf(" Code generated by gonstructor %s; DO NOT EDIT.", strings.Join(os.Args[1:], " "))),
		g.NewNewline(),
		g.NewPackage(pkg.Name),
		g.NewNewline(),
	)

	for _, constructorType := range constructorTypes {
		var constructorGenerator constructor.Generator
		switch constructorType {
		case allArgsConstructorType:
			constructorGenerator = &constructor.AllArgsConstructorGenerator{
				TypeName: *typeName,
				Fields:   fields,
			}
		case builderConstructorType:
			constructorGenerator = &constructor.BuilderGenerator{
				TypeName: *typeName,
				Fields:   fields,
			}
		default:
			// unreachable, just in case
			log.Fatalf("[error] unexpected constructor type has come [given=%s]", constructorType)
		}
		rootStmt = rootStmt.AddStatements(constructorGenerator.Generate())
	}

	code, err := rootStmt.EnableGoimports().EnableSyntaxChecking().Generate(0)
	if err != nil {
		log.Fatal(fmt.Errorf("[error] failed to generate code: %w", err))
	}

	err = ioutil.WriteFile(getFilenameToGenerate(args), []byte(code), 0644)
	if err != nil {
		log.Fatal(fmt.Errorf("[error] failed output generated code to a file: %w", err))
	}
}

func getConstructorTypes() ([]string, error) {
	typs := strings.Split(*constructorTypes, ",")
	for _, typ := range typs {
		if typ != allArgsConstructorType && typ != builderConstructorType {
			return nil, fmt.Errorf("unexpected constructor type has come [given=%s]", typ)
		}
	}
	return typs, nil
}

func isDirectory(name string) bool {
	info, err := os.Stat(name)
	if err != nil {
		log.Fatal(err)
	}
	return info.IsDir()
}

func getFilenameToGenerate(args []string) string {
	if *output != "" {
		return *output
	}

	var dir string
	if len(args) == 1 && isDirectory(args[0]) {
		dir = args[0]
	} else {
		dir = filepath.Dir(args[0])
	}
	return fmt.Sprintf("%s/%s_gen.go", dir, strcase.ToSnake(*typeName))
}
