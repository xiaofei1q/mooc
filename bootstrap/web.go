package bootstrap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aoaostar/mooc/pkg/config"
	"github.com/aoaostar/mooc/pkg/task"
	"github.com/aoaostar/mooc/pkg/util"
	"github.com/aoaostar/mooc/pkg/yinghua"
	"github.com/aoaostar/mooc/pkg/yinghua/types"
	"github.com/sirupsen/logrus"
)

// 程序运行状态管理
var (
	programStatus struct {
		isRunning bool
		cmd       *exec.Cmd
		mu        sync.Mutex
	}
)

// InitWeb 初始化Web服务
func InitWeb() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		// 获取User-Agent头信息
		userAgent := strings.ToLower(request.UserAgent())
		
		// 检测是否为移动设备
		isMobile := false
		mobileKeywords := []string{"mobile", "android", "iphone", "ipad", "windows phone", "blackberry", "opera mini", "webos"}
		
		for _, keyword := range mobileKeywords {
			if strings.Contains(userAgent, keyword) {
				isMobile = true
				break
			}
		}
		
		// 根据设备类型选择对应的页面
		var filePath string
		if isMobile {
			filePath = "./view/mobile_index.html"
		} else {
			filePath = "./view/new_index.html"
		}

		file, err := os.Open(filePath)
		if err != nil {
			logrus.Error("无法打开文件: ", err)
			http.Error(writer, "页面未找到", http.StatusNotFound)
			return
		}
		defer file.Close()

		readAll, err := io.ReadAll(file)
		if err != nil {
			logrus.Error("读取文件失败: ", err)
			http.Error(writer, "内部服务器错误", http.StatusInternalServerError)
			return
		}

		_, err = writer.Write(readAll)
		if err != nil {
			logrus.Error("写入响应失败: ", err)
			http.Error(writer, "内部服务器错误", http.StatusInternalServerError)
			return
		}
	})

	http.HandleFunc("/mobile_index.html", func(writer http.ResponseWriter, request *http.Request) {
		// 设置响应头
		http.ServeFile(writer, request, "view/mobile_index.html")
	})
	http.HandleFunc("/ajax", func(writer http.ResponseWriter, request *http.Request) {
		// 获取当前日期，格式为2006-01-02
		currentDate := time.Now().Format("2006-01-02")
		logFilePath := fmt.Sprintf("./logs/aoaostar-%s.log", currentDate)

		text, err := util.ReadText(logFilePath, 0, 100)
		if err != nil {
			logrus.Error(err)

		}
		_, err = io.WriteString(writer, strings.Join(text, "\n"))
		if err != nil {
			logrus.Error(err)
		}

	})

	// 添加获取指定课程的API接口
	http.HandleFunc("/course/", func(writer http.ResponseWriter, request *http.Request) {
		// 设置允许跨域
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Content-Type", "application/json")

		// 从URL路径中提取课程ID
		path := request.URL.Path
		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			writer.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(writer).Encode(map[string]string{"error": "无效的课程ID"})
			return
		}

		courseIDStr := parts[2]
		courseID, err := strconv.Atoi(courseIDStr)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(writer).Encode(map[string]string{"error": "无效的课程ID格式"})
			return
		}

		// 初始化YingHua客户端（使用第一个用户）
		if len(config.Conf.Users) == 0 {
			writer.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(writer).Encode(map[string]string{"error": "未配置用户"})
			return
		}
		yh := yinghua.New(config.Conf.Users[0])

		// 登录
		if err := yh.Login(); err != nil {
			logrus.Error("登录失败:", err)
			writer.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(writer).Encode(map[string]string{"error": "登录失败: " + err.Error()})
			return
		}

		// 获取所有课程
		if err := yh.GetCourses(); err != nil {
			logrus.Error("获取课程列表失败:", err)
			writer.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(writer).Encode(map[string]string{"error": "获取课程列表失败: " + err.Error()})
			return
		}

		// 查找指定ID的课程
		var targetCourse *types.CoursesList
		for _, course := range yh.Courses {
			if course.ID == courseID {
				targetCourse = &course
				break
			}
		}

		if targetCourse == nil {
			writer.WriteHeader(http.StatusNotFound)
			json.NewEncoder(writer).Encode(map[string]string{"error": "未找到指定课程"})
			return
		}

		// 返回课程详情
		writer.WriteHeader(http.StatusOK)
		json.NewEncoder(writer).Encode(targetCourse)
	})

	// 添加通过课程名称获取课程的API接口
	http.HandleFunc("/course/name/", func(writer http.ResponseWriter, request *http.Request) {
		// 设置允许跨域
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Content-Type", "application/json")

		// 从URL路径中提取课程名称
		path := request.URL.Path
		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			writer.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(writer).Encode(map[string]string{"error": "无效的课程名称"})
			return
		}

		courseName := parts[2]
		// 解码URL编码的课程名称
		decodedName, err := url.QueryUnescape(courseName)
		if err != nil {
			decodedName = courseName
		}

		// 初始化YingHua客户端（使用第一个用户）
		if len(config.Conf.Users) == 0 {
			writer.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(writer).Encode(map[string]string{"error": "未配置用户"})
			return
		}
		yh := yinghua.New(config.Conf.Users[0])

		// 登录
		if err := yh.Login(); err != nil {
			logrus.Error("登录失败:", err)
			writer.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(writer).Encode(map[string]string{"error": "登录失败: " + err.Error()})
			return
		}

		// 获取所有课程
		if err := yh.GetCourses(); err != nil {
			logrus.Error("获取课程列表失败:", err)
			writer.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(writer).Encode(map[string]string{"error": "获取课程列表失败: " + err.Error()})
			return
		}

		// 查找指定名称的课程（模糊匹配）
		var targetCourses []types.CoursesList
		for _, course := range yh.Courses {
			if strings.Contains(strings.ToLower(course.Name), strings.ToLower(decodedName)) {
				targetCourses = append(targetCourses, course)
			}
		}

		if len(targetCourses) == 0 {
			writer.WriteHeader(http.StatusNotFound)
			json.NewEncoder(writer).Encode(map[string]string{"error": "未找到指定课程"})
			return
		}

		// 返回匹配的课程列表
		writer.WriteHeader(http.StatusOK)
		json.NewEncoder(writer).Encode(targetCourses)
	})

	// 保存配置接口
	http.HandleFunc("/save-config", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// 解析请求体
		var newConfig config.Config
		if err := json.NewDecoder(request.Body).Decode(&newConfig); err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(writer).Encode(map[string]string{"error": "无效的配置格式"})
			return
		}

		// 保存配置到文件
		configData, err := json.MarshalIndent(newConfig, "", "  ")
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(writer).Encode(map[string]string{"error": "配置序列化失败"})
			return
		}

		if err := os.WriteFile("./config.json", configData, 0644); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(writer).Encode(map[string]string{"error": "保存配置文件失败"})
			return
		}

		// 更新内存中的配置
		config.Conf = newConfig

		writer.WriteHeader(http.StatusOK)
		json.NewEncoder(writer).Encode(map[string]string{"success": "配置保存成功"})
	})

	// 运行程序接口 - 实际是启动任务处理
	http.HandleFunc("/run-program", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		programStatus.mu.Lock()
		defer programStatus.mu.Unlock()

		if programStatus.isRunning {
			writer.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(writer).Encode(map[string]string{"error": "任务已经在运行中"})
			return
		}

		// 标记任务为运行中
		programStatus.isRunning = true

		// 启动协程处理任务
		go func() {
			defer func() {
				programStatus.mu.Lock()
				programStatus.isRunning = false
				programStatus.mu.Unlock()
			}()

			// 遍历所有用户并处理任务
			for _, user := range config.Conf.Users {
				processUserTask(user)
				// 为了避免请求过于频繁，每个用户之间间隔1秒
				time.Sleep(time.Second)
			}
		}()

		writer.WriteHeader(http.StatusOK)
		json.NewEncoder(writer).Encode(map[string]string{"success": "任务已启动"})
	})

	// 停止程序接口
	http.HandleFunc("/stop-program", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		programStatus.mu.Lock()
		defer programStatus.mu.Unlock()

		if !programStatus.isRunning {
			writer.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(writer).Encode(map[string]string{"error": "任务未在运行中"})
			return
		}

		// 清空任务列表以停止任务处理
		task.Tasks = []task.Task{}

		// 添加一个停止标记文件，供任务处理逻辑检查
		stopFile := "./stop_flag"
		if err := os.WriteFile(stopFile, []byte("stop"), 0644); err != nil {
			logrus.Error("创建停止标记文件失败: ", err)
		}

		programStatus.isRunning = false

		writer.WriteHeader(http.StatusOK)
		json.NewEncoder(writer).Encode(map[string]string{"success": "任务已停止"})
	})

	// 查询程序状态接口
	http.HandleFunc("/program-status", func(writer http.ResponseWriter, request *http.Request) {
		programStatus.mu.Lock()
		defer programStatus.mu.Unlock()

		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(map[string]bool{"isRunning": programStatus.isRunning})
	})

	// 查询任务进度接口
	http.HandleFunc("/task-progress", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		// 使用task包中的GetProgress函数获取进度
		total, completed, percentage := task.GetProgress()

		json.NewEncoder(writer).Encode(map[string]interface{}{
			"total":      total,
			"completed":  completed,
			"percentage": percentage,
		})
	})

	// 查询用户课程进度接口
	http.HandleFunc("/user-course-progress", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		// 使用task包中的GetUserCourseProgress函数获取用户课程进度
		userProgress := task.GetUserCourseProgress()

		json.NewEncoder(writer).Encode(userProgress)
	})

	// 读取配置接口
	http.HandleFunc("/get-config", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(config.Conf)
	})

	logrus.Infof("web端启动成功, 请访问 %s 查看服务状态", config.Conf.Global.Server)
	err := http.ListenAndServe(config.Conf.Global.Server, nil)
	if err != nil {
		logrus.Fatal(err.Error())
	}
}

