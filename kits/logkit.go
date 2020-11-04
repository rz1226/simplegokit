package kits

/*
var glogs  LogKit

func init() {
	glogs = NewLogKit("自定义log分类", "apilog")
}

补充：为什么加入ready状态，后续可能会加很多功能，每个都要考虑初始化的问题，否则会面临并发竞争的问题，ready状态是万能的模式
可以应对后续加入的无数扩展功能。如果仅仅依靠newxx函数这个初始初始化动作初始化一切，代码很难写,也很难保持向前兼容

*/

import (
	"bytes"
	"fmt"
	"strconv"
	"time"
)

const (
	sIZE       = 500
	LOGMAXSIZE = 1000
)

type LogKit struct {
	logs    map[string]*CircleQueue
	names   []string          //有了这个show的时候不用锁
	readmes map[string]string //名称的注释
	ready   *Ready            //是否就绪，就绪意味着可以开始放入数据，拿出数据的操作。意味着再也能进行初始化阶段的动作
}

func NewLogKit(lognames ...string) *LogKit {
	if len(lognames) > LOGMAXSIZE {
		panic("logkit logs 超过了 maxsize ")
	}

	lk := &LogKit{}
	lk.ready = NewReady()
	lk.readmes = make(map[string]string)
	lk.logs = make(map[string]*CircleQueue)
	for _, v := range lognames {
		lk.logs[v] = NewCircleQueue(sIZE)
	}
	lk.names = lognames
	return lk
}

func (lk *LogKit) Ready() {
	lk.ready.SetTrue()
}

func (lk *LogKit) SetReadme(name string, readme string) {
	if lk.ready.IsReady() == true {
		return
	}
	lk.readmes[name] = readme
}

func (lk *LogKit) Names() []string {
	return lk.names
}

//展示数据
func (lk *LogKit) Show(bbname string) string {
	if lk.ready.IsReady() == false {
		return bbname + "日志没有就绪"
	}
	logNames := lk.Names()
	str := ""
	for _, name := range logNames {
		readme, ok := lk.readmes[name]
		if !ok {
			readme = ""
		}
		str += "\n----------------------\n日志名称: " + name + "   \n" + "日志说明:" + readme + "\n"
		str += lk.FetchContents(name, 40)
	}
	return str
}

//显示最近的count条记录
func (lk *LogKit) FetchContents(logname string, count int) string {
	if lk.ready.IsReady() == false {
		return "日志没有就绪"
	}
	q, ok := lk.logs[logname]
	if !ok {
		return logname + " 没有这个LogKit队列\n"
	}
	values, newestId := q.GetSeveral(count)
	return formatFetchedLog(values, newestId)
}

//把日志信息放入，然后返回格式化后的字符串
func (lk *LogKit) PutContentsAndFormat(logname string, a ...interface{}) string {
	if lk.ready.IsReady() == false {
		return ""
	}
	if len(logname) == 0 {
		return ""
	}
	cq, ok := lk.logs[logname]
	if !ok {
		return ""
	}
	buffer := bytes.Buffer{}
	buffer.WriteString(logname)
	buffer.WriteString(" ")
	buffer.WriteString(time.Now().Format("2006-01-02 15:04:05"))
	buffer.WriteString(" ")
	buffer.WriteString(fmt.Sprintln(a...))
	logStr := buffer.String()
	cq.Put(logStr)
	return logStr
}

func formatFetchedLog(values []interface{}, id uint64) string {
	buffer := bytes.Buffer{}
	buffer.WriteString("序号: " + strconv.FormatUint(id, 10) + "\n")
	for _, v := range values {
		str, ok := v.(string)
		if ok {
			buffer.WriteString(str)
		}
	}
	return buffer.String()
}
