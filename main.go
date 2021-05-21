package main

import (
	"io/ioutil"
	"log"

	"github.com/ffjlabo/auto-wire/config"
	"github.com/ffjlabo/auto-wire/pkg/ast"
	"github.com/ffjlabo/auto-wire/pkg/wire"
)

func main() {
	// wire.go
	importPath := config.ModuleName + "/" + config.UsecaseDir
	content, err := wire.GenerateWireContent(config.DiDir, importPath)
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
	bindMap := map[string]*ast.InterfaceSpec{
		"User": &ast.InterfaceSpec{
			Name:       "UserRepository",
			ImportPath: config.ModuleName + "/" + config.DomainRepo,
		},
	}

	content, err = wire.GenerateProviderContent(config.InfraRepo, providerName, bindMap)
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

	content, err = wire.GenerateProviderContent(config.UsecaseDir, providerName, nil)
	if err != nil {
		log.Fatal(err)
	}

	filePath = config.UsecaseDir + "/provider.go"
	err = ioutil.WriteFile(filePath, content, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
