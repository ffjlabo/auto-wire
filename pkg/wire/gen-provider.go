package wire

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/ffjlabo/auto-wire/pkg/ast"
	"github.com/ffjlabo/auto-wire/pkg/util"
)

func GenerateProviderContent(pkgDir string, providerName string, bindMap map[string]*ast.InterfaceSpec) ([]byte, error) {
	importList := []string{}
	providerList := []string{}
	bMap := map[string]*ast.InterfaceSpec{}

	filePath := pkgDir + "/" + "provider.go"
	if _, err := os.Stat(filePath); err == nil {
		// ファイルがすでに存在してたら
		importList, err = ast.FindImportPath(filePath)
		if err != nil {
			return nil, err
		}

		// "github.com/google/wire"を無視
		importList = importList[1:]

		providerList, err = ast.FindProviderName(filePath)
		if err != nil {
			return nil, err
		}

		bMap, err = ast.FindWireBind(filePath)
		if err != nil {
			return nil, err
		}
	}

	// ファイルが存在しない時
	if !util.IsContained(providerList, providerName) {
		providerList = append(providerList, providerName)
	}

	// bindMapを再生性
	for structName, interfaceSpec := range bindMap {
		bMap[structName] = interfaceSpec
	}

	// bMapからimportListにパスを追加
	for _, interfaceSpec := range bMap {
		importPath := interfaceSpec.ImportPath
		if !util.IsContained(importList, importPath) {
			importList = append(importList, importPath)
		}
	}

	// bMapをwire.Bind(new(interface), new(*struct)) の形式にする
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
