package bootstrap

import (
	"fmt"
	"github.com/aoaostar/mooc/pkg/config"
	"github.com/aoaostar/mooc/pkg/task"
	"github.com/aoaostar/mooc/pkg/util"
	"github.com/aoaostar/mooc/pkg/yinghua"
	"github.com/aoaostar/mooc/pkg/yinghua/types"
	"github.com/sirupsen/logrus"
)

func Run() {

	InitLog()

	util.Copyright()

	err := InitConfig()

	if err != nil {
		logrus.Fatal(err)
	}

	// 启动Web服务
	InitWeb()

	logrus.Info("程序已启动并进入待机状态，请通过网页控制执行任务")

	// 阻塞主线程，保持程序运行
	select {}
}
// send 处理单个用户的课程任务
func send(user config.User) {

	instance := yinghua.New(user)

	err := instance.Login()
	if err != nil {
		logrus.Error(fmt.Sprintf("用户 %s 登录失败: %v", user.Username, err))
		return
	}
	instance.Output("登录成功")

	err = instance.GetCourses()
	if err != nil {
		logrus.Error(fmt.Sprintf("用户 %s 获取课程列表失败: %v", user.Username, err))
		return
	}

	instance.Output(fmt.Sprintf("获取全部在学课程成功, 共计 %d 门\n", len(instance.Courses)))

	// 清空之前的任务列表
	task.Tasks = []task.Task{}

	// 检查是否指定了课程名称
	if len(user.CourseNames) > 0 {
		// 根据课程名称筛选课程
		var selectedCourses []types.CoursesList
		for _, name := range user.CourseNames {
			courses, err := instance.GetCourseByName(name)
			if err != nil {
				logrus.Error(fmt.Sprintf("查找课程 '%s' 失败: %v", name, err))
				continue
			}

			if len(courses) == 0 {
				logrus.Warn(fmt.Sprintf("未找到课程: '%s'", name))
			} else {
				selectedCourses = append(selectedCourses, courses...)
				instance.Output(fmt.Sprintf("找到课程: '%s', 共 %d 个匹配结果", name, len(courses)))
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
		for _, course := range instance.Courses {
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
	instance.Output("登录成功")

	err = instance.GetCourses()
	if err != nil {
		logrus.Fatal(err)
	}

	instance.Output(fmt.Sprintf("获取全部在学课程成功, 共计 %d 门\n", len(instance.Courses)))

	// 检查是否指定了课程名称
	if len(user.CourseNames) > 0 {
		// 根据课程名称筛选课程
		var selectedCourses []types.CoursesList
		for _, name := range user.CourseNames {
			courses, err := instance.GetCourseByName(name)
			if err != nil {
				logrus.Error(fmt.Sprintf("查找课程 '%s' 失败: %v", name, err))
				continue
			}

			if len(courses) == 0 {
				logrus.Warn(fmt.Sprintf("未找到课程: '%s'", name))
			} else {
				selectedCourses = append(selectedCourses, courses...)
				instance.Output(fmt.Sprintf("找到课程: '%s', 共 %d 个匹配结果", name, len(courses)))
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
		for _, course := range instance.Courses {
			task.Tasks = append(task.Tasks, task.Task{
				User:   user,
				Course: course,
				Status: false,
			})
		}
	}
}
