package utils

import "strings"

func CleanPathSuffix(path string, suffix string) string {

	//	 如果是空字符串直接返回
	if path == "" {
		return path
	}

	//	 检查结尾是否符合剔除要求
	if strings.HasSuffix(path, suffix) {
		return path[:len(path)-len(suffix)]
	}
	return path
}
