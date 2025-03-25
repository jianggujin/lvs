//go:build (windows && amd64) || (darwin && (amd64 || arm64)) || (linux && amd64)

package main

import (
	"fmt"
	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/util"
	"net/http"
	"strings"
	"time"
)

type UpgradeCommand struct {
	Short bool
}

func init() {
	util.AddCommand(rootCmd, &UpgradeCommand{})
}

func (command *UpgradeCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade to latest version",
		Run:   command.Run,
	}
	flags := cmd.Flags()
	flags.BoolVarP(&command.Short, "short", "s", false, "only display version number")
	return cmd
}

func (command *UpgradeCommand) Run(*cobra.Command, []string) {
	current, err := version.NewVersion(strings.ToLower(config.BuildVersion))
	if err != nil {
		fmt.Println("failed to retrieve the current version")
		return
	}

	client := command.NewHttpClient(util.WithTimeout(30 * time.Second))
	var releases = []util.Release{&util.GithubRelease{Http: client}, &util.GiteeRelease{Http: client}}

	for _, release := range releases {
		item, err := release.Last("jianggujin", "lvs")
		if err != nil {
			fmt.Printf("failed to retrieve the latest version information from %s(%v)\n", release.Channel(), err)
			continue
		}
		if item == nil {
			fmt.Printf("getting the latest version information from %s is empty\n", release.Channel())
			continue
		}
		last, err := version.NewVersion(strings.ToLower(item.Name))
		if err != nil {
			fmt.Printf("failed to retrieve the latest version information from %s(%v)\n", release.Channel(), err)
			continue
		}
		if last.GreaterThan(current) {
			fmt.Printf("there is a new version available [%s => %s](%s)", current.Original(), last.Original(), release.Channel())
			fmt.Printf("new version information [%s]\n", item.Url)
		} else {
			fmt.Printf("the current version [%s] is already the latest version(%s)", current.Original(), release.Channel())
		}
	}
}

func (command *UpgradeCommand) NewHttpClient(opts ...util.HttpClientOption) *http.Client {
	ops := append([]util.HttpClientOption{util.WithProxyStr(config.GetString(config.KeyLvsProxy))}, opts...)
	return util.NewHttpClient(ops...)
}

func (command *UpgradeCommand) Get(url string, opts ...util.HttpClientOption) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 LVS/%s", config.BuildVersion))
	return command.NewHttpClient(opts...).Do(req)
}
