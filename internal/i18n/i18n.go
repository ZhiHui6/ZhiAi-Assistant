package i18n

import (
	"fmt"
	"sync/atomic"
)

var current atomic.Value

var dict = map[string]map[string]string{
	"en": {
		"显示面板":          "Show panel",
		"收起面板":          "Hide panel",
		"钉住面板":          "Pin panel",
		"贴边自动隐藏":        "Auto-hide at edge",
		"开机自启":          "Start with Windows",
		"设置…":           "Settings…",
		"退出":            "Quit",
		"智爱小助手":         "ZhiAi Assistant",
		"智爱小助手设置":       "ZhiAi Assistant Settings",
		"%d 项待办":        "%d to-do",
		"%d 项逾期":        "%d overdue",
		"%s（热键被占用）":     "%s (hotkey in use)",
		"%s（热键注册失败）":    "%s (hotkey registration failed)",
		"%s · 有新版本 v%s": "%s · v%s available",
		"所有文件":          "All files",
		"提示音":           "Sounds",
		"程序与快捷方式":       "Programs and shortcuts",
		"文件夹快捷方式":       "Folder shortcuts",
		"选择要收纳的文件":      "Choose files to collect",
		"选择要收纳的文件夹":     "Choose a folder to collect",
		"选择要导入的备份":      "Choose a backup to import",
		"选择提示音（WAV）":    "Choose a sound (WAV)",
		"请选择 .wav 文件":   "Please choose a .wav file",
		"未选择文件":         "No file selected",
		"无法播放铃音":        "Unable to play the sound",
		"正在启动…":         "Starting…",
		"正在加载界面…":       "Loading interface…",
		"已就绪":           "Ready",
		"界面初始化失败":       "Interface failed to start",
		"启动超时":          "Startup timed out",
	},
	"zh-TW": {
		"显示面板":          "顯示面板",
		"收起面板":          "收起面板",
		"钉住面板":          "釘住面板",
		"贴边自动隐藏":        "貼邊自動隱藏",
		"开机自启":          "開機自動啟動",
		"设置…":           "設定…",
		"退出":            "結束",
		"智爱小助手":         "智愛小助手",
		"智爱小助手设置":       "智愛小助手設定",
		"%d 项待办":        "%d 項待辦",
		"%d 项逾期":        "%d 項逾期",
		"%s（热键被占用）":     "%s（熱鍵被占用）",
		"%s（热键注册失败）":    "%s（熱鍵註冊失敗）",
		"%s · 有新版本 v%s": "%s · 有新版本 v%s",
		"所有文件":          "所有檔案",
		"提示音":           "提示音",
		"程序与快捷方式":       "程式與捷徑",
		"文件夹快捷方式":       "資料夾捷徑",
		"选择要收纳的文件":      "選擇要收納的檔案",
		"选择要收纳的文件夹":     "選擇要收納的資料夾",
		"选择要导入的备份":      "選擇要匯入的備份",
		"选择提示音（WAV）":    "選擇提示音（WAV）",
		"请选择 .wav 文件":   "請選擇 .wav 檔案",
		"未选择文件":         "未選擇檔案",
		"无法播放铃音":        "無法播放鈴聲",
		"正在启动…":         "正在啟動…",
		"正在加载界面…":       "正在載入介面…",
		"已就绪":           "已就緒",
		"界面初始化失败":       "介面初始化失敗",
		"启动超时":          "啟動逾時",
	},
}

func Set(locale string) {
	if _, ok := dict[locale]; !ok {
		locale = "zh-CN"
	}
	current.Store(locale)
}

func Current() string {
	if v, ok := current.Load().(string); ok {
		return v
	}
	return "zh-CN"
}

func T(source string) string {
	if d, ok := dict[Current()]; ok {
		if v, ok := d[source]; ok {
			return v
		}
	}
	return source
}

func TF(source string, args ...interface{}) string {
	return fmt.Sprintf(T(source), args...)
}
