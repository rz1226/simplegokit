package coroutinekit

import (
	"errors"
	"fmt"
	"github.com/rz1226/simplegokit/blackboardkit"
	"github.com/rz1226/simplegokit/httpkit"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/*
协程管理监控
实现goroutine运行情况监控等功能
适合常驻协程处理任务
不适合随时启动的短时间任务。

m := NewCoroutineKit()
m.Start(  "name", num, f(), panicRestart )


//如何知道协程退出，把函数包起来
func x(){
	f()
	//检测退出
}
还有要检测panic，如果panic可能也会退出

*/
const MAX_NUM = 100
const STATUS_INIT = 0
const STATUS_RUN = 1
const STATUS_OUT = 2
const STATUS_PANIC = 3

var defaultco *CoroutineKit

func init() {
	defaultco = newCoroutineKit()
}
func Start(name string, num int, f func(), panicRestart bool) error {
	return defaultco.start(name, num, f, panicRestart)
}
func Show() string {
	return defaultco.showAll()
}

type CoroutineKit struct {
	mu          *sync.Mutex
	nodes       []*Node                      //每一组相同的goroutine占用一个node  主要作用可以按照启动的顺序展示监控信息
	nodeNames   map[string]*Node             //保存所有名称，用来去重
	historyInfo *blackboardkit.BlackBoradKit //记录历史的信息，例如之前已经完全退出的goroutine to do
}

func newCoroutineKit() *CoroutineKit {
	ck := &CoroutineKit{}
	ck.nodes = make([]*Node, 0, 1000)
	ck.mu = &sync.Mutex{}
	ck.nodeNames = make(map[string]*Node)
	ck.historyInfo = blackboardkit.NewBlockBorad()
	ck.historyInfo.SetName("coroutinekit 历史信息")
	ck.historyInfo.Ready()
	return ck
}

//加入goroutine，1 名称，不要重复，重复会报错，2 启动多少个goroutine 3 执行函数  4 遇到panic后是否要重新启动
func (ck *CoroutineKit) start(name string, num int, f func(), panicRestart bool) error {
	ck.mu.Lock()
	defer ck.mu.Unlock()
	name = strings.TrimSpace(name)
	//检查是否有重复名称
	_, ok := ck.nodeNames[name]
	if ok {
		return errors.New("duplicated name")
	}
	node := newNode(ck, name, num, f, panicRestart)
	ck.nodes = append(ck.nodes, node)
	ck.nodeNames[name] = node
	node.start() //启动
	return nil
}

func (ck *CoroutineKit) showAll() string {
	str := ""

	ck.mu.Lock()
	defer ck.mu.Unlock()
	for _, node := range ck.nodes {
		str += node.showAll()
	}
	return str
}

type Node struct {
	name         string //coroutine名字, 如果没有名字可以填写""
	runnings     []*Routine
	f            func()
	panicRestart bool
	father       *CoroutineKit
	mu           *sync.Mutex
}

func newNode(father *CoroutineKit, name string, num int, f func(), panicRestart bool) *Node {
	if num <= 0 {
		num = 1
	}
	if num > MAX_NUM {
		num = MAX_NUM
	}
	n := &Node{}
	n.name = name
	n.f = f
	n.panicRestart = panicRestart
	n.father = father
	n.mu = &sync.Mutex{}
	n.runnings = make([]*Routine, num, num)
	for i := 0; i < num; i++ {
		p := &Routine{}
		p.name = name
		p.startTime = ""
		p.endTime = ""
		p.panicTime = ""
		p.status = STATUS_INIT
		p.panicTimes = 0
		p.mu = &sync.Mutex{}
		p.lastPanicInfo = ""
		n.runnings[i] = p
	}
	return n
}

func (n *Node) showAll() string {
	str := ""
	str1 := "正在运行的数量  :"
	str2 := "已经退出的数量  :"
	str3 := "已经panic的数量 :"
	str4 := "总数量          :"
	str5 := "panic历史数量   :"
	count1 := 0
	count2 := 0
	count3 := 0
	count4 := 0
	count5 := 0
	n.mu.Lock()
	defer n.mu.Unlock()
	for k, v := range n.runnings {
		str += "------->\nGoroutine序号：" + strconv.Itoa(k)
		readme, num1, num2, num3, num4, num5 := v.show()
		str += readme
		count1 += num1
		count2 += num2
		count3 += num3
		count4 += num4
		count5 += num5

	}
	return "------------------" + n.name + "---------------------------->>\n" +
		str4 + strconv.Itoa(count4) + "\n" +
		str1 + strconv.Itoa(count1) + "\n" +
		str2 + strconv.Itoa(count2) + "\n" +
		str3 + strconv.Itoa(count3) + "\n" +
		str5 + strconv.Itoa(count5) + "\n" +
		str
}

func (n *Node) start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	num := len(n.runnings)
	for i := 0; i < num; i++ {
		n.startOne(i)
	}
}
func (n *Node) startOne(goroutineNo int) {
	newf := func(no int) {
		defer func() {
			if co := recover(); co != nil {
				//检查panic
				str := fmt.Sprintln(co)
				strStackInfo := GetPrintStack()
				n.setPanic(no, str + strStackInfo)
			}
		}()
		//开始运行
		n.setRun(no)
		n.f()
		//检测退出
		n.setOut(no)
	}
	go newf(goroutineNo)
}

