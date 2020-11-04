package blackboardkit

import (
	"fmt"
	"github.com/rz1226/simplegokit/kits"

	"sync"
	"time"
)

var allbb *AllBB

func init() {
	allbb = sNewAllBB()

}
func Show() string {
	return allbb.show()
}

//所有的bb
type AllBB struct {
	data []*BlackBoradKit
	mu   *sync.Mutex
}

func sNewAllBB() *AllBB {
	a := &AllBB{}
	a.mu = &sync.Mutex{}
	a.data = make([]*BlackBoradKit, 0)
	return a
}
func (a *AllBB) add(bb *BlackBoradKit) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.data = append(a.data, bb)
}
func (a *AllBB) show() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	str := ""
	for _, v := range a.data {
		str += v.Show()
	}
	return str
}

//监控信息黑板

type BlackBoradKit struct {
	logKit           *kits.LogKit
	timerKit         *kits.TimerKit
	counterKit       *kits.CounterKit
	serverStartTime  string
	noPrintToConsole bool
	name             string
	ready            *kits.Ready
}

func NewBlockBorad() *BlackBoradKit {
	bb := &BlackBoradKit{}
	bb.ready = kits.NewReady()
	bb.serverStartTime = time.Now().Format("2006-01-02 15:04:05")
	bb.InitLogKit("default", "info", "error")
	bb.InitCounterKit("default")
	bb.InitTimerKit("default")
	bb.SetLogReadme("default", "默认日志记录")
	bb.SetLogReadme("info", "默认info日志")
	bb.SetLogReadme("error", "默认error日志")
	bb.SetCounterReadme("default", "默认计数器")
	bb.SetTimerReadme("default", "默认计时器")
	bb.noPrintToConsole = true //默认不直接打印信息
	bb.name = ""
	allbb.add(bb)
	return bb
}

//设置好的bb一定要运行这个进行就绪
func (bb *BlackBoradKit) Ready() {
	bb.logKit.Ready()
	bb.counterKit.Ready()
	bb.timerKit.Ready()
	bb.ready.SetTrue()
}

//是否同时打印到标准输出
func (bb *BlackBoradKit) SetNoPrintToConsole(result bool) {
	if bb.ready.IsReady() == true {
		return
	}
	bb.noPrintToConsole = result
}

//给bb名字
func (bb *BlackBoradKit) SetName(name string) {
	if bb.ready.IsReady() == true {
		return
	}
	bb.name = name
}

//给日志kit加说明
func (bb *BlackBoradKit) SetLogReadme(name string, readme string) {
	if bb.ready.IsReady() == true {
		return
	}
	bb.logKit.SetReadme(name, readme)
}

//给计数器kit加说明
func (bb *BlackBoradKit) SetCounterReadme(name string, readme string) {
	if bb.ready.IsReady() == true {
		return
	}
	bb.counterKit.SetReadme(name, readme)
}

//给计时器kit加说明
func (bb *BlackBoradKit) SetTimerReadme(name string, readme string) {
	if bb.ready.IsReady() == true {
		return
	}
	bb.timerKit.SetReadme(name, readme)
}

//初始化日志kit
func (bb *BlackBoradKit) InitLogKit(names ...string) {
	if bb.ready.IsReady() == true {
		return
	}
	bb.logKit = kits.NewLogKit(names...)
}

//初始化计数器kit
func (bb *BlackBoradKit) InitCounterKit(names ...string) {
	if bb.ready.IsReady() == true {
		return
	}
	bb.counterKit = kits.NewCounterKit(names...)
}

//初始化计时器kit
func (bb *BlackBoradKit) InitTimerKit(names ...string) {
	if bb.ready.IsReady() == true {
		return
	}
	bb.timerKit = kits.NewTimerKit(names...)
}

/*----------------------------log--------------------------------*/
func (bb *BlackBoradKit) Log(logName string, logs ...interface{}) {
	str := bb.logKit.PutContentsAndFormat(logName, logs...)
	if bb.noPrintToConsole == false {
		fmt.Print(str)
	}
}

/*---------------------------timer---------------------------------*/
func (bb *BlackBoradKit) Start(timerName, tickInfo string) *kits.Tick {
	return bb.timerKit.Start(timerName, tickInfo)
}
func (bb *BlackBoradKit) End(tick *kits.Tick) {
	bb.timerKit.End(tick)
}

/*---------------------------counter---------------------------------*/
func (bb *BlackBoradKit) Inc(name string) {
	bb.counterKit.Inc(name)
}
func (bb *BlackBoradKit) IncBy(name string, num int64) {
	bb.counterKit.IncBy(name, num)
}

/*--------------------------show---------------------------*/
//获取监控信息
func (bb *BlackBoradKit) Show() string {
	if bb.ready.IsReady() == false {
		return "blackboard没有就绪"
	}

	str := "\n\n\n----------------" + bb.name + " blackboard info ----------------- : \n\n\n"

	str += "服务器启动时间:" + bb.serverStartTime + "\n"
	str += bb.logKit.Show(bb.name)

	str += "\n\n\n"
	str += bb.counterKit.Show(bb.name)

	str += "\n\n\n"
	str += bb.timerKit.Show(bb.name)
	return str
}
