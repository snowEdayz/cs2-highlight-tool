package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bodgit/sevenzip"
)

var (
	chinaIPOnce   sync.Once
	chinaIPCached bool
)

type GeoIPResponse struct {
	CountryCode string `json:"country_code"`
}

// Gitee Release API 响应结构
type GiteeRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func isChinaIP() bool {
	chinaIPOnce.Do(func() {
		client := &http.Client{
			Timeout: 3 * time.Second,
		}
		resp, err := client.Get(ipTestURL)
		if err != nil {
			chinaIPCached = false
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			chinaIPCached = false
			return
		}

		var geoData GeoIPResponse
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&geoData)
		if err != nil {
			chinaIPCached = false
			return
		}
		countryCode := geoData.CountryCode

		if countryCode == "CN" {
			printSuccess("检测到中国 IP")
			chinaIPCached = true
		} else {
			printWarning("检测到非中国 IP")
			chinaIPCached = false
		}

		// conn, err := net.DialTimeout("tcp", "www.google.com:80", 3*time.Second)
		// if err == nil {
		// 	conn.Close()
		// 	chinaIPCached = false
		// 	return
		// }

		// connCN, errCN := net.DialTimeout("tcp", "www.baidu.com:80", 3*time.Second)
		// if errCN == nil {
		// 	connCN.Close()
		// 	chinaIPCached = true
		// 	return
		// }

		// chinaIPCached = false
	})

	return chinaIPCached
}

func getFFmpegDownloadURL() string {
	if isChinaIP() {
		return ffmpegDownloadURLCN
	}
	return ffmpegDownloadURLGlobal
}

// 获取最新 HLAE 版本信息
func getLatestHLAEVersion() (*GiteeRelease, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	useGitee := isChinaIP()

	apiURL := hlaeReleaseAPIURLGitee
	if !useGitee {
		apiURL = hlaeReleaseAPIURLGitHub
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "CS2-Highlight-Tool")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("版本 API 返回错误: %d", resp.StatusCode)
	}

	var release GiteeRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &release, nil
}

// 下载文件
func downloadFile(url, filepath string) error {
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	printInfo(fmt.Sprintf("开始下载: %s", url))

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	// 显示下载进度
	totalSize := resp.ContentLength
	downloaded := int64(0)
	buffer := make([]byte, 32*1024)
	lastPrint := time.Now()
	emitProgress(true, 0, totalSize <= 0)
	defer emitProgress(false, 0, false)

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			_, writeErr := out.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("写入文件失败: %w", writeErr)
			}
			downloaded += int64(n)

			// 每秒更新一次进度
			if time.Since(lastPrint) > time.Second {
				if totalSize > 0 {
					percent := float64(downloaded) / float64(totalSize) * 100
					emitProgress(true, percent, false)
				} else {
					emitProgress(true, 0, true)
				}
				lastPrint = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取数据失败: %w", err)
		}
	}

	emitProgress(true, 100, false)
	printSuccess(fmt.Sprintf("下载完成: %.2f MB", float64(downloaded)/(1024*1024)))
	return nil
}

// 解压 ZIP 文件
func unzipFile(zipPath, destDir string) error {
	printInfo("正在解压文件...")

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开 ZIP 文件失败: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// 防止 Zip Slip 漏洞
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法的文件路径: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("创建文件失败: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("读取压缩文件失败: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("解压文件失败: %w", err)
		}
	}

	printSuccess("解压完成")
	return nil
}

