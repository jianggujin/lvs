package util

import (
	"fmt"
	"github.com/schollz/progressbar/v3"
	"os"
	"time"
)

func Default(max int64, description string, options ...progressbar.Option) *progressbar.ProgressBar {
	options = append([]progressbar.Option{
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65 * time.Millisecond),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		// 添加详情滚动显示
		// progressbar.OptionSetMaxDetailRow(2),
		progressbar.OptionSetRenderBlankState(true),
	}, options...)
	return progressbar.NewOptions64(max, options...)
}

func DefaultBytes(maxBytes int64, description string, options ...progressbar.Option) *progressbar.ProgressBar {
	options = append([]progressbar.Option{
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowTotalBytes(true),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65 * time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	}, options...)
	return progressbar.NewOptions64(maxBytes, options...)
}
