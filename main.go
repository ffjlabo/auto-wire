package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/ffjlabo/auto-wire/config"
)

func FindProviderName(filePath string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.Mode(0))
	if err != nil {
		return nil, err
	}

	var genDecl *ast.GenDecl
	for _, decl := range f.Decls {
		d, ok := decl.(*ast.GenDecl)
		if ok && d.Tok == token.VAR {
			genDecl = d
		}
	}

	if genDecl == nil {
		return nil, err
	}

	var valueSpec *ast.ValueSpec
	for _, spec := range genDecl.Specs {
		s, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for _, name := range s.Names {
			if name.Name != "Set" {
				continue
			}
		}

		valueSpec = s
	}

	if valueSpec == nil {
		return nil, err
	}

	var callExpr *ast.CallExpr
	for _, value := range valueSpec.Values {
		expr, ok := value.(*ast.CallExpr)
		if !ok {
			continue
		}

		fun, ok := expr.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		xIdent, ok := fun.X.(*ast.Ident)
		if !ok {
			continue
		}

		if xIdent.Name != "wire" || fun.Sel.Name != "NewSet" {
			continue
		}

		callExpr = expr
	}

	if callExpr == nil {
		return nil, err
	}

	providerNameList := []string{}
	for _, arg := range callExpr.Args {
		ident, ok := arg.(*ast.Ident)
		if !ok {
			continue
		}
		providerNameList = append(providerNameList, ident.Name)
	}

	return providerNameList, nil
}

func FindImportPath(filePath string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.Mode(0))
	if err != nil {
		return nil, err
	}

	importList := []string{}
	for _, importSpec := range f.Imports {
		importPath := strings.Replace(importSpec.Path.Value, "\"", "", -1)
		importList = append(importList, importPath)
	}

	return importList, nil
}

func isContained(arr []string, val string) bool {
	for _, str := range arr {
		if str == val {
			return true
		}
	}

	return false
}

func GenerateWireContent(importPath string) ([]byte, error) {
	importList := []string{}
	providerSetList := []string{}

	filePath := config.DiDir + "/wire.go"
	if _, err := os.Stat(filePath); err == nil {
		// ファイルがすでに存在してたら
		importList, err = FindImportPath(filePath)
		if err != nil {
			return nil, err
		}

		// "github.com/google/wire"を無視
		importList = importList[1:]
	}

	// importPathの重複確認
	if !isContained(importList, importPath) {
		importList = append(importList, importPath)
	}

	// importPathからproviderSetListを作成
	for _, path := range importList {
		paths := strings.Split(path, "/")
		pkgName := paths[len(paths)-1]

		providerSetList = append(providerSetList, pkgName+"."+"Set")
	}

	tpl, err := template.ParseFiles("./template/wire.go.tmpl")
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"importList":  importList,
		"wireSetList": providerSetList,
	}

	var buf bytes.Buffer

	err = tpl.Execute(&buf, data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type InterfaceSpec struct {
	Name       string
	ImportPath string
}

func FindWireBind(filePath string) (map[string]*InterfaceSpec, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.Mode(0))
	if err != nil {
		return nil, err
	}

	var genDecl *ast.GenDecl
	for _, decl := range f.Decls {
		d, ok := decl.(*ast.GenDecl)
		if ok && d.Tok == token.VAR {
			genDecl = d
		}
	}

	if genDecl == nil {
		return nil, err
	}

	var valueSpec *ast.ValueSpec
	for _, spec := range genDecl.Specs {
		s, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for _, name := range s.Names {
			if name.Name != "Set" {
				continue
			}
		}

		valueSpec = s
	}

	if valueSpec == nil {
		return nil, err
	}

	var callExpr *ast.CallExpr
	for _, value := range valueSpec.Values {
		expr, ok := value.(*ast.CallExpr)
		if !ok {
			continue
		}

		fun, ok := expr.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		xIdent, ok := fun.X.(*ast.Ident)
		if !ok {
			continue
		}

		if xIdent.Name != "wire" || fun.Sel.Name != "NewSet" {
			continue
		}

		callExpr = expr
	}

	if callExpr == nil {
		return nil, err
	}

	// wire.Bindとなるast.SelectorExprを取得
	var bindCallExpr []*ast.CallExpr
	for _, arg := range callExpr.Args {
		cExpr, ok := arg.(*ast.CallExpr)
		if !ok {
			continue
		}

		bindCallExpr = append(bindCallExpr, cExpr)
	}

	// argsを取得 1つめがstruct、2つ目がinterface用
	importMap := map[string]string{}
	for _, importSpec := range f.Imports {
		importPath := strings.Replace(importSpec.Path.Value, "\"", "", -1)

		paths := strings.Split(importPath, "/")
		importName := paths[len(paths)-1]

		importMap[importName] = importPath
	}

	bindMap := map[string]*InterfaceSpec{}
	for _, cExpr := range bindCallExpr {
		// interface
		interfaceCallExpr, _ := cExpr.Args[0].(*ast.CallExpr)
		interfaceSelectorExpr, _ := interfaceCallExpr.Args[0].(*ast.SelectorExpr)
		importNameIdent, _ := interfaceSelectorExpr.X.(*ast.Ident)

		importName := importNameIdent.Name
		interfaceName := interfaceSelectorExpr.Sel.Name

		interfaceSpec := InterfaceSpec{
			Name:       interfaceName,
			ImportPath: importMap[importName],
		}

		// struct
		structCallExpr, _ := cExpr.Args[1].(*ast.CallExpr)
		starExpr, _ := structCallExpr.Args[0].(*ast.StarExpr)

		structIdent, _ := starExpr.X.(*ast.Ident)
		structName := structIdent.Name

		bindMap[structName] = &interfaceSpec
	}

	return bindMap, nil
}

