// Package complete 提供 cobra 命令的通用自动补全辅助函数
package complete

import (
	"strings"

	"github.com/spf13/cobra"
)

// Bool 返回 true/false 补全
func Bool(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return filterByPrefix([]string{"true", "false"}, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// Dir 补全目录路径
func Dir(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveFilterDirs
}

// File 补全文件路径（任意扩展名）
func File(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveFilterFileExt
}

// NoFile 禁用文件补全，通常配合自定义值列表使用
func NoFile() cobra.ShellCompDirective {
	return cobra.ShellCompDirectiveNoFileComp
}

// Values 根据给定的候选项列表和用户已输入前缀，返回匹配的补全建议
func Values(suggestions []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return filterByPrefix(suggestions, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// SubcommandNames 从父命令动态获取所有子命令名（适用于父命令的 ValidArgsFunction）
func SubcommandNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var names []string
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}
	return filterByPrefix(names, toComplete), cobra.ShellCompDirectiveNoFileComp
}

// filterByPrefix 根据前缀过滤候选项
func filterByPrefix(suggestions []string, prefix string) []string {
	if prefix == "" {
		return suggestions
	}
	var result []string
	for _, s := range suggestions {
		if strings.HasPrefix(s, prefix) {
			result = append(result, s)
		}
	}
	return result
}
