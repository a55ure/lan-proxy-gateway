package proxy

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ExtractProxies reads a Clash/mihomo YAML config file and extracts
// the "proxies:" section into a separate file for use as a proxy-provider.
// Returns the number of proxies extracted.
func ExtractProxies(inputPath, outputPath string) (int, error) {
	in, err := os.Open(inputPath)
	if err != nil {
		return 0, fmt.Errorf("无法打开配置文件: %w", err)
	}
	defer in.Close()

	var lines []string
	found := false
	count := 0
	inProxiesSection := false
	lastLineWasDash := false

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// 检测 proxies: 段落开始
		if strings.HasPrefix(trimmedLine, "proxies:") {
			found = true
			inProxiesSection = true
			lines = append(lines, line)
			continue
		}

		if !found {
			continue
		}

		// 检测下一个顶级 key（排除空行和注释）
		if inProxiesSection && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
			// 检查是否是新的顶级 key（不以空格或 - 开头）
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(trimmedLine, "-") {
				// 遇到下一个顶级 key，停止
				break
			}

			// 检测代理节点 - 支持多种格式
			if strings.HasPrefix(trimmedLine, "-") {
				// 检查是否是单独的 "-"（多行格式的开始）
				if trimmedLine == "-" {
					lastLineWasDash = true
				} else {
					// 格式1: - name: xxx（行内格式）
					if strings.Contains(trimmedLine, "name:") {
						count++
					} else if strings.Contains(trimmedLine, "{") && strings.Contains(trimmedLine, "name:") {
						// 格式2: - {name: xxx, ...}
						count++
					}
					lastLineWasDash = false
				}
			} else if lastLineWasDash && strings.Contains(trimmedLine, "name:") {
				// 格式3: 多行格式，上一行是 "-"，这一行是 "name: xxx"
				count++
				lastLineWasDash = false
			} else if lastLineWasDash && (strings.Contains(trimmedLine, "server:") || strings.Contains(trimmedLine, "type:")) {
				// 格式4: 多行格式，但 name 在 server 或 type 之后
				// 这种情况下需要往前查找 name
				count++
				lastLineWasDash = false
			}
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("读取配置文件失败: %w", err)
	}

	if !found {
		return 0, fmt.Errorf("未能从配置文件中找到 proxies: 段落")
	}

	if count == 0 {
		return 0, fmt.Errorf("未能从配置文件中提取到代理节点，请确认 proxies: 段落包含有效的代理配置")
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return 0, fmt.Errorf("写入代理文件失败: %w", err)
	}

	return count, nil
}