func GenerateProviderContent(pkgDir string, providerName string, bindMap map[string]*InterfaceSpec) ([]byte, error) {
	importList := []string{}   // bindMapと今までのimportからできる
	providerList := []string{} // 今までのproviderとproviderName
	bMap := map[string]*InterfaceSpec{}

	filePath := pkgDir + "/" + "provider.go"
	if _, err := os.Stat(filePath); err == nil {
		// ファイルがすでに存在してたら
		importList, err = FindImportPath(filePath)
		if err != nil {
			return nil, err
		}

		// "github.com/google/wire"を無視
		importList = importList[1:]

		providerList, err = FindProviderName(filePath)
		if err != nil {
			return nil, err
		}

		bMap, err = FindWireBind(filePath)
		if err != nil {
			return nil, err
		}
	}

	// ファイルが存在しない時
	if !isContained(providerList, providerName) {
		providerList = append(providerList, providerName)
	}

	// bindMapを再生性
	for structName, interfaceSpec := range bindMap {
		fmt.Println(interfaceSpec)
		bMap[structName] = &InterfaceSpec{}
		bMap[structName] = interfaceSpec
	}

	// bMapからimportListにパスを追加
	for _, interfaceSpec := range bMap {
		importPath := interfaceSpec.ImportPath
		if !isContained(importList, importPath) {
			importList = append(importList, importPath)
		}
	}

	// bMapをwire.Bind(new(), new()) の形式にする
	bindList := []string{}
	for structName, interfaceSpec := range bMap {
		paths := strings.Split(interfaceSpec.ImportPath, "/")
		importName := paths[len(paths)-1]

		interfaceName := importName + "." + interfaceSpec.Name
		bind := fmt.Sprintf("wire.Bind(new(%s), new(*%s))", interfaceName, structName)
		bindList = append(bindList, bind)
	}

	tpl, err := template.ParseFiles("./template/provider.go.tmpl")
	if err != nil {
		return nil, err
	}

	paths := strings.Split(pkgDir, "/")
	pkgName := paths[len(paths)-1]
	data := map[string]interface{}{
		"pkgName":      pkgName,
		"importList":   importList,
		"providerList": providerList,
		"bindList":     bindList,
	}

	var buf bytes.Buffer

	err = tpl.Execute(&buf, data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func main() {
	// wire.go
	importPath := config.ModuleName + "/" + config.UsecaseDir
	content, err := GenerateWireContent(importPath)
	if err != nil {
		log.Fatal(err)
	}

	filePath := config.DiDir + "/wire.go"
	err = ioutil.WriteFile(filePath, content, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// provider.go作成例 1
	providerName := "NewUserRepository"
	bindMap := map[string]*InterfaceSpec{
		"User": &InterfaceSpec{
			Name:       "UserRepository",
			ImportPath: config.ModuleName + "/" + config.DomainRepo,
		},
	}

	content, err = GenerateProviderContent(config.InfraRepo, providerName, bindMap)
	if err != nil {
		log.Fatal(err)
	}

	filePath = config.InfraRepo + "/provider.go"
	err = ioutil.WriteFile(filePath, content, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// provider.go作成例 2
	providerName = "NewSpotifyRepository"

	content, err = GenerateProviderContent(config.UsecaseDir, providerName, nil)
	if err != nil {
		log.Fatal(err)
	}

	filePath = config.UsecaseDir + "/provider.go"
	err = ioutil.WriteFile(filePath, content, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
