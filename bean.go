package main

import (
	"time"

	"fyne.io/fyne/v2/data/binding"
)

// HeliumInfo helium版本信息
type HeliumInfo struct {
	Version     string `json:"version"`
	Size        int64  `json:"size"`
	Sha256      string `json:"sha256"`
	DownloadUrl string `json:"download_url"`
}

// ChromePlusInfo chrome ++
type ChromePlusInfo struct {
	Version     string `json:"version"`
	DownloadUrl string `json:"downloadurl"`
}

// SysInfo 系统信息
type SysInfo struct {
	goarch, goos string
}

// 配置信息
type SettingsData struct {
	installPath               binding.String //安装目录
	oldVer                    binding.String //旧版本号
	arch                      binding.String //架构选择 (x64/arm64)
	curVer                    binding.String //最新版本号
	fileSize                  binding.String //文件大小(格式化显示)
	fileSizeRaw               binding.Int    //文件大小(原始字节)
	SHA256                    binding.String //文件SHA256
	downBtnStatus             binding.Bool   //下载按钮状态
	checkBtnStatus            binding.Bool   //检查按钮状态
	folderEntryStatus         binding.Bool   //安装目录修改状态
	chromePlus                binding.String //chrome_plus
	oldPlusVer                binding.String //已安装chrome_plus版本
	curPlusVer                binding.String //最新chrome_plus版本
	plusDownloadUrl           binding.String //最新chrome_plus下载地址
	plusFileSizeRaw           binding.Int    //plus文件大小(原始字节)
	plusBtnStatus             binding.Bool   //plus下载安装状态
	plusProcessStatus         binding.Bool   //plus下载安装进度的进度条状态
	processStatus             binding.Bool   //下载安装进度的进度条状态
	remainInstallFileSettings binding.Bool   //是否保留安装文件
	remainHistoryFileSettings binding.Bool   //是否保留历史文件
	themeSettings             binding.String //主题设置
	langSettings              binding.String //语言设置
	ghProxy                   binding.String //Github代理
	proxyType                 binding.String //代理类型
	downloadChromeViaProxy    binding.Bool   //Chrome下载是否走代理
	autoUpdate                binding.Bool   //自动更新
}

// 配置选项
type Config struct {
	InstallPath            string `json:"install_path"`              //安装目录
	Arch                   string `json:"arch"`                      //架构选择
	OldPlusVer             string `json:"old_plus_ver"`              //已安装chrome_plus版本
	ChromePlus             string `json:"chrome_plus"`               //chrome_plus
	RemainInstallFile      bool   `json:"remain_install_file"`       //是否保留安装文件
	RemainHistoryFile      bool   `json:"remain_history_file"`       //是否保留历史文件
	Theme                  string `json:"theme"`                     //主题设置
	Lang                   string `json:"lang"`                      //语言设置
	GhProxy                string `json:"gh_proxy"`                  //Github代理加速
	ProxyType              string `json:"proxy_type"`                //代理类型
	DownloadChromeViaProxy bool   `json:"download_chrome_via_proxy"` //Chrome下载是否走代理
	AutoUpdate             bool   `json:"auto_update"`               //自动更新
}

type GithubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Prerelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		URL                string      `json:"url"`
		ID                 int         `json:"id"`
		NodeID             string      `json:"node_id"`
		Name               string      `json:"name"`
		Label              interface{} `json:"label"`
		ContentType        string      `json:"content_type"`
		State              string      `json:"state"`
		Size               int         `json:"size"`
		DownloadCount      int         `json:"download_count"`
		Digest             string      `json:"digest"`
		CreatedAt          time.Time   `json:"created_at"`
		UpdatedAt          time.Time   `json:"updated_at"`
		BrowserDownloadURL string      `json:"browser_download_url"`
	} `json:"assets"`
	Body string `json:"body"`
}

type TestText struct {
	Label string
}
