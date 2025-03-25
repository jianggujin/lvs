//go:build (windows && (amd64 || 386 || arm64)) || (linux && (amd64 || arm || armv7l || arm64 || ppc64le || s390x)) || (darwin && (amd64 || arm64))

package node

import (
	"errors"
	"fmt"
	version2 "github.com/hashicorp/go-version"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"io"
	"jianggujin.com/lvs/internal/config"
	"jianggujin.com/lvs/internal/util"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func init() {
	nodeCmd.AddCommand(&InstallCommand{})
}

type InstallCommand struct {
	Latest      bool
	Lts         bool
	Security    bool
	Force       bool
	stepCount   int
	currentStep int
}

func (command *InstallCommand) Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Install the specified node.js version",
		Aliases: []string{"i"},
		RunE:    command.RunE,
	}
	flags := cmd.Flags()
	flags.BoolVarP(&command.Latest, "latest", "l", true, "latest version, if false, use the earliest version")
	flags.BoolVarP(&command.Lts, "lts", "L", true, "long-term support version")
	flags.BoolVarP(&command.Security, "security", "s", false, "security fix version")
	flags.BoolVarP(&command.Force, "force", "f", false, "force download of specified version")
	return cmd
}

func (command *InstallCommand) RunE(_ *cobra.Command, versions []string) error {
	if len(versions) == 0 {
		version, err := config.GetWorkspaceUseVersion(config.ModuleNode)
		if err != nil && !os.IsNotExist(err) {
			return util.WrapError(err)
		}
		if version == "" {
			return nil
		}
		versions = []string{version}
	}

	installHome := config.GetPath(config.KeyNodeHome)
	tempHome := config.GetPath(config.KeyLvsTempHome)
	version := config.GetStringWithDefault(config.KeyNodeAliasPrefix+versions[0], versions[0])
	if version == "latest" {
		command.Latest = true
		command.Force = false
	} else {
		version = nodeCmd.FixVersion(version)
	}
	fmt.Printf("install %s start(latest: %v lts: %v security: %v force: %v)\n", version, command.Latest, command.Lts, command.Security, command.Force)

	command.stepCount = 4
	if !command.Force {
		command.stepCount = 5
		var err error
		version, err = command.findMatchVersion(version)
		if err != nil {
			return util.WrapErrorMsg("find match version error").SetErr(err)
		}
		version = nodeCmd.FixVersion(version)
	} else {
		if _, err := nodeCmd.Semver(version); err != nil {
			return util.WrapErrorMsg("[%s] is not a valid version", version)
		}
	}

	if err := command.install(installHome, tempHome, version); err != nil {
		return util.WrapErrorMsg("install %s error", version).SetErr(err)
	}
	fmt.Printf("install %s finish\n", version)

	return nil
}

func (command *InstallCommand) findMatchVersion(version string) (string, error) {
	rawMsg := "[%d/%d] find matching %s version [%s] %s"
	command.currentStep++
	currentStep := command.currentStep
	spinner := util.Default(-1, fmt.Sprintf(rawMsg, currentStep, command.stepCount, version, "?", "█"))
	defer spinner.Close()

	var filter func(*Version) (bool, error)

	if "latest" == version {
		filter = func(version *Version) (bool, error) {
			return true, nil
		}
	} else {
		semver, err := nodeCmd.Semver(version)
		if err != nil {
			spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, version, "none", "×"))
			return "", err
		}

		if semver.Prerelease() != "" || semver.Metadata() != "" || version == nodeCmd.FixVersion(semver.Core().String()) {
			spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, version, version, "√"))
			return version, nil
		}
		segments := semver.Segments()
		for i := len(segments) - 1; i >= 0; i-- {
			if segments[i] != 0 {
				segments[i] = segments[i] + 1
				break
			}
		}
		fmtParts := make([]string, len(segments))
		for i, s := range segments {
			fmtParts[i] = strconv.FormatInt(int64(s), 10)
		}
		constraints, err := version2.NewConstraint(fmt.Sprintf(">=%s,<%s", semver.String(), strings.Join(fmtParts, ".")))
		if err != nil {
			spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, version, "none", "×"))
			return "", err
		}
		filter = func(version *Version) (bool, error) {
			if command.Lts && "false" == cast.ToString(version.Lts) {
				return false, nil
			}
			if command.Security && !version.Security {
				return false, nil
			}

			se, e := version.Semver()
			if e != nil {
				return false, e
			}
			return constraints.Check(se), nil
		}
	}
	versions, err := nodeCmd.ListVersions(filter)
	if err != nil {
		spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, version, "none", "×"))
		return "", err
	}
	if len(versions) == 0 {
		spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, version, "none", "×"))
		return "", errors.New("unable to find a version that matches the criteria")
	}
	if command.Latest {
		spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, version, versions[0].Version, "√"))
		return versions[0].Version, nil
	}
	spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, version, versions[len(versions)-1].Version, "√"))
	return versions[len(versions)-1].Version, nil
}

