package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/util"
	"os"
	"strings"
	"time"
)

const banner = `
 /$$    /$$    /$$  /$$$$$$
| $$   | $$   | $$ /$$__  $$
| $$   | $$   | $$| $$  \__/
| $$   |  $$ / $$/|  $$$$$$ 
| $$    \  $$ $$/  \____  $$
| $$     \  $$$/   /$$  \ $$
| $$$$$$$$\  $/   |  $$$$$$/
|________/ \_/     \______/ 

Lightweight Version Suite(Love SHX) %s %s
Powered by jianggujin(www.jianggujin.com)
`

var rootCmd = &cobra.Command{
	Short: "LVS(Lightweight Version Suite)",
	Long:  fmt.Sprintf(banner, config.BuildVersion, config.BuildTime),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd:   true,
		DisableNoDescFlag:   false,
		DisableDescriptions: false,
		HiddenDefaultCmd:    true,
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	timeZone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return
	}
	time.Local = timeZone
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		err = util.Sudo(err)
		var msg string
		if err != nil {
			msg = err.Error()
		}
		// 未找到命令，尝试使用默认命令查找
		if strings.HasPrefix(msg, "unknown command") {
			commandName := config.GetString(config.KeyLvsDefaultCommand)
			if commandName != "" {
				rootCmd.SetArgs(append([]string{commandName}, os.Args[1:]...))
				err = util.Sudo(rootCmd.Execute())
				if err == nil {
					os.Exit(0)
				}
				msg = err.Error()
			}
		}
		fmt.Println(msg)
		os.Exit(1)
	}
}
