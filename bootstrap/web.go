package bootstrap

import (
	"encoding/json"
	"github.com/aoaostar/mooc/pkg/config"
	"github.com/aoaostar/mooc/pkg/util"
	"github.com/aoaostar/mooc/pkg/yinghua"
	"github.com/aoaostar/mooc/pkg/yinghua/types"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"net/url"
)

func InitWeb() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {

		file, err := os.Open("./view/index.html")
		if err != nil {
			logrus.Fatal(err.Error())
		}
		readAll, err := io.ReadAll(file)

		if err != nil {
			logrus.Fatal(err.Error())
		}
		_, err = writer.Write(readAll)

		if err != nil {
			logrus.Fatal(err.Error())
		}

	})
	http.HandleFunc("/ajax", func(writer http.ResponseWriter, request *http.Request) {

	text, err := util.ReadText("./logs/aoaostar.log", 0, 100)
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

	logrus.Infof("web端启动成功, 请访问 %s 查看服务状态", config.Conf.Global.Server)
	err := http.ListenAndServe(config.Conf.Global.Server, nil)
	if err != nil {
		logrus.Fatal(err.Error())
	}

}