func (command *InstallCommand) install(home, tempHome, version string) error {
	download, err := nodeCmd.ConvertDownload(version)
	if err != nil {
		return err
	}

	installed, err := command.checkInstallStatus(filepath.Join(home, version), download)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	tempPath, err := command.download(tempHome, download)
	defer os.Remove(tempPath)
	if err != nil {
		return err
	}

	return command.extractArchive(tempPath, download)
}

func (command *InstallCommand) checkInstallStatus(dir string, download *Download) (bool, error) {
	rawMsg := "[%d/%d] detecting [%s] installation status %s"
	command.currentStep++
	currentStep := command.currentStep
	spinner := util.Default(-1, fmt.Sprintf(rawMsg, currentStep, command.stepCount, download.Version, "█"))
	defer spinner.Close()
	name := "node"
	if runtime.GOOS == "windows" {
		name = "node.exe"
		if util.Exists(filepath.Join(dir, name)) {
			spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, download.Version, "√"))
			return true, nil
		}
	} else {
		// linux、darwin解压后放在bin目录中
		if util.Exists(filepath.Join(dir, "bin", name)) {
			spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, download.Version, "√"))
			return true, nil
		}
	}
	spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, download.Version, "×"))
	if err := os.RemoveAll(dir); err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
	}
	return false, nil
}

func (command *InstallCommand) fetchArchive(download *Download, consumer func(*http.Response) error) error {
	rawMsg := "[%d/%d] retrieve [%s] archive file information %s"
	command.currentStep++
	currentStep := command.currentStep
	spinner := util.Default(-1, fmt.Sprintf(rawMsg, currentStep, command.stepCount, download.Version, "█"))
	defer spinner.Close()

	resp, err := nodeCmd.Get(fmt.Sprintf("%s%s/%s.%s",
		config.GetString(config.KeyNodeMirror), download.Version, download.BaseName, download.Ext))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, download.Version, "×"))
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	spinner.Describe(fmt.Sprintf(rawMsg, currentStep, command.stepCount, download.Version, "√"))
	spinner.Close()
	return consumer(resp)
}

func (command *InstallCommand) download(tempHome string, download *Download) (string, error) {
	if err := os.MkdirAll(tempHome, os.ModePerm); err != nil {
		return "", err
	}
	var tempFile string
	err := command.fetchArchive(download, func(resp *http.Response) error {
		tempFile = filepath.Join(tempHome, fmt.Sprintf("%s-%s.%s", download.BaseName, time.Now().Format("20060102150405"), download.Ext))
		file, err := os.Create(tempFile)
		if err != nil {
			return err
		}
		defer file.Close()

		totalSize := resp.ContentLength
		rawMsg := "[%d/%d] download [%s] archive file"
		command.currentStep++
		currentStep := command.currentStep
		bar := util.DefaultBytes(totalSize, fmt.Sprintf(rawMsg, currentStep, command.stepCount, download.Version))
		defer bar.Close()

		_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
		return err
	})
	return tempFile, err
}

func (command *InstallCommand) extractArchive(tempPath string, download *Download) error {
	nodeHome := config.GetPath(config.KeyNodeHome)
	functionFn := func(name string) (string, error) {
		after, ok := strings.CutPrefix(name, download.BaseName)
		if !ok {
			return "", fmt.Errorf("invalid file name %s", name)
		}
		return download.Version + after, nil
	}
	rawMsg := "[%d/%d] extract [%s] archive files"
	command.currentStep++
	currentStep := command.currentStep
	bar := util.DefaultBytes(-1, fmt.Sprintf(rawMsg, currentStep, command.stepCount, download.Version))
	defer bar.Close()

	fn := util.UntarFile

	if "zip" == download.Ext {
		fn = util.UnzipFile
	}
	err := fn(tempPath, nodeHome, functionFn, bar)
	return err
}