// 解压 7z 文件
func extract7z(archivePath, destDir string) error {
	printInfo("正在解压文件...")

	r, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("打开 7z 文件失败: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// 防止 Zip Slip 漏洞
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("非法的文件路径: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("创建文件失败: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("读取压缩文件失败: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("解压文件失败: %w", err)
		}
	}

	printSuccess("解压完成")
	return nil
}

func resolveFFmpegDir(exeDir string, cfg *Config) string {
	if cfg != nil && cfg.FFmpegDir != "" {
		return cfg.FFmpegDir
	}
	return filepath.Join(exeDir, "ffmpeg", "bin")
}

func resolveFFmpegExe(exeDir string, cfg *Config) string {
	return filepath.Join(resolveFFmpegDir(exeDir, cfg), "ffmpeg.exe")
}

func findFFmpegRootDir(baseDir string) (string, error) {
	var found string
	err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		candidate := filepath.Join(path, "bin", "ffmpeg.exe")
		if _, statErr := os.Stat(candidate); statErr == nil {
			found = path
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("解压后未找到 ffmpeg.exe")
	}
	return found, nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func moveDir(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyDir(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func ensureFFmpegAvailable(exeDir string, cfg *Config, configPath string) error {
	expectedFFmpegRoot := filepath.Join(exeDir, "ffmpeg")
	expectedFFmpegBin := filepath.Join(expectedFFmpegRoot, "bin")
	if cfg.FFmpegDir != expectedFFmpegBin {
		cfg.FFmpegDir = expectedFFmpegBin
		if configPath != "" {
			if err := saveConfig(configPath, cfg); err != nil {
				return fmt.Errorf("保存配置失败: %w", err)
			}
		}
	}

	ffmpegExe := resolveFFmpegExe(exeDir, cfg)
	if _, err := os.Stat(ffmpegExe); err == nil {
		return nil
	}

	printWarning("未找到 FFmpeg，开始下载...")
	printInfo("正在下载 FFmpeg...")
	tempArchive := filepath.Join(exeDir, "ffmpeg_temp.7z")
	downloadURL := getFFmpegDownloadURL()
	if err := downloadFile(downloadURL, tempArchive); err != nil {
		return fmt.Errorf("下载 FFmpeg 失败: %w", err)
	}
	defer os.Remove(tempArchive)

	extractDir := filepath.Join(exeDir, "_ffmpeg_extract")
	_ = os.RemoveAll(extractDir)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	printInfo("正在解压 FFmpeg...")
	if err := extract7z(tempArchive, extractDir); err != nil {
		return fmt.Errorf("解压 FFmpeg 失败: %w", err)
	}

	sourceDir, err := findFFmpegRootDir(extractDir)
	if err != nil {
		return err
	}
	fallbackBin := filepath.Join(sourceDir, "bin")

	targetDir := expectedFFmpegRoot
	if _, err := os.Stat(targetDir); err == nil {
		if err := os.RemoveAll(targetDir); err != nil {
			printWarning(fmt.Sprintf("清理旧 FFmpeg 目录失败，改用临时目录: %v", err))
			cfg.FFmpegDir = fallbackBin
			if configPath != "" {
				_ = saveConfig(configPath, cfg)
			}
			if _, statErr := os.Stat(resolveFFmpegExe(exeDir, cfg)); statErr == nil {
				printSuccess("FFmpeg 已准备就绪")
				return nil
			}
			return fmt.Errorf("FFmpeg 解压后未找到 ffmpeg.exe")
		}
	}

	if err := moveDir(sourceDir, targetDir); err != nil {
		printWarning(fmt.Sprintf("移动 FFmpeg 目录失败，改用临时目录: %v", err))
		cfg.FFmpegDir = fallbackBin
		if configPath != "" {
			_ = saveConfig(configPath, cfg)
		}
		if _, statErr := os.Stat(resolveFFmpegExe(exeDir, cfg)); statErr == nil {
			printSuccess("FFmpeg 已准备就绪")
			return nil
		}
		return fmt.Errorf("FFmpeg 解压后未找到 ffmpeg.exe")
	}
	_ = os.RemoveAll(extractDir)

	if _, err := os.Stat(resolveFFmpegExe(exeDir, cfg)); err != nil {
		return fmt.Errorf("FFmpeg 解压后未找到 ffmpeg.exe")
	}

	printSuccess("FFmpeg 已准备就绪")
	return nil
}

// 检查并更新 HLAE
func checkAndUpdateHLAE(exeDir string, cfg *Config, configPath string, debugMode bool) error {
	printTitle("\nHLAE 版本检查")

	// 获取最新版本信息
	release, err := getLatestHLAEVersion()
	if err != nil {
		printWarning(fmt.Sprintf("无法获取最新版本信息: %v", err))
		printWarning("将继续使用现有版本")
		return nil
	}

	if len(release.Assets) == 0 {
		printWarning("未找到可用的下载文件")
		return nil
	}

	// 查找 hlae_*.zip 文件（排除 .asc 签名文件）
	var latestAsset *struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}

	for i := range release.Assets {
		asset := &release.Assets[i]
		// 查找 hlae_开头的 .zip 文件，但排除 .zip.asc
		if strings.HasPrefix(asset.Name, "hlae_") &&
			strings.HasSuffix(asset.Name, ".zip") &&
			!strings.HasSuffix(asset.Name, ".zip.asc") {
			latestAsset = asset
			break
		}
	}

	if latestAsset == nil {
		printWarning("未找到 HLAE ZIP 安装包")
		return nil
	}

	latestVersion := latestAsset.Name

	printInfo(fmt.Sprintf("最新版本: %s", latestVersion))
	printInfo(fmt.Sprintf("当前版本: %s", cfg.HLAEVersion))

	// 检查是否需要更新
	hlaeExe := filepath.Join(exeDir, "hlae", "HLAE.exe")
	hlaeMissing := false
	if _, err := os.Stat(hlaeExe); err != nil {
		hlaeMissing = true
	}
	if cfg.HLAEVersion == latestVersion && !hlaeMissing {
		printSuccess("HLAE 已是最新版本")
		return nil
	}

	// 需要更新
	printWarning("检测到新版本，开始更新...")

	// 直接使用 Gitee 下载链接（无需镜像加速）
	downloadURL := latestAsset.BrowserDownloadURL
	printInfo(fmt.Sprintf("下载地址: %s", downloadURL))

	// 下载到临时文件
	printInfo("正在下载 HLAE...")
	tempZip := filepath.Join(exeDir, "hlae_temp.zip")
	if err := downloadFile(downloadURL, tempZip); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}

	// Debug 模式下保留临时文件
	if !debugMode {
		defer os.Remove(tempZip)
	} else {
		printInfo(fmt.Sprintf("Debug 模式：保留临时文件 %s", tempZip))
	}

	// 解压到 hlae 目录
	hlaeDir := filepath.Join(exeDir, "hlae")

	// 如果 hlae 目录已存在，先删除
	if _, err := os.Stat(hlaeDir); err == nil {
		printInfo("删除旧版本...")
		if err := os.RemoveAll(hlaeDir); err != nil {
			return fmt.Errorf("删除旧版本失败: %w", err)
		}
	}

	// 创建 hlae 目录
	if err := os.MkdirAll(hlaeDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 解压
	printInfo("正在解压 HLAE...")
	if err := unzipFile(tempZip, hlaeDir); err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	// 更新配置文件中的版本号
	cfg.HLAEVersion = latestVersion
	if configPath != "" {
		if err := saveConfig(configPath, cfg); err != nil {
			return fmt.Errorf("保存配置失败: %w", err)
		}
	}

	printSuccess(fmt.Sprintf("HLAE 更新完成: %s", latestVersion))
	return nil
}
