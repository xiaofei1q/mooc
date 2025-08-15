package task

import (
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/aoaostar/mooc/pkg/config"
	"github.com/aoaostar/mooc/pkg/yinghua"
	"github.com/aoaostar/mooc/pkg/yinghua/types"
	"github.com/sirupsen/logrus"
)

// 用户课程进度跟踪结构
type UserCourseProgress struct {
	UserID     string  // 用户名作为唯一标识
	CourseID   int     // 课程ID
	CourseName string  // 课程名称
	Progress   float64 // 0-100 的百分比
	Status     string  // "pending", "in_progress", "completed", "failed"
}

// 进度跟踪变量
var (
	progress struct {
		Total     int
		Completed int
		mu        sync.Mutex
	}

	// 使用映射存储每个用户的课程进度
	UserProgressMap = struct {
		data map[string]map[int]UserCourseProgress
		mu   sync.Mutex
	}{
		data: make(map[string]map[int]UserCourseProgress),
	}
)

type Task struct {
	User   config.User
	Course types.CoursesList
	Status bool
}

var Tasks []Task

func Start() {
	// 检查是否有停止标记，如果有则删除
	stopFile := "./stop_flag"
	if _, err := os.Stat(stopFile); err == nil {
		if err := os.Remove(stopFile); err != nil {
			logrus.Error("删除停止标记文件失败: ", err)
		}
	}

	// 初始化进度跟踪
	progress.mu.Lock()
	progress.Total = len(Tasks)
	progress.Completed = 0
	progress.mu.Unlock()

	// 初始化用户课程进度映射
	UserProgressMap.mu.Lock()
	UserProgressMap.data = make(map[string]map[int]UserCourseProgress)
	for _, task := range Tasks {
		userID := task.User.Username
		if _, exists := UserProgressMap.data[userID]; !exists {
			UserProgressMap.data[userID] = make(map[int]UserCourseProgress)
		}
		UserProgressMap.data[userID][task.Course.ID] = UserCourseProgress{
			UserID:     userID,
			CourseID:   task.Course.ID,
			CourseName: task.Course.Name,
			Progress:   0,
			Status:     "pending",
		}
	}
	UserProgressMap.mu.Unlock()

	limit := int(math.Min(float64(config.Conf.Global.Limit), float64(len(Tasks))))
	jobs := make(chan Task, limit)
	wg := sync.WaitGroup{}
	for i := 0; i < limit; i++ {
		go func() {
			defer wg.Done()
			for job := range jobs {
				work(job)
				// 更新完成任务数
				progress.mu.Lock()
				progress.Completed++
				progress.mu.Unlock()
			}
		}()
		wg.Add(1)
	}

	logrus.Infof("任务系统启动成功, 协程数: %d, 任务数: %d", limit, len(Tasks))

	for _, task := range Tasks {
		jobs <- task
	}
	close(jobs)
	wg.Wait()
	logrus.Infof("恭喜您, 所有任务都已全部完成~~~ %d", len(Tasks))
}

// 检查是否存在停止标记
func checkStopFlag() bool {
	stopFile := "./stop_flag"
	_, err := os.Stat(stopFile)
	return err == nil
}

// GetProgress 获取当前任务进度
func GetProgress() (total int, completed int, percentage float64) {
	progress.mu.Lock()
	defer progress.mu.Unlock()

	total = progress.Total
	completed = progress.Completed

	if total > 0 {
		percentage = float64(completed) / float64(total) * 100
	} else {
		percentage = 0
	}

	return
}

