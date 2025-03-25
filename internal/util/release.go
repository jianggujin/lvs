package util

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Release interface {
	Last(owner, repo string) (*Item, error)
	Releases(owner, repo string, page, perPage int) ([]*Item, error)
	DownloadUrl(owner, repo, tagName, fileName string) string
	Channel() string
}

type Item struct {
	Name    string `json:"name"`
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Url     string `json:"url"`
}

type GiteeRelease struct {
	UserAgent string
	Http      *http.Client
}

func (r *GiteeRelease) Channel() string {
	return "gitee"
}

func (r *GiteeRelease) Last(owner, repo string) (*Item, error) {
	items, err := r.Releases(owner, repo, 1, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[len(items)-1], nil
}

func (r *GiteeRelease) Releases(owner, repo string, page, perPage int) ([]*Item, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/releases?page=%d&per_page=%d&direction=desc", owner, repo, page, perPage), nil)
	if err != nil {
		return nil, err
	}
	if r.UserAgent != "" {
		req.Header.Set("User-Agent", r.UserAgent)
	}
	resp, err := r.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var items []*Item
	if err = json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	for i := 0; i < len(items); i++ {
		items[i].Url = fmt.Sprintf("https://gitee.com/%s/%s/releases/tag/%s", owner, repo, items[i].TagName)
	}
	return items, nil
}

func (r *GiteeRelease) DownloadUrl(owner, repo, tagName, fileName string) string {
	return fmt.Sprintf("https://gitee.com/%s/%s/releases/download/%s/%s", owner, repo, tagName, fileName)
}

type GithubRelease struct {
	UserAgent string
	Http      *http.Client
}

func (r *GithubRelease) Channel() string {
	return "github"
}

func (r *GithubRelease) Last(owner, repo string) (*Item, error) {
	items, err := r.Releases(owner, repo, 1, 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[len(items)-1], nil
}

func (r *GithubRelease) Releases(owner, repo string, page, perPage int) ([]*Item, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?page=%d&per_page=%d", owner, repo, page, perPage), nil)
	if err != nil {
		return nil, err
	}
	if r.UserAgent != "" {
		req.Header.Set("User-Agent", r.UserAgent)
	}
	resp, err := r.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var items []*Item
	if err = json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	for i := 0; i < len(items); i++ {
		items[i].Url = fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, items[i].TagName)
	}
	return items, nil
}

func (r *GithubRelease) DownloadUrl(owner, repo, tagName, fileName string) string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", owner, repo, tagName, fileName)
}
