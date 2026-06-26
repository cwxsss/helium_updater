package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/robfig/cron/v3"
)

var cronManager = cron.New(cron.WithSeconds())

func chromeAutoUpdate(a fyne.App, win fyne.Window, data *SettingsData) {
	if desk, ok := a.(desktop.App); ok {
		addUpdateCron(data)
		updateMenu := fyne.NewMenuItem(LoadString("SystemTrayAutoUpdateMenu"), func() {
			_ = data.autoUpdate.Set(!getBool(data.autoUpdate))
		})
		updateMenu.Checked = getBool(data.autoUpdate)
		if getBool(data.autoUpdate) {
			cronManager.Start()
		} else {
			cronManager.Stop()
		}
		m := fyne.NewMenu("",
			updateMenu,
			fyne.NewMenuItem(LoadString("SystemTrayShowMenu"), func() {
				win.Show()
			}),
			fyne.NewMenuItem(LoadString("SystemTrayHideMenu"), func() {
				win.Hide()
			}),
		)
		data.autoUpdate.AddListener(binding.NewDataListener(func() {
			updateMenu.Checked = getBool(data.autoUpdate)
			if getBool(data.autoUpdate) {
				cronManager.Start()
			} else {
				cronManager.Stop()
			}
			m.Refresh()
		}))
		desk.SetSystemTrayMenu(m)
	}
	logger.Debug("Set system tray menu success.")
}

var runFlag = 0

var currentData *SettingsData

func addUpdateCron(data *SettingsData) {
	currentData = data
	spec := "0 0 0/1 * * ?"
	_, _ = cronManager.AddFunc(spec, func() {
		parentPath, _ := data.installPath.Get()
		chromeInUse := isProcessExist(filepath.Join(parentPath, "chrome.exe"))
		if runFlag == 1 || chromeInUse {
			return
		}
		if getString(data.oldVer) != "-" {
			runFlag = 1
			info, err := getHeliumInfo(data)
			if err != nil {
				logger.Errorf("自动更新获取版本信息失败: %v", err)
				runFlag = 0
				return
			}
			_ = data.curVer.Set(info.Version)
			_ = data.fileSize.Set(formatFileSize(info.Size))
			_ = data.SHA256.Set(info.Sha256)
			_ = data.downBtnStatus.Set(false)
			oldVer := GetVersion(data, "chrome.exe")
			logger.Info("helium version:", oldVer)
			_ = data.oldVer.Set(oldVer)
			ov, _ := data.oldVer.Get()
			cv, _ := data.curVer.Get()
			if cv != ov {
				data.fileSizeRaw.Set(int(info.Size))
				autoInstall(data, info)
			}
			runFlag = 0
			downloadBtn.SetText(LoadString("InstallBtnLabel"))
		}
	})
}

func autoInstall(data *SettingsData, info HeliumInfo) {
	parentPath, _ := data.installPath.Get()
	fileName := getHeliumDownloadFileName(info.DownloadUrl)
	fileName = filepath.Join(parentPath, fileName)

	expectedSha256 := info.Sha256

	// 先检查本地文件
	needDownload := true
	if fileExist(fileName) {
		localSha256 := sumFileSHA256(fileName)
		if expectedSha256 != "" && localSha256 == expectedSha256 {
			needDownload = false
		}
	}

	if needDownload {
		sha256 := downloadHelium(info.DownloadUrl, fileName)
		if expectedSha256 != "" && sha256 != expectedSha256 {
			logger.Errorf("自动更新 SHA256 校验失败")
			return
		}
	}

	chromeInUse := isProcessExist(filepath.Join(parentPath, "chrome.exe"))
	if !chromeInUse {
		extractDir := detectExtractDir(parentPath)
		verExtractDir := getVersionExtractDir(extractDir, info.Version)
		// 安全解压到临时目录
		tmpDir := filepath.Join(parentPath, "helium_update_tmp")
		_ = os.RemoveAll(tmpDir)
		if err := unzipAll(fileName, tmpDir); err != nil {
			logger.Errorf("自动更新解压失败: %v", err)
			_ = os.RemoveAll(tmpDir)
			return
		}

		// 嵌套子目录检测
		if !fileExist(filepath.Join(tmpDir, "chrome.exe")) {
			entries, _ := os.ReadDir(tmpDir)
			for _, e := range entries {
				if e.IsDir() && fileExist(filepath.Join(tmpDir, e.Name(), "chrome.exe")) {
					tmpDir = filepath.Join(tmpDir, e.Name())
					break
				}
			}
		}

		if !fileExist(filepath.Join(tmpDir, "chrome.exe")) {
			logger.Error("自动更新: 临时目录中未找到 chrome.exe")
			_ = os.RemoveAll(filepath.Join(parentPath, "helium_update_tmp"))
			return
		}

		// 清理旧版本目录并移动新文件
		cleanHeliumDir(verExtractDir)
		moveFiles(tmpDir, verExtractDir)
		_ = os.RemoveAll(filepath.Join(parentPath, "helium_update_tmp"))

		//清理
		if !getBool(data.remainInstallFileSettings) {
			_ = os.Remove(fileName)
		}
		if !getBool(data.remainHistoryFileSettings) {
			_ = os.RemoveAll(filepath.Join(parentPath, getString(data.oldVer)))
		}
		_ = data.oldVer.Set(info.Version)
	}
}

func downloadHelium(url, fileName string) string {
	autoDownloadProgress := widget.NewProgressBar()
	autoDownloadProgress.SetValue(0)
	autoDownloadProgress.TextFormatter = func() string {
		percentageStr := fmt.Sprintf("%.1f%%", autoDownloadProgress.Value*100.0/0.9)
		downloadBtn.SetText(LoadString("AutoUpdateProgress") + percentageStr)
		return ""
	}

	dl := NewDownloader(nil, url, fileName, 16, autoDownloadProgress)
	if currentData != nil {
		if fs, _ := currentData.fileSizeRaw.Get(); fs > 0 {
			dl.FileSize = int64(fs)
		}
	}
	dl.Start()
	err := <-dl.Done
	if err != nil {
		logger.Errorf("自动更新下载失败: %v", err)
		return ""
	}

	return sumFileSHA256(fileName)
}
