package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func baseScreen(win fyne.Window, data *SettingsData) fyne.CanvasObject {
	installPathHandle(data)
	folderEntry := widget.NewEntryWithData(data.installPath)
	folderEntry.OnChanged = func(path string) {
		installPathHandle(data)
	}
	showFolderPicker := func() {
		folderDialog := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err == nil && lu != nil {
				_ = data.installPath.Set(lu.Path())
			}
		}, win)
		currentPath, _ := data.installPath.Get()
		if currentPath != "" {
			if listableURI, err := storage.ListerForURI(storage.NewFileURI(currentPath)); err == nil {
				folderDialog.SetLocation(listableURI)
			}
		}
		folderDialog.Show()
	}
	folderBtn := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), showFolderPicker)
	data.folderEntryStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.folderEntryStatus.Get(); b {
			folderEntry.Disable()
			folderBtn.Disable()
		} else {
			folderEntry.Enable()
			folderBtn.Enable()
		}
	}))
	checkBtn := widget.NewButtonWithIcon(LoadString("CheckBtnLabel"), theme.SearchIcon(), func() {
		err := syncHeliumInfo(data)
		if err != nil {
			alertInfo(LoadString("UpdateCheckErrorMsg"), win)
		}
	})
	createLnkBtn := widget.NewButtonWithIcon(LoadString("CreateLnkBtnLabel"), theme.ContentAddIcon(), func() {
		err := createDeskLnk(data)
		if err != nil {
			alertInfo(LoadString("CreateLnkFailMsg"), win)
		} else {
			alertInfo(LoadString("CreateLnkSuccessMsg"), win)
		}
	})
	downloadBtn = widget.NewButtonWithIcon(LoadString("InstallBtnLabel"), theme.DownloadIcon(), func() {
		ov, _ := data.oldVer.Get()
		cv, _ := data.curVer.Get()
		if cv == ov {
			alertInfo(LoadString("NoNeedUpdateMsg"), win)
		} else {
			parentPath, _ := data.installPath.Get()
			chromeInUse := isProcessExist(filepath.Join(parentPath, "chrome.exe"))
			if chromeInUse {
				alertInfo(LoadString("ChromeRunningMsg"), win)
			} else {
				if runFlag == 1 {
					alertInfo(LoadString("ChromeUpdateRunningMsg"), win)
				} else {
					runFlag = 1
					if getString(data.oldVer) == "-" {
						alertConfirm(LoadString("FirstInstallMsg"), func(b bool) {
							if b {
								execHeliumInstall(data, downloadProgress)
							}
						}, win)
					} else {
						execHeliumInstall(data, downloadProgress)
					}
				}
			}
		}
	})
	data.downBtnStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.downBtnStatus.Get(); b {
			downloadBtn.Disable()
		} else {
			downloadBtn.Enable()
		}
	}))
	data.checkBtnStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.checkBtnStatus.Get(); b {
			checkBtn.Disable()
		} else {
			checkBtn.Enable()
		}
	}))
	// 架构选择（替代原版本分支）
	archRadio := widget.NewRadioGroup([]string{LoadString("X64Option"), LoadString("Arm64Option")}, func(value string) {
		if value == LoadString("X64Option") {
			data.arch.Set("x64")
		} else if value == LoadString("Arm64Option") {
			data.arch.Set("arm64")
		}
	})
	archRadio.Horizontal = true
	currentArch, _ := data.arch.Get()
	if currentArch == "x64" {
		archRadio.Selected = LoadString("X64Option")
	} else if currentArch == "arm64" {
		archRadio.Selected = LoadString("Arm64Option")
	} else {
		archRadio.Selected = LoadString("X64Option")
	}
	buttons := container.NewHBox(folderBtn)
	bar := container.NewBorder(nil, nil, buttons, nil, folderEntry)
	curVerLabel := widget.NewLabelWithData(data.curVer)
	curVerLabel.TextStyle.Bold = true
	oldVer := GetVersion(data, "chrome.exe")
	logger.Info("helium version:", oldVer)
	_ = data.oldVer.Set(oldVer)
	form := widget.NewForm(
		&widget.FormItem{Text: LoadString("InstallLabel"), Widget: bar},
		&widget.FormItem{Text: LoadString("BranchLabel"), Widget: archRadio},
		&widget.FormItem{Text: LoadString("NowVerLabel"), Widget: widget.NewLabelWithData(data.oldVer)},
		&widget.FormItem{Text: LoadString("LatestVerLabel"), Widget: curVerLabel},
		&widget.FormItem{Text: LoadString("FileSizeLabel"), Widget: widget.NewLabelWithData(data.fileSize)},
		&widget.FormItem{Text: "SHA256", Widget: widget.NewLabelWithData(data.SHA256)},
	)
	downloadProgress = widget.NewProgressBar()
	downloadProgress.TextFormatter = func() string {
		fs, _ := data.fileSize.Get()
		if downloadErrorFlag.Load() {
			if msg := downloadErrorMsg.Load(); msg != nil {
				return msg.(string)
			}
			return LoadString("DownloadFailedMsg")
		} else if downloadProgress.Max*0.9 == downloadProgress.Value {
			return fmt.Sprintf(LoadString("DownLoadedProcessMsg"), fs)
		} else if downloadProgress.Max == downloadProgress.Value {
			return LoadString("InstalledMsg")
		} else if downloadProgress.Value == 0.95 {
			return LoadString("Download95Msg")
		}
		fsFloatStr := strings.Split(fs, " ")[0]
		fsFloat, err := strconv.ParseFloat(fsFloatStr, 64)
		if err != nil {
			return LoadString("DownloadNotStartedMsg")
		}
		return fmt.Sprintf(LoadString("DownloadingMsg"), fsFloat*downloadProgress.Value, fs)
	}
	data.processStatus.AddListener(binding.NewDataListener(func() {
		if b, _ := data.processStatus.Get(); b {
			downloadProgress.Show()
		} else {
			downloadProgress.Hide()
		}
	}))
	if !getBool(data.autoUpdate) {
		go func() {
			err := syncHeliumInfo(data)
			if err != nil {
				alertInfo(LoadString("UpdateCheckErrorMsg"), win)
			}
		}()
	}
	logger.Debug("Base tab load success.")
	return container.New(&buttonLayout{}, form, container.NewVBox(downloadProgress, container.NewGridWithColumns(3, checkBtn, downloadBtn, createLnkBtn)))
}

