package wire

import (
	"bytes"
	_ "embed"
	"os"
	"strings"
	"text/template"

	"github.com/ffjlabo/auto-wire/pkg/ast"
	"github.com/ffjlabo/auto-wire/pkg/util"
)

//go:embed template/wire.go.tmpl
var wireTemplate string

// GenerateWireContent generate the content of wire.go
func GenerateWireContent(wireDir string, importPath string) ([]byte, error) {
	importList := []string{}
	providerSetList := []string{}

	filePath := wireDir + "/" + "wire.go"
	if _, err := os.Stat(filePath); err == nil {
		// ファイルがすでに存在してたら
		importList, err = ast.FindImportPath(filePath)
		if err != nil {
			return nil, err
		}

		// "github.com/google/wire"を無視
		importList = importList[1:]
	}

	// importPathの重複確認
	if !util.IsContained(importList, importPath) {
		importList = append(importList, importPath)
	}

	// importPathからproviderSetListを作成
	for _, path := range importList {
		paths := strings.Split(path, "/")
		pkgName := paths[len(paths)-1]

		providerSetList = append(providerSetList, pkgName+"."+"Set")
	}

	tpl, err := template.New("wireTemplate").Parse(wireTemplate)
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
