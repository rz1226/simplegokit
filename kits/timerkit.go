package kits

//时间记录
import (
	"strconv"
	"sync/atomic"
	"time"
)

/*
		tick := tk.Start("a","调用abc接口")
		tick2 := tk.Start("b","调用abc接口1")
		tick3 := tk.Start("c","调用abc接口2")
		time.Sleep(time.Millisecond*10)
		tk.End( tick)
		time.Sleep(time.Millisecond*10)
		tk.End(tick2)
		time.Sleep(time.Millisecond*10)
		tk.End(tick3)
//记录最近的时间比较长的操作
*/
const (
	sLOWLOGNAME   = "default"
	lASTLOGNAME   = "last"
	sLOWNANO      = 10000000 //耗时大于这个+平均数算作慢 nano单位
	TIMEKITMAXKIT = 1000
)

type TimerKit struct {
	kits    map[string]*timerKitNode
	names   []string
	readmes map[string]string //名称的注释
	ready   *Ready            //是否就绪，就绪意味着可以开始放入数据，拿出数据的操作。意味着再也能进行初始化阶段的动作
}

func NewTimerKit(names ...string) *TimerKit {
	if len(names) > TIMEKITMAXKIT {
		panic("timerkit too big")
	}
	tk := &TimerKit{}
	tk.ready = NewReady()
	tk.readmes = make(map[string]string)
	tk.kits = make(map[string]*timerKitNode)
	for _, v := range names {
		tk.kits[v] = newTimerKitNode(v)
	}
	tk.names = names
	return tk
}
func (tk *TimerKit) Names() []string {
	return tk.names
}

func (tk *TimerKit) Ready() {
	tk.ready.SetTrue()
}

func (tk *TimerKit) SetReadme(name string, readme string) {
	if tk.ready.IsReady() == true {
		return
	}
	tk.readmes[name] = readme
}

func (tk *TimerKit) Show(bbname string) string {
	if tk.ready.IsReady() == false {
		return bbname + "计时器没有就绪"
	}
	timerNames := tk.Names()
	str := ""
	for _, name := range timerNames {
		readme, ok := tk.readmes[name]
		if !ok {
			readme = ""
		}
		str += "----------------------\n计时器名称:" + name + "\n计时器说明:" + readme + "\n"
		str += tk.Info(name)
	}
	return str
}

func (tk *TimerKit) Start(timerName, tickInfo string) *Tick {
	if tk.ready.IsReady() == false {
		return nil
	}
	timerKitNode, ok := tk.kits[timerName]
	if !ok {
		return nil
	}
	return timerKitNode.start(tickInfo)
}
func (tk *TimerKit) End(tick *Tick) {
	if tk.ready.IsReady() == false {
		return
	}
	tickName := tick.timerName
	timerKitNode, ok := tk.kits[tickName]
	if !ok {
		return
	}
	if tick == nil {
		return
	}
	timerKitNode.end(tick)
}
func (tk *TimerKit) Info(timerName string) string {
	if tk.ready.IsReady() == false {
		return ""
	}
	timerKitNode, ok := tk.kits[timerName]
	if !ok {
		return ""
	}
	//这里不用考虑组装字符串的性能，因为没必要， 而且数据很小的常见，+的低性能劣势并不明显
	resStr := ""
	resStr += "count:" + strconv.FormatInt(timerKitNode.getCount(), 10) + "次\n"
	resStr += "sum:" + strconv.FormatFloat(timerKitNode.getSum(), 'f', 6, 64) + "秒\n"
	resStr += "avg:" + strconv.FormatFloat(timerKitNode.getAvg(), 'f', 6, 64) + "秒每次\n"
	resStr += "高耗时记录: \n" + timerKitNode.showslow()
	resStr += "最近记录: \n" + timerKitNode.showlast()
	return resStr
}

type timerKitNode struct {
	name    string  //计时器的名字
	count   int64   //经过了多少次计数
	sum     int64   //总计时时间
	slowest *LogKit //该计时器的慢操作列表
	last    *LogKit
}

//单次计时操作
type Tick struct {
	timerName string
	info      string
	startTime int64
}

func newTimerKitNode(name string) *timerKitNode {
	tkn := &timerKitNode{}
	tkn.name = name
	tkn.count = 0
	tkn.sum = 0
	tkn.slowest = NewLogKit(sLOWLOGNAME)
	tkn.slowest.Ready()
	tkn.last = NewLogKit(lASTLOGNAME)
	tkn.last.Ready()
	return tkn
}

func (t *timerKitNode) start(tickInfo string) *Tick {
	te := &Tick{}
	te.info = tickInfo
	te.timerName = t.name
	te.startTime = time.Now().UnixNano()
	return te
}

func (t *timerKitNode) end(tick *Tick) {
	du := time.Now().UnixNano() - tick.startTime
	atomic.AddInt64(&t.count, 1)
	atomic.AddInt64(&t.sum, du)
	count := atomic.LoadInt64(&t.count)
	t.last.PutContentsAndFormat(lASTLOGNAME, "操作是:"+tick.info+" 耗时秒是:", float64(du)/float64(time.Second))
	sum := atomic.LoadInt64(&t.sum)
	if du > sLOWNANO+sum/(count+1) {
		t.slowest.PutContentsAndFormat(sLOWLOGNAME, "操作是:"+tick.info+" 耗时秒:", float64(du)/float64(time.Second))
	}
}
func (t *timerKitNode) getCount() int64 {
	return atomic.LoadInt64(&t.count)
}

func (t *timerKitNode) getSum() float64 {
	return float64(atomic.LoadInt64(&t.sum)) / float64(time.Second)
}
func (t *timerKitNode) getAvg() float64 {
	sum := atomic.LoadInt64(&t.sum)
	count := atomic.LoadInt64(&t.count)
	if count == 0 {
		return 0
	}
	avg := sum / count
	return float64(avg) / float64(time.Second)
}
func (t *timerKitNode) showslow() string {
	return t.slowest.FetchContents(sLOWLOGNAME, 30)
}
func (t *timerKitNode) showlast() string {
	return t.last.FetchContents(lASTLOGNAME, 30)
}