// processUserTask 处理单个用户的课程任务
func processUserTask(user config.User) {
	yh := yinghua.New(user)

	err := yh.Login()
	if err != nil {
		logrus.Error(fmt.Sprintf("用户 %s 登录失败: %v", user.Username, err))
		return
	}
	yh.Output("登录成功")

	err = yh.GetCourses()
	if err != nil {
		logrus.Error(fmt.Sprintf("用户 %s 获取课程列表失败: %v", user.Username, err))
		return
	}

	yh.Output(fmt.Sprintf("获取全部在学课程成功, 共计 %d 门\n", len(yh.Courses)))

	// 注：不再清空任务列表，以便累积所有用户的任务
	// task.Tasks = []task.Task{}

	// 检查是否指定了课程名称
	if len(user.CourseNames) > 0 {
		// 根据课程名称筛选课程
		var selectedCourses []types.CoursesList
		for _, name := range user.CourseNames {
			courses, err := yh.GetCourseByName(name)
			if err != nil {
				logrus.Error(fmt.Sprintf("查找课程 '%s' 失败: %v", name, err))
				continue
			}

			if len(courses) == 0 {
				logrus.Warn(fmt.Sprintf("未找到课程: '%s'", name))
			} else {
				selectedCourses = append(selectedCourses, courses...)
				yh.Output(fmt.Sprintf("找到课程: '%s', 共 %d 个匹配结果", name, len(courses)))
			}
		}

		// 添加筛选后的课程到任务列表
		for _, course := range selectedCourses {
			task.Tasks = append(task.Tasks, task.Task{
				User:   user,
				Course: course,
				Status: false,
			})
		}
	} else {
		// 如果没有指定课程名称，则添加所有课程
		for _, course := range yh.Courses {
			task.Tasks = append(task.Tasks, task.Task{
				User:   user,
				Course: course,
				Status: false,
			})
		}
	}

	// 如果任务列表不为空，启动任务处理
	if len(task.Tasks) > 0 {
		task.Start()
	} else {
		logrus.Warn("没有找到可添加的任务")
	}
}
