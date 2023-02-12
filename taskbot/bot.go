package main

import (
	"bytes"
	"context"
	"fmt"
	tgbotapi "github.com/skinass/telegram-bot-api/v5"
	"golang.org/x/exp/slices"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
)

var (
	BotToken   = "5764244057:AAGT9R7Ux_NQbpmnHAE-5k7vfY5LyMSutsE"
	WebhookURL = "https://dabc-188-44-42-48.eu.ngrok.io"
)

type TasksForUser struct {
	User  string
	ID    int64
	ToDo  []int
	Owner []int
}

type Task struct {
	ID    int
	Task  string
	User  int64
	Owner int64
}

var count = 0

// ключ - ID
var tasks = make([]Task, 0)

// ключ - id юзера
var usersTasks = make([]TasksForUser, 0)

type TaskResp struct {
	TaskID            int
	Task              string
	Owner             string
	AssigneeUser      string
	CurrentUserAssign bool
	NotAssignee       bool
	Assignee          bool
}

func HanlerTasks(arg ...string) (string, error) {
	userID, err := strconv.ParseInt(arg[1], 10, 64)
	if err != nil {
		log.Fatalf("HanlerNew failed: %s", err)
		return "", err
	}
	var res string
	if len(tasks) == 0 {
		res = "Нет задач"
	} else {
		var list []string
		for _, task := range tasks {

			resp := TaskResp{Assignee: false, NotAssignee: false, CurrentUserAssign: false}
			resp.TaskID = task.ID
			resp.Task = task.Task

			idxOwner := slices.IndexFunc(usersTasks, func(user TasksForUser) bool {
				return task.Owner == user.ID
			})
			resp.Owner = usersTasks[idxOwner].User

			switch task.User {
			case -1:
				resp.NotAssignee = true
			case userID:
				resp.CurrentUserAssign = true
			default:
				idxUser := slices.IndexFunc(usersTasks, func(user TasksForUser) bool {
					return task.User == user.ID
				})
				resp.AssigneeUser = usersTasks[idxUser].User
				resp.Assignee = true
			}
			buf := new(bytes.Buffer)
			err = tmpl.ExecuteTemplate(buf, "TasksTemplate.txt", resp)
			if err != nil {
				log.Fatalf("HanlerTasks failed: %s", err)
				return "", err
			}
			list = append(list, buf.String())
		}
		res = strings.Join(list, "\n\n")
	}
	return res, nil
}

func HanlerNew(arg ...string) (string, error) {
	userName := arg[0]
	userID, err := strconv.ParseInt(arg[1], 10, 64)
	if err != nil {
		log.Fatalf("HanlerNew failed: %s", err)
		return "", err
	}
	taskStr := arg[2]
	count++
	newTask := Task{
		count,
		taskStr,
		-1,
		userID,
	}
	tasks = append(tasks, newTask)
	idxUser := slices.IndexFunc(usersTasks, func(user TasksForUser) bool {
		return userID == user.ID
	})
	if idxUser != -1 {
		usersTasks[idxUser].Owner = append(usersTasks[idxUser].Owner, count)
	} else {
		usersTasks = append(usersTasks, TasksForUser{
			userName,
			userID,
			[]int{},
			[]int{count},
		})
	}

	buf := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(buf, "NewTemplate.txt", newTask)
	if err != nil {
		log.Fatalf("HanlerNew failed: %s", err)
		return "", err
	}
	return buf.String(), nil
}

func HanlerAssign(arg ...string) (string, error) {
	userName := arg[0]
	userID, err := strconv.ParseInt(arg[1], 10, 64)
	if err != nil {
		log.Fatalf("HanlerAssign failed: %s", err)
		return "", err
	}
	taskID, err := strconv.Atoi(arg[2])
	if err != nil {
		log.Fatalf("HanlerAssign failed: %s", err)
		return "", err
	}

	resp := TaskResp{Assignee: false, NotAssignee: false, CurrentUserAssign: false}

	idxTask := slices.IndexFunc(tasks, func(task Task) bool {
		return taskID == task.ID
	})
	resp.TaskID = tasks[idxTask].ID
	resp.Task = tasks[idxTask].Task

	idxUser := slices.IndexFunc(usersTasks, func(user TasksForUser) bool {
		return userID == user.ID
	})
	if idxUser != -1 {
		usersTasks[idxUser].ToDo = append(usersTasks[idxUser].ToDo, taskID)
	} else {
		usersTasks = append(usersTasks, TasksForUser{
			userName,
			userID,
			[]int{taskID},
			[]int{},
		})
	}

	if tasks[idxTask].User != -1 || userID != tasks[idxTask].Owner {
		resp.Assignee = true
		resp.AssigneeUser = userName
		buf := new(bytes.Buffer)
		err = tmpl.ExecuteTemplate(buf, "AssignTemplate.txt", resp)
		if err != nil {
			log.Fatalf("HanlerTasks failed: %s", err)
			return "", err
		}

		var msgAddr int64
		if tasks[idxTask].User != -1 {
			msgAddr = tasks[idxTask].User
		} else if userID != tasks[idxTask].Owner {
			msgAddr = tasks[idxTask].Owner
		}
		msg := tgbotapi.NewMessage(
			msgAddr,
			buf.String(),
		)
		_, err = bot.Send(msg)
		if err != nil {
			log.Fatalf("HanlerAssign failed: %s", err)
			return "", err
		}
	}
	tasks[idxTask].User = userID
	resp.Assignee = false
	resp.CurrentUserAssign = true
	buf := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(buf, "AssignTemplate.txt", resp)
	if err != nil {
		log.Fatalf("HanlerAssign failed: %s", err)
		return "", err
	}
	return buf.String(), nil
}