//发生panic的时候
func (n *Node) setPanic(no int, info string) {
	p := n.runnings[no]
	p.mu.Lock()
	defer p.mu.Unlock()
	atomic.AddUint64(&p.panicTimes, 1) //原子操作貌似是没有必要的
	p.lastPanicInfo = info
	p.status = STATUS_PANIC
	p.panicTime = time.Now().Format("2006-01-02 15:04:05")
	if n.panicRestart == true {
		time.Sleep(time.Millisecond * 100)
		n.startOne(no)
	}

}

//正常退出
func (n *Node) setOut(no int) {
	p := n.runnings[no]
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = STATUS_OUT
	p.endTime = time.Now().Format("2006-01-02 15:04:05")
}

//开始运行
func (n *Node) setRun(no int) {
	p := n.runnings[no]
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = STATUS_RUN
	p.startTime = time.Now().Format("2006-01-02 15:04:05")
	p.endTime = ""
}

type Routine struct {
	mu            *sync.Mutex
	name          string
	startTime     string
	endTime       string
	panicTime     string
	status        uint32 //0没有启动 1运行中 2退出 3panic
	panicTimes    uint64 //panic发生的次数
	lastPanicInfo string //最后一次panic的信息
}

//string信息收集  num1启动中的数量 num2已经退出的数量 num3已经panic的数量 num4 总数量  num5 历史panic数量
func (r *Routine) show() (string, int, int, int, int, int) {
	str := ""
	num1 := 0
	num2 := 0
	num3 := 0
	num4 := 0
	num5 := 0
	r.mu.Lock()
	defer r.mu.Unlock()
	str += "\nGoroutine名称:" + r.name + "\n"
	statusReadme := ""
	if r.status == STATUS_INIT {
		statusReadme = "未启动"
	} else if r.status == STATUS_RUN {
		statusReadme = "运行中"
		num1 = 1
	} else if r.status == STATUS_OUT {
		statusReadme = "已退出"
		num2 = 1
	} else if r.status == STATUS_PANIC {
		statusReadme = "已恐慌"
		num3 = 1
	}
	num4 = 1
	num5 = int(r.panicTimes)
	str += "状态     :" + statusReadme + "\n"
	str += "启动时间  :" + r.startTime + "\n"
	str += "退出时间  :" + r.endTime + "\n"
	str += "异常时间  :" + r.panicTime + "\n"
	str += "异常次数  :" + strconv.FormatUint(r.panicTimes, 10) + "\n"
	str += "最后异常信息:" + r.lastPanicInfo + "\n"

	return str, num1, num2, num3, num4, num5
}

/**********************************************监控***************************************************/
func StartMonitor(port string) {
	go httpkit.NewSimpleHttpServer().Add("/", httpShowAll).Start(port)
}

func httpShowAll(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("yes")
	r.ParseForm()
	str := defaultco.showAll()
	fmt.Fprintln(w, str)
}

func GetPrintStack() string{
	buf := debug.Stack()
	return fmt.Sprintf("==> %s\n", string(buf ))
}
