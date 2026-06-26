package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"

	jsoniter "github.com/json-iterator/go"
)

// 从 GitHub Releases API 获取 Helium 版本信息
func getHeliumInfo(data *SettingsData) (HeliumInfo, error) {
	apiUrl := "https://api.github.com/repos/imputnet/helium-windows/releases?per_page=5"
	client, reqUrl := setProxy(data, apiUrl)

	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return HeliumInfo{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return HeliumInfo{}, fmt.Errorf("GitHub API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return HeliumInfo{}, err
	}

	var releases []GithubRelease
	if err := jsoniter.Unmarshal(body, &releases); err != nil {
		return HeliumInfo{}, fmt.Errorf("解析 GitHub API 响应失败: %w", err)
	}

	if len(releases) == 0 {
		return HeliumInfo{}, fmt.Errorf("未找到任何发布版本")
	}

	// 获取用户选择的架构
	arch, _ := data.arch.Get()
	if arch == "" {
		arch = "x64"
	}

	// 查找最新的非预发布版本
	for _, release := range releases {
		if release.Prerelease {
			continue
		}
		for _, asset := range release.Assets {
			// 匹配便携版 .zip: helium_{version}_{arch}-windows.zip
			targetSuffix := fmt.Sprintf("_%s-windows.zip", arch)
			if strings.HasSuffix(asset.Name, targetSuffix) {
				sha256 := strings.TrimPrefix(asset.Digest, "sha256:")
				return HeliumInfo{
					Version:     release.TagName,
					Size:        int64(asset.Size),
					Sha256:      strings.ToUpper(sha256),
					DownloadUrl: asset.BrowserDownloadURL,
				}, nil
			}
		}
	}

	return HeliumInfo{}, fmt.Errorf("未找到架构 %s 的便携版发布包", arch)
}

// 获取下载文件名
func getHeliumDownloadFileName(downloadUrl string) string {
	return filepath.Base(downloadUrl)
}

func getHttpProxyClient(sd *SettingsData) *http.Client {
	ghProxy := getString(sd.ghProxy)
	if ghProxy == "" {
		return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
			Proxy: GetProxyURL(),
		}}
	}

	// Ensure proper proxy URL prefix
	proxyType := getString(sd.proxyType)
	if proxyType == "HTTP(S)" && !strings.HasPrefix(ghProxy, "http") {
		ghProxy = "http://" + ghProxy
	} else if proxyType == "SOCKS5" && !strings.HasPrefix(ghProxy, "socks5") {
		ghProxy = "socks5://" + ghProxy
	}

	urlproxy, err := url.Parse(ghProxy)
	if err != nil {
		logger.Errorf("Invalid proxy URL: %v", err)
		return &http.Client{Timeout: 5 * time.Second}
	}

	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(urlproxy),
		},
	}
}

func GetSystemProxy() (enabled bool, proxyServer string, err error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return false, "", err
	}
	defer key.Close()
	enable, _, err := key.GetIntegerValue("ProxyEnable")
	if err != nil {
		return false, "", err
	}

	server, _, err := key.GetStringValue("ProxyServer")
	if err != nil {
		return false, "", err
	}

	return enable == 1, server, nil
}

func GetProxyURL() func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		// 1. 检测系统代理
		enabled, proxyServer, err := GetSystemProxy()
		if err == nil && enabled && proxyServer != "" {
			return url.Parse("http://" + proxyServer) // 假设是 HTTP 代理
		}

		// 2. 回退到环境变量
		return http.ProxyFromEnvironment(req)
	}
}
