package i18n

import (
	"fmt"
	"sync/atomic"
)

type entry struct {
	en string
	tw string
}

var uiTable = map[string]entry{
	"显示面板":          {"Show panel", "顯示面板"},
	"收起面板":          {"Hide panel", "收起面板"},
	"钉住面板":          {"Pin panel", "釘住面板"},
	"贴边自动隐藏":        {"Auto-hide at edge", "貼邊自動隱藏"},
	"开机自启":          {"Start with Windows", "開機自動啟動"},
	"设置…":           {"Settings…", "設定…"},
	"退出":            {"Quit", "結束"},
	"智爱小助手":         {"ZhiAi Assistant", "智愛小助手"},
	"智爱小助手设置":       {"ZhiAi Assistant Settings", "智愛小助手設定"},
	"%d 项待办":        {"%d to-do", "%d 項待辦"},
	"%d 项逾期":        {"%d overdue", "%d 項逾期"},
	"%s（热键被占用）":     {"%s (hotkey in use)", "%s（熱鍵被占用）"},
	"%s（热键注册失败）":    {"%s (hotkey registration failed)", "%s（熱鍵註冊失敗）"},
	"%s · 有新版本 v%s": {"%s · v%s available", "%s · 有新版本 v%s"},
	"所有文件":          {"All files", "所有檔案"},
	"提示音":           {"Sounds", "提示音"},
	"程序与快捷方式":       {"Programs and shortcuts", "程式與捷徑"},
	"文件夹快捷方式":       {"Folder shortcuts", "資料夾捷徑"},
	"选择要收纳的文件":      {"Choose files to collect", "選擇要收納的檔案"},
	"选择要添加的附件":      {"Choose attachments to add", "選擇要新增的附件"},
	"选择要收纳的文件夹":     {"Choose a folder to collect", "選擇要收納的資料夾"},
	"选择要导入的备份":      {"Choose a backup to import", "選擇要匯入的備份"},
	"选择提示音（WAV）":    {"Choose a sound (WAV)", "選擇提示音（WAV）"},
	"请选择 .wav 文件":   {"Please choose a .wav file", "請選擇 .wav 檔案"},
	"未选择文件":         {"No file selected", "未選擇檔案"},
	"无法播放铃音":        {"Unable to play the sound", "無法播放鈴聲"},
	"正在启动…":         {"Starting…", "正在啟動…"},
	"正在加载界面…":       {"Loading interface…", "正在載入介面…"},
	"已就绪":           {"Ready", "已就緒"},
	"界面初始化失败":       {"Interface failed to start", "介面初始化失敗"},
	"启动超时":          {"Startup timed out", "啟動逾時"},
}

var current atomic.Value

func lookup(source string) (entry, bool) {
	if e, ok := uiTable[source]; ok {
		return e, true
	}
	e, ok := errorTable[source]
	return e, ok
}

func (e entry) pick(locale string) string {
	switch locale {
	case "en":
		return e.en
	case "zh-TW":
		return e.tw
	}
	return ""
}

func Set(locale string) {
	if locale != "en" && locale != "zh-TW" {
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
	locale := Current()
	if locale != "en" && locale != "zh-TW" {
		return source
	}
	e, ok := lookup(source)
	if !ok {
		return source
	}
	return e.pick(locale)
}

func TF(source string, args ...interface{}) string {
	return fmt.Sprintf(T(source), args...)
}
