package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tght/lan-proxy-gateway/internal/platform"
	"github.com/tght/lan-proxy-gateway/internal/ui"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "卸载 LAN Proxy Gateway",
	Long: `卸载 LAN Proxy Gateway，包括：
- 停止正在运行的网关
- 移除二进制文件
- 可选：删除配置文件和数据

用法:
  gateway uninstall          # 交互式卸载
  gateway uninstall --all    # 删除所有文件（包括配置和数据）
  gateway uninstall --keep   # 保留配置文件和数据`,
	Run: runUninstall,
}

var (
	uninstallAll  bool
	uninstallKeep bool
)

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallAll, "all", false, "删除所有文件（包括配置和数据）")
	uninstallCmd.Flags().BoolVar(&uninstallKeep, "keep", false, "保留配置文件和数据")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) {
	ui.ShowLogo()
	color.New(color.Bold).Println("LAN Proxy Gateway 卸载向导")
	ui.Separator()

	reader := bufio.NewReader(os.Stdin)

	// Step 1: Check if running
	ui.Step(1, 4, "检查网关状态...")
	p := platform.New()
	if p.IsRunning() {
		ui.Warn("网关正在运行，正在停止...")
		runStop(cmd, args)
	}

	// Step 2: Get installation path
	ui.Step(2, 4, "查找安装位置...")
	binary, err := p.FindBinary()
	if err != nil {
		ui.Warn("未找到 gateway 二进制文件")
		binary = "/usr/local/bin/gateway" // fallback
	}
	ui.Info("安装位置: %s", binary)

	// Step 3: Ask about config and data
	keepConfig := uninstallKeep
	removeAll := uninstallAll

	if !uninstallAll && !uninstallKeep {
		ui.Step(3, 4, "选择卸载选项...")
		fmt.Println()
		color.New(color.Bold).Println("是否保留配置文件和数据？")
		fmt.Printf("  %s\n", color.New(color.Faint).Sprint("配置文件: gateway.yaml"))
		fmt.Printf("  %s\n", color.New(color.Faint).Sprint("数据目录: data/"))
		fmt.Println()
		fmt.Print("保留配置文件？[Y/n] ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		keepConfig = answer != "n"
	} else {
		ui.Step(3, 4, "处理配置文件...")
	}

	// Step 4: Uninstall service if exists
	ui.Step(4, 4, "清理系统服务...")
	uninstallService()

	// Remove binary
	ui.Info("正在删除二进制文件...")
	if runtime.GOOS == "windows" {
		// On Windows, we can't delete the running binary
		// Move it to temp and schedule deletion
		tmpPath := os.TempDir() + "\\gateway.old"
		os.Rename(binary, tmpPath)
		ui.Success("二进制文件已标记为删除（重启后生效）")
	} else {
		if err := os.Remove(binary); err != nil {
			ui.Warn("删除二进制文件失败: %s", err)
			ui.Info("请手动删除: sudo rm -f %s", binary)
		} else {
			ui.Success("二进制文件已删除")
		}
	}

	// Handle config and data
	if removeAll || !keepConfig {
		ui.Info("正在删除配置文件和数据...")
		configFiles := []string{"gateway.yaml", ".secret"}
		for _, f := range configFiles {
			if _, err := os.Stat(f); err == nil {
				os.Remove(f)
				ui.Success("已删除: %s", f)
			}
		}
		
		// Remove data directory
		if _, err := os.Stat("data"); err == nil {
			os.RemoveAll("data")
			ui.Success("已删除: data/")
		}
	} else {
		ui.Info("配置文件和数据已保留")
		ui.Info("如需删除，请手动执行:")
		ui.Info("  rm -f gateway.yaml .secret")
		ui.Info("  rm -rf data/")
	}

	// Summary
	fmt.Println()
	ui.Separator()
	color.New(color.FgGreen, color.Bold).Println("  卸载完成！")
	ui.Separator()
	fmt.Println()
	
	if keepConfig {
		fmt.Printf("  %s\n", color.New(color.Faint).Sprint("配置文件已保留，下次安装时可继续使用"))
	}
	fmt.Println()
}

func uninstallService() {
	p := platform.New()
	
	switch runtime.GOOS {
	case "darwin":
		// Remove launchd plist
		plistPaths := []string{
			"/Library/LaunchDaemons/com.lan-proxy-gateway.plist",
			os.Getenv("HOME") + "/Library/LaunchAgents/com.lan-proxy-gateway.plist",
		}
		for _, path := range plistPaths {
			if _, err := os.Stat(path); err == nil {
				exec.Command("launchctl", "unload", path).Run()
				os.Remove(path)
				ui.Success("已移除 launchd 服务")
			}
		}
		
	case "linux":
		// Remove systemd service
		servicePath := "/etc/systemd/system/lan-proxy-gateway.service"
		if _, err := os.Stat(servicePath); err == nil {
			exec.Command("systemctl", "stop", "lan-proxy-gateway").Run()
			exec.Command("systemctl", "disable", "lan-proxy-gateway").Run()
			os.Remove(servicePath)
			exec.Command("systemctl", "daemon-reload").Run()
			ui.Success("已移除 systemd 服务")
		}
		
	case "windows":
		// Remove Windows service
		cmd := exec.Command("sc", "query", "LANProxyGateway")
		if err := cmd.Run(); err == nil {
			exec.Command("sc", "stop", "LANProxyGateway").Run()
			exec.Command("sc", "delete", "LANProxyGateway").Run()
			ui.Success("已移除 Windows 服务")
		}
	}
}