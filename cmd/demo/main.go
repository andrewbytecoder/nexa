package main

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
)

func toCamelCase(name string) string {
	camelCase := strcase.ToLowerCamel(name)
	if name == strings.ToUpper(name) {
		camelCase = strcase.ToLowerCamel(strings.ToLower(name))
	}

	return camelCase
}

func main() {

	camelStr := toCamelCase("httpPort")
	fmt.Println(camelStr)
	envVarStyle := strcase.ToScreamingSnake(camelStr)
	fmt.Println(envVarStyle) // 输出: HTTP_PORT

	// 另一个例子
	anotherStr := toCamelCase("MY_APP_DATABASE_URL")
	fmt.Println(anotherStr)
	fmt.Println(strcase.ToScreamingSnake(anotherStr)) // 输出: MY_APP_DATABASE_URL

}