func HanlerUnassign(arg ...string) (string, error) {
	userID, err := strconv.ParseInt(arg[1], 10, 64)
	if err != nil {
		log.Fatalf("HanlerUnassign failed: %s", err)
		return "", err
	}
	taskID, err := strconv.Atoi(arg[2])
	if err != nil {
		log.Fatalf("HanlerUnassign failed: %s", err)
		return "", err
	}
	idxUser := slices.IndexFunc(usersTasks, func(user TasksForUser) bool {
		return userID == user.ID
	})
	idxTask := slices.IndexFunc(tasks, func(task Task) bool {
		return taskID == task.ID
	})
	if tasks[idxTask].User == userID {
		idx := slices.IndexFunc(usersTasks[idxUser].ToDo, func(id int) bool {
			return id == taskID
		})
		usersTasks[idxUser].ToDo = slices.Delete(usersTasks[idxUser].ToDo, idx, idx+1)

		tasks[idxTask].User = -1
		buf := new(bytes.Buffer)
		err = tmpl.ExecuteTemplate(buf, "UnassignTemplate.txt", tasks[idxTask])
		if err != nil {
			log.Fatalf("HanlerUnassign failed: %s", err)
			return "", err
		}
		msg := tgbotapi.NewMessage(
			tasks[idxTask].Owner,
			buf.String(),
		)
		_, err = bot.Send(msg)
		if err != nil {
			log.Fatalf("HanlerUnassign failed: %s", err)
			return "", err
		}
		return "Принято", nil

	} else {
		return "Задача не на вас", nil
	}
}

func HanlerResolve(arg ...string) (string, error) {
	userName := arg[0]
	userID, err := strconv.ParseInt(arg[1], 10, 64)
	if err != nil {
		log.Fatalf("HanlerResolve failed: %s", err)
		return "", err
	}
	taskID, err := strconv.Atoi(arg[2])
	if err != nil {
		log.Fatalf("HanlerResolve failed: %s", err)
		return "", err
	}
	idxUser := slices.IndexFunc(usersTasks, func(user TasksForUser) bool {
		return userID == user.ID
	})

	resp := TaskResp{Assignee: false, NotAssignee: false, CurrentUserAssign: false}

	if slices.Contains(usersTasks[idxUser].ToDo, taskID) {

		idx := slices.IndexFunc(usersTasks[idxUser].ToDo, func(id int) bool {
			return id == taskID
		})
		usersTasks[idxUser].ToDo = slices.Delete(usersTasks[idxUser].ToDo, idx, idx+1)
		idxTask := slices.IndexFunc(tasks, func(task Task) bool {
			return taskID == task.ID
		})
		idxOwner := slices.IndexFunc(usersTasks, func(user TasksForUser) bool {
			return tasks[idxTask].Owner == user.ID
		})
		idxTaskOwner := slices.IndexFunc(usersTasks[idxOwner].Owner, func(id int) bool {
			return id == taskID
		})
		usersTasks[idxOwner].Owner = slices.Delete(usersTasks[idxOwner].Owner, idxTaskOwner, idxTaskOwner+1)

		resp.Task = tasks[idxTask].Task
		if tasks[idxTask].Owner != userID {
			resp.NotAssignee = true
			resp.AssigneeUser = userName
			buf := new(bytes.Buffer)
			err = tmpl.ExecuteTemplate(buf, "ResolveTemplate.txt", resp)
			if err != nil {
				log.Fatalf("HanlerResolve failed: %s", err)
				return "", err
			}
			msg := tgbotapi.NewMessage(
				tasks[idxTask].Owner,
				buf.String(),
			)
			_, err = bot.Send(msg)
			if err != nil {
				log.Fatalf("HanlerResolve failed: %s", err)
				return "", err
			}
		}
		resp.NotAssignee = false
		buf := new(bytes.Buffer)
		err = tmpl.ExecuteTemplate(buf, "ResolveTemplate.txt", resp)
		if err != nil {
			log.Fatalf("HanlerResolve failed: %s", err)
			return "", err
		}
		tasks = slices.Delete(tasks, idxTask, idxTask+1)
		return buf.String(), nil
	}
	return "", nil
}

