package bootstrap

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 自定义日志格式化器
type CustomFormatter struct {
	logrus.TextFormatter
	// 用户名颜色映射
	userColors map[string]int
} // 初始化用户名颜色映射
func NewCustomFormatter() *CustomFormatter {
	return &CustomFormatter{
		TextFormatter: logrus.TextFormatter{
			ForceColors:     true,
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		},
		userColors: make(map[string]int),
	}
}

// 获取用户名对应的颜色
func (f *CustomFormatter) getUserColor(username string) int {
	// 如果用户已有颜色，直接返回
	if color, exists := f.userColors[username]; exists {
		return color
	}

	// 定义几种可用颜色
	colors := []int{34, 36, 33, 35, 32}

	// 根据用户名哈希选择颜色
	hash := 0
	for _, c := range username {
		hash += int(c)
	}
	color := colors[hash%len(colors)]

	// 存储颜色
	f.userColors[username] = color
	return color
}

// Format 自定义日志格式
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// 自定义日志级别颜色
	levelColor := 0
	switch entry.Level {
	case logrus.DebugLevel:
		levelColor = 36 // Cyan
	case logrus.InfoLevel:
		levelColor = 32 // Green
	case logrus.WarnLevel:
		levelColor = 33 // Yellow
	case logrus.ErrorLevel:
		levelColor = 31 // Red
	case logrus.FatalLevel, logrus.PanicLevel:
		levelColor = 41 // Background Red
	}

	// 格式化日志条目
	timestamp := entry.Time.Format(f.TimestampFormat)

	// 处理消息，为用户名添加颜色
	message := entry.Message
	if f.ForceColors {
		// 查找用户名模式 [用户名]
		// 匹配[协程ID=xxx][username]格式中的用户名
		re := regexp.MustCompile(`\[协程ID=(\d+)\]\[(.*?)\]`)
		matches := re.FindStringSubmatch(message)
		if len(matches) > 2 {
			coroutineID := matches[1]
			username := matches[2]
			userColor := f.getUserColor(username)
			// 替换用户名为带颜色的版本
			message = re.ReplaceAllString(message, fmt.Sprintf("[协程ID=%s]\x1b[%dm[%s]\x1b[0m", coroutineID, userColor, username))
		}

		return []byte(fmt.Sprintf("\x1b[%dm[%s]\x1b[0m [\x1b[35m%s\x1b[0m] %s\n",
			levelColor, timestamp, entry.Level.String(), message)), nil
	}

	return []byte(fmt.Sprintf("[%s] [%s] %s\n",
		timestamp, entry.Level.String(), message)), nil
}

func InitLog() {
	// 创建日志目录
	logDir := "./logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.MkdirAll(logDir, 0755)
	}

	// 配置日志文件写入器
	logWriter := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, fmt.Sprintf("aoaostar-%s.log", time.Now().Format("2006-01-02"))),
		MaxSize:    10,   // 单文件最大容量,单位是MB
		MaxBackups: 7,    // 最大保留过期文件个数
		MaxAge:     30,   // 保留过期文件的最大时间间隔,单位是天
		Compress:   true, // 压缩滚动日志
		LocalTime:  true,
	}

	// 配置日志格式
	logrusFormatter := NewCustomFormatter()
	logrus.SetFormatter(logrusFormatter)

	// 设置日志输出目标
	logrus.SetOutput(io.MultiWriter(os.Stdout, logWriter))

	// 设置日志级别
	logrus.SetLevel(logrus.InfoLevel)

	// 禁用调用者信息
	logrus.SetReportCaller(false)

	logrus.Info("日志系统初始化成功")
}