func syncHeliumInfo(data *SettingsData) error {
	info, err := getHeliumInfo(data)
	if err != nil {
		return err
	}
	data.curVer.Set(info.Version)
	data.fileSize.Set(formatFileSize(info.Size))
	data.fileSizeRaw.Set(int(info.Size))
	data.SHA256.Set(info.Sha256)
	data.downBtnStatus.Set(false)
	return nil
}

func execHeliumInstall(data *SettingsData, downloadProgress *widget.ProgressBar) {
	data.checkBtnStatus.Set(true)
	data.folderEntryStatus.Set(true)
	data.processStatus.Set(true)

	info, err := getHeliumInfo(data)
	if err != nil {
		logger.Errorf("获取下载信息失败: %v", err)
		downloadErrorFlag.Store(true)
		downloadErrorMsg.Store("获取版本信息失败: " + err.Error())
		data.checkBtnStatus.Set(false)
		data.folderEntryStatus.Set(false)
		runFlag = 0
		return
	}

	parentPath, _ := data.installPath.Get()
	downloadErrorFlag.Store(false)
	downloadProgress.SetValue(0)
	fileName := getHeliumDownloadFileName(info.DownloadUrl)
	fileName = filepath.Join(parentPath, fileName)

	expectedSha256 := strings.TrimSpace(info.Sha256)

	// 先检查本地是否已有有效文件，跳过下载
	needDownload := true
	if fileExist(fileName) {
		localSha256 := sumFileSHA256(fileName)
		if expectedSha256 != "" && localSha256 == expectedSha256 {
			logger.Info("本地已有有效安装包，跳过下载")
			needDownload = false
		} else {
			logger.Infof("本地文件 SHA256 不匹配，重新下载 (本地=%s, 期望=%s)", localSha256, expectedSha256)
			os.Remove(fileName)
		}
	}

	if needDownload {
		dl := NewDownloader(data, info.DownloadUrl, fileName, 16, downloadProgress)
		if fs, _ := data.fileSizeRaw.Get(); fs > 0 {
			dl.FileSize = int64(fs)
		}
		dl.UseProxy = getBool(data.downloadChromeViaProxy)
		dl.Start()

		err = <-dl.Done
		if err != nil {
			logger.Errorf("下载失败: %v", err)
			downloadErrorFlag.Store(true)
			downloadErrorMsg.Store("下载文件失败: " + err.Error())
			fyne.DoAndWait(func() { downloadProgress.SetValue(0) })
			data.checkBtnStatus.Set(false)
			data.folderEntryStatus.Set(false)
			runFlag = 0
			return
		}

		// SHA256 校验
		sha256 := sumFileSHA256(fileName)
		if expectedSha256 != "" && sha256 != expectedSha256 {
			logger.Errorf("SHA256 校验失败: 期望=%s, 实际=%s", expectedSha256, sha256)
			downloadErrorFlag.Store(true)
			downloadErrorMsg.Store("SHA256 校验失败")
			fyne.DoAndWait(func() { downloadProgress.SetValue(0) })
			data.checkBtnStatus.Set(false)
			data.folderEntryStatus.Set(false)
			runFlag = 0
			return
		}
	}

	// 静默运行安装程序
	fyne.DoAndWait(func() { downloadProgress.SetValue(0.95) })

	cmd := exec.Command(fileName, "/S", fmt.Sprintf("/D=%s", parentPath))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := cmd.CombinedOutput(); err != nil {
		logger.Errorf("安装程序执行失败: %v, 输出: %s", err, string(out))
		downloadErrorFlag.Store(true)
		downloadErrorMsg.Store(fmt.Sprintf("安装程序执行失败: %v", err))
		fyne.DoAndWait(func() { downloadProgress.SetValue(0) })
		data.checkBtnStatus.Set(false)
		data.folderEntryStatus.Set(false)
		runFlag = 0
		return
	}

	// 验证安装是否成功
	if !fileExist(filepath.Join(parentPath, "chrome.exe")) {
		logger.Error("安装后未找到 chrome.exe")
		downloadErrorFlag.Store(true)
		downloadErrorMsg.Store("安装完成但未找到 chrome.exe")
		fyne.DoAndWait(func() { downloadProgress.SetValue(0) })
		data.checkBtnStatus.Set(false)
		data.folderEntryStatus.Set(false)
		runFlag = 0
		return
	}

	// 清理安装包
	if !getBool(data.remainInstallFileSettings) {
		_ = os.Remove(fileName)
	}
	if !getBool(data.remainHistoryFileSettings) {
		_ = os.RemoveAll(filepath.Join(parentPath, getString(data.oldVer)))
	}

	fyne.DoAndWait(func() { downloadProgress.SetValue(1) })
	data.oldVer.Set(info.Version)
	data.checkBtnStatus.Set(false)
	data.folderEntryStatus.Set(false)
	runFlag = 0
}