func HanlerMy(arg ...string) (string, error) {
	userID, err := strconv.ParseInt(arg[1], 10, 64)
	if err != nil {
		log.Fatalf("HanlerMy failed: %s", err)
		return "", err
	}
	var res []string
	idxUser := slices.IndexFunc(usersTasks, func(user TasksForUser) bool {
		return userID == user.ID
	})
	if idxUser != -1 {
		for _, taskID := range usersTasks[idxUser].ToDo {
			idxTask := slices.IndexFunc(tasks, func(task Task) bool {
				return taskID == task.ID
			})
			username := slices.IndexFunc(usersTasks, func(user TasksForUser) bool {
				return tasks[idxTask].Owner == user.ID
			})
			resp := TaskResp{Assignee: false, NotAssignee: false, CurrentUserAssign: false}
			resp.TaskID = tasks[idxTask].ID
			resp.Task = tasks[idxTask].Task
			resp.Owner = usersTasks[username].User
			if tasks[idxTask].User == userID {
				resp.CurrentUserAssign = true
			}

			buf := new(bytes.Buffer)
			err = tmpl.ExecuteTemplate(buf, "MyOwnerTemplate.txt", resp)
			if err != nil {
				log.Fatalf("HanlerMy failed: %s", err)
				return "", err
			}
			res = append(res, buf.String())
		}
	}
	return strings.Join(res, "\n"), nil
}

func HanlerOwner(arg ...string) (string, error) {
	userName := arg[0]
	userID, err := strconv.ParseInt(arg[1], 10, 64)
	if err != nil {
		log.Fatalf("HanlerOwner failed: %s", err)
		return "", err
	}
	var res []string
	idxUser := slices.IndexFunc(usersTasks, func(user TasksForUser) bool {
		return userID == user.ID
	})
	if idxUser != -1 {
		for _, taskID := range usersTasks[idxUser].Owner {
			idxTask := slices.IndexFunc(tasks, func(task Task) bool {
				return taskID == task.ID
			})
			resp := TaskResp{Assignee: false, NotAssignee: false, CurrentUserAssign: false}
			resp.TaskID = tasks[idxTask].ID
			resp.Task = tasks[idxTask].Task
			resp.Owner = userName

			if tasks[idxTask].User == -1 {
				resp.NotAssignee = true
			} else if tasks[idxTask].User == userID {
				resp.CurrentUserAssign = true
			}
			buf := new(bytes.Buffer)
			err = tmpl.ExecuteTemplate(buf, "MyOwnerTemplate.txt", resp)
			if err != nil {
				log.Fatalf("HanlerOwner failed: %s", err)
				return "", err
			}
			res = append(res, buf.String())
		}
	}
	return strings.Join(res, ""), nil
}

var Handler = map[string]func(...string) (string, error){
	"tasks":    HanlerTasks,
	"new":      HanlerNew,
	"assign":   HanlerAssign,
	"unassign": HanlerUnassign,
	"resolve":  HanlerResolve,
	"my":       HanlerMy,
	"owner":    HanlerOwner,
}

var bot *tgbotapi.BotAPI

var tmpl *template.Template

func startTaskBot(ctx context.Context) error {
	var err error
	bot, err = tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		log.Fatalf("NewBotAPI failed: %s", err)
	}
	bot.Debug = true
	fmt.Printf("Authorized on account %s\n", bot.Self.UserName)

	wh, err := tgbotapi.NewWebhook(WebhookURL)
	if err != nil {
		log.Fatalf("NewWebhook failed: %s", err)
	}

	_, err = bot.Request(wh)
	if err != nil {
		log.Fatalf("SetWebhook failed: %s", err)
	}

	updates := bot.ListenForWebhook("/")
	ctx.Value("")
	go func() {
		log.Fatalln("http err:", http.ListenAndServe(":8081", nil))
	}()
	fmt.Println("start listen :8081")

	tmpl = template.Must(template.ParseGlob("./templates/*"))

	for update := range updates {
		cmd := update.Message.Command()
		cmdArg := update.Message.CommandArguments()
		complexCmd := strings.Split(update.Message.Command(), "_")
		if len(complexCmd) == 2 {
			cmd = complexCmd[0]
			cmdArg = complexCmd[1]
		}

		var msg tgbotapi.MessageConfig
		var res string
		if handler, ok := Handler[cmd]; ok {
			res, err = handler(update.Message.From.UserName, strconv.FormatInt(update.Message.From.ID, 10), cmdArg)
			if err != nil {
				log.Fatalf("SetWebhook failed: %s", err)
			}
		} else {
			res = "Неизвестная команда"
		}
		log.Printf("upd: %#v\n", update)
		msg = tgbotapi.NewMessage(
			update.Message.Chat.ID,
			res,
		)
		_, err = bot.Send(msg)
		if err != nil {
			log.Fatalf("Response failed: %s", err)
			return err
		}
	}
	return nil
}

func main() {
	err := startTaskBot(context.Background())
	if err != nil {
		panic(err)
	}
}