func work(task Task) {
	userID := task.User.Username
	courseID := task.Course.ID

	// 更新任务状态为进行中
	UserProgressMap.mu.Lock()
	if _, exists := UserProgressMap.data[userID]; exists {
		if progress, exists := UserProgressMap.data[userID][courseID]; exists {
			progress.Status = "in_progress"
			UserProgressMap.data[userID][courseID] = progress
		}
	}
	UserProgressMap.mu.Unlock()

	// 检查是否有停止标记
	if checkStopFlag() {
		logrus.Info("检测到停止标记，跳过任务")

		// 更新任务状态为失败
		UserProgressMap.mu.Lock()
		if _, exists := UserProgressMap.data[userID]; exists {
			if progress, exists := UserProgressMap.data[userID][courseID]; exists {
				progress.Status = "failed"
				UserProgressMap.data[userID][courseID] = progress
			}
		}
		UserProgressMap.mu.Unlock()
		return
	}
	instance := yinghua.New(task.User) // 检查是否有停止标记
	if checkStopFlag() {
		logrus.Info("检测到停止标记，取消登录")
		return
	}
	err := instance.Login()
	if err != nil {
		logrus.Fatal(err)
	}

	instance.Output("登录成功")

	// 检查是否有停止标记
	if checkStopFlag() {
		logrus.Info("检测到停止标记，取消课程处理")
		return
	}

	if task.Course.Progress == 1 {
		instance.Output(fmt.Sprintf("当前课程[%s][%d] 进度: %s, 跳过", task.Course.Name, task.Course.ID, task.Course.Progress1))

		// 更新任务状态为完成
		UserProgressMap.mu.Lock()
		if _, exists := UserProgressMap.data[userID]; exists {
			if progress, exists := UserProgressMap.data[userID][courseID]; exists {
				progress.Progress = 100
				progress.Status = "completed"
				UserProgressMap.data[userID][courseID] = progress
			}
		}
		UserProgressMap.mu.Unlock()

		return
	}
	if task.Course.State == 2 {
		instance.Output(fmt.Sprintf("当前课程[%s][%d] 已结束, 进度设置为100%%", task.Course.Name, task.Course.ID))

		// 更新任务状态为完成
		UserProgressMap.mu.Lock()
		// 确保用户进度映射存在
		if _, exists := UserProgressMap.data[userID]; !exists {
			UserProgressMap.data[userID] = make(map[int]UserCourseProgress)
		}
		// 确保课程进度存在并更新
		progress := UserProgressMap.data[userID][courseID]
		progress.Progress = 100
		progress.Status = "completed"
		UserProgressMap.data[userID][courseID] = progress
		UserProgressMap.mu.Unlock()

		return
	}
	instance.Output(fmt.Sprintf("当前课程[%s][%d] 进度: %s", task.Course.Name, task.Course.ID, task.Course.Progress1))
	err = instance.StudyCourse(task.Course)
	if err != nil {
		instance.OutputWith(fmt.Sprintf("课程[%s][%d]: %s", task.Course.Name, task.Course.ID, err.Error()), logrus.Errorf)

		// 更新任务状态为失败
		UserProgressMap.mu.Lock()
		if _, exists := UserProgressMap.data[userID]; exists {
			if progress, exists := UserProgressMap.data[userID][courseID]; exists {
				progress.Status = "failed"
				UserProgressMap.data[userID][courseID] = progress
			}
		}
		UserProgressMap.mu.Unlock()
	} else {
		// 更新任务状态为完成
		UserProgressMap.mu.Lock()
		if _, exists := UserProgressMap.data[userID]; exists {
			if progress, exists := UserProgressMap.data[userID][courseID]; exists {
				progress.Progress = 100
				progress.Status = "completed"
				UserProgressMap.data[userID][courseID] = progress
			}
		}
		UserProgressMap.mu.Unlock()
	}

}

// GetUserCourseProgress 获取所有用户的课程进度数据
func GetUserCourseProgress() map[string]map[int]UserCourseProgress {
	UserProgressMap.mu.Lock()
	defer UserProgressMap.mu.Unlock()

	// 创建一个副本以避免并发问题
	result := make(map[string]map[int]UserCourseProgress)
	for userID, courses := range UserProgressMap.data {
		result[userID] = make(map[int]UserCourseProgress)
		for courseID, progress := range courses {
			result[userID][courseID] = progress
		}
	}

	return result
}