func createDeskLnk(data *SettingsData) error {
	parentPath, _ := data.installPath.Get()
	exePath := filepath.Join(parentPath, "chrome.exe")
	if fileExist(exePath) {
		desktopPath, err := GetDesktopPath()
		if err != nil {
			logger.Debug(err)
			return err
		}
		logger.Debug("Desktop Path:", desktopPath)
		linkPath := filepath.Join(desktopPath, "Helium.lnk")
		err = makeLink(exePath, linkPath)
		if err != nil {
			logger.Debug(err)
		}
		return err
	}
	return errors.New("executable file not found")
}

var (
	downloadProgress  *widget.ProgressBar
	downloadBtn       *widget.Button
	downloadErrorFlag atomic.Bool
	downloadErrorMsg  atomic.Value // string, 存储具体错误信息
)

// 处理Helium安装路径
func installPathHandle(data *SettingsData) {
	p, _ := data.installPath.Get()
	dir, err := os.Getwd()
	if isValidPath(p) {
		dir = p
	} else {
		data.installPath.Set(dir)
	}
	if err != nil {
		logger.Panic(err)
	}

	if findChromeExe(dir) {
		data.installPath.Set(dir)
		oldVer := readHeliumVersion(dir)
		data.oldVer.Set(oldVer)
		logger.Info("helium version:", oldVer)
	} else {
		data.oldVer.Set("-")
		logger.Info("未检测到已安装的 Helium")
	}
	if getBool(data.downBtnStatus) {
		data.checkBtnStatus.Set(false)
	}
}

// 在任何子目录中查找 chrome.exe
func findChromeExe(dir string) bool {
	if fileExist(filepath.Join(dir, "chrome.exe")) {
		return true
	}
	if fileExist(filepath.Join(dir, "Application", "chrome.exe")) {
		return true
	}
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				if fileExist(filepath.Join(dir, e.Name(), "chrome.exe")) {
					return true
				}
			}
		}
	}
	return false
}

// 读取 Helium/chrome.exe 的实际版本
func readHeliumVersion(dir string) string {
	if v := GetVersionFromPath(filepath.Join(dir, "chrome.exe")); v != "" {
		return v
	}
	if v := GetVersionFromPath(filepath.Join(dir, "Application", "chrome.exe")); v != "" {
		return v
	}
	return "-"
}
