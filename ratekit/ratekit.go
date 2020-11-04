package ratekit

import (
	"fmt"
	"github.com/rz1226/simplegokit/blackboardkit"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

/*
	这个库的功能是以特定的速度异步执行批量闭包函数，并可以设置每个失败重试的次数，和一些全局信息例如总失败次数，扔掉
	的任务次数（用完所有重试次数也没有成功的任务函数）。
*/

const RATEKITCHANSIZE = 10000             //chan长度
const DEFAULT_TASK_NUM_EVERY_SECOND = 100 //默认每秒跑多少个任务
const MAX_TASK_NUM_EVERY_SECOND = 10000   //最大速度

type Task struct {
	f            func() bool //任务函数，要符合这个格式，实际应用中一般来说是个闭包
	leftTryCount uint32      //一共可以执行的次数
}

type RateKit struct {
	currentCount        uint32 //最近一秒钟的执行次数
	currentRealCount    uint32 //实际执行次数，就是上个值去掉超限的
	limitCountPerSecond uint32 //每秒最大执行次数
	asynChan            chan Task
	factory             *Factory
	wait                *sync.WaitGroup              //用来控制其他go程任务真正处理数据的时间点
	bb                  *blackboardkit.BlackBoradKit //记录日志信息用的黑板
	rs                  *RecentSum                   //最近几次flush的时候队列中的数据的sum
}

//limitcount 每秒钟运行次数上限制， workernum工作go程数量， issync是否是同步模式
func NewRateKit(limitCount uint32) *RateKit {
	rk := &RateKit{}
	rk.wait = &sync.WaitGroup{}
	rk.wait.Add(1)
	rk.currentCount = 0
	rk.currentRealCount = 0
	if limitCount <= 0 {
		limitCount = DEFAULT_TASK_NUM_EVERY_SECOND //设置了个默认值
	}
	if limitCount > MAX_TASK_NUM_EVERY_SECOND {
		limitCount = MAX_TASK_NUM_EVERY_SECOND
	}
	rk.limitCountPerSecond = limitCount
	rk.asynChan = make(chan Task, RATEKITCHANSIZE)
	rk.bb = blackboardkit.NewBlockBorad()
	rk.bb.InitCounterKit("count_all_return_true", "count_all_return_false_retry",
		"count_all_return_false_throw", "count_all_rate_limited_retry",
		"count_all_return_false_throw_default")
	rk.bb.SetCounterReadme("count_all_return_true", "函数返回true的总数")
	rk.bb.SetCounterReadme("count_all_return_false_retry", "函数返回false并尝试重试总数")
	rk.bb.SetCounterReadme("count_all_return_false_throw", "函数返回false并丢弃了总数")
	rk.bb.SetCounterReadme("count_all_rate_limited_retry", "函数执行因限速被否定并重试总数")
	rk.bb.SetCounterReadme("count_all_return_false_throw_default", "函数执行返回false无法放入队列而丢弃总数")
	rk.bb.InitLogKit("error", "info", "put")
	rk.bb.SetLogReadme("error", "错误日志")
	rk.bb.SetLogReadme("info", "限速器运行情况每秒监控日志信息")
	rk.bb.SetLogReadme("put", "闭包函数进入限速器的流水日志")
	rk.bb.SetNoPrintToConsole(true)
	rk.bb.SetName("ratekit限速器")
	rk.bb.Ready()

	//启动异步任务go程
	rk.factory = newFactory(rk.asynTask, rk.wait)
	rk.factory.ajust(10)
	rk.rs = NewRecentSum(5) //记录最近5次的chan长度的sum
	go rk.flush()
	rk.wait.Done()
	return rk
}

//every一秒钟计数清零,以及动态调整worker数量
//lock free
func (rk *RateKit) flush() {
	defer func() {
		if co := recover(); co != nil {
			fmt.Println(co)
			rk.bb.Log("error", "flush goroutine exit", co)
			time.Sleep(time.Millisecond * 100)
			rk.flush()
		}
	}()
	rk.wait.Wait()
	for {
		time.Sleep(time.Second * 1)
		//除了定时清零计数器，在这里加入动态调整worker数量
		currentCount := atomic.LoadUint32(&rk.currentCount)
		limitCountPerSecond := atomic.LoadUint32(&rk.limitCountPerSecond)
		if currentCount < limitCountPerSecond {
			//加更多的worker
			lenChan := len(rk.asynChan)
			rk.rs.Put(lenChan)
			if lenChan > 10 {
				runningNum := atomic.LoadUint32(&rk.factory.runningNum)
				add := (WORKERNUM-runningNum)/20 + 5
				rk.factory.ajust(runningNum + add)
			} else {
				//最近几次的sum都是0
				if rk.rs.Sum() == 0 {

					//-worker的数量
					runningNum := atomic.LoadUint32(&rk.factory.runningNum)
					if runningNum > 3 {
						rk.factory.ajust(runningNum - 2)
					}
				}
			}

		} else if currentCount > limitCountPerSecond {
			reduce := rk.factory.runningNum/30 + 2
			//-worker的数量
			runningNum := atomic.LoadUint32(&rk.factory.runningNum)
			rk.factory.ajust(runningNum - reduce)
		} else {
			//不变
		}
		rk.collectInfo()
		atomic.StoreUint32(&rk.currentCount, 0)
		atomic.StoreUint32(&rk.currentRealCount, 0)

	}

}

//第二个参数是包括第一次运行和重试的总次数
//如果chan满了，这个函数会阻塞
//这个函数的基本功能是把函数包装后放入队列
//lock free
//需要控制入队列的速度，否则队列满的话，会阻塞大批worker在任务未完成重新放入队列的时候
func (rk *RateKit) Go(f func() bool, leftTryCount uint32) {
	rk.bb.Log("put", "数据进入!")
	duration := uint32(0)
	if len(rk.asynChan) > RATEKITCHANSIZE/2 {
		limitCountPerSecond := atomic.LoadUint32(&rk.limitCountPerSecond)
		duration = (uint32(1000*1000) / limitCountPerSecond) * 2

	} else if len(rk.asynChan) > (RATEKITCHANSIZE/2 + RATEKITCHANSIZE/4) {
		limitCountPerSecond := atomic.LoadUint32(&rk.limitCountPerSecond)
		duration = (uint32(1000*1000) / limitCountPerSecond) * 4
	}

	time.Sleep(time.Microsecond * time.Duration(duration))

	if leftTryCount == 0 {
		leftTryCount = 1
	}
	if leftTryCount > 10 {
		leftTryCount = 10
	}
	rf := Task{}
	rf.f = f
	rf.leftTryCount = leftTryCount
	rk.asynChan <- rf
}

//这个就是要送给worker执行任务函数
//这个函数的基本任务是把任务从队列拿出来然后执行
func (rk *RateKit) asynTask() bool {
	rf := <-rk.asynChan
	f := rf.f
	leftTryCount := rf.leftTryCount
	atomic.AddUint32(&rk.currentCount, 1)
	//注意，这里无法保证百分之百的正确。误差可以忍受
	currentCount := atomic.LoadUint32(&rk.currentCount)
	limitCountPerSecond := atomic.LoadUint32(&rk.limitCountPerSecond)

	if currentCount <= limitCountPerSecond {
		//执行
		atomic.AddUint32(&rk.currentRealCount, 1)
		res := f()
		leftTryCount--
		if res == true {
			//运行成功
			rk.bb.Inc("count_all_return_true")
			return true
		} else {
			//一个微小的停顿，这里主要针对本库的应用场景是异步调用接口等
			time.Sleep(time.Millisecond * time.Duration(getRandInt(3)))
			if leftTryCount >= 1 {
				//因为错误重新放入队列
				rf := Task{}
				rf.f = f
				rf.leftTryCount = leftTryCount
				//如果队列满了，这里要可以放弃一些数据,因为如果这里不放弃，可能会在极端情况下全部worker阻塞
				select {
				case rk.asynChan <- rf:
					rk.bb.Inc("count_all_return_false_retry")
				default:
					rk.bb.Inc("count_all_return_false_throw_default")
				}

			} else {
				//重试次数用光，丢弃
				rk.bb.Inc("count_all_return_false_throw")
			}
			return false
		}
	} else {
		//速度限制,现在运行不了,放入队列
		//这里要停一下，否则会快速的放入队列很厉害的消耗cpu
		time.Sleep(time.Millisecond * time.Duration(getRandInt(150)))

		rf := Task{}
		rf.f = f
		rf.leftTryCount = leftTryCount
		rk.asynChan <- rf
		rk.bb.Inc("count_all_rate_limited_retry")
		return false
	}
	return true
}

func (rk *RateKit) collectInfo() {
	str := ""
	str += "worker:" + strconv.FormatUint(uint64(atomic.LoadUint32(&rk.factory.runningNum)), 10) + "\n"
	str += "前一秒钟处理数量:" + strconv.FormatUint(uint64(atomic.LoadUint32(&rk.currentCount)), 10) + "\n"
	str += "前一秒钟处理实际数量:" + strconv.FormatUint(uint64(atomic.LoadUint32(&rk.currentRealCount)), 10) + "\n"
	str += "限速:" + strconv.FormatUint(uint64(atomic.LoadUint32(&rk.limitCountPerSecond)), 10) + "\n"
	str += "队列长度:" + strconv.Itoa(len(rk.asynChan)) + "\n"

	rk.bb.Log("info", str)
}

func (rk *RateKit) Show() string {

	str := ""
	str += "------------------------------------------------------------------------------------------------"
	str += rk.bb.Show()
	str += "------------------------------------------------------------------------------------------------"
	str += rk.factory.bb.Show()
	return str
}

/***************************************worker*******************************/
const STATUS_WORKER_RUNNING = 1 //正在运行
const STATUS_WORKER_NOT_RUNNING = 0
const WORKERNUM = 1000

type Factory struct {
	workerList []*Worker
	runningNum uint32 //正在运行的worker数量
	wait       *sync.WaitGroup
	bb         *blackboardkit.BlackBoradKit //记录日志信息用的黑板
}

//第一个参数是worker工作的函数
func newFactory(f func() bool, wait *sync.WaitGroup) *Factory {
	fac := &Factory{}
	if WORKERNUM <= 0 {
		fmt.Println("不能设置WORKERNUM小于等于零")
		os.Exit(1)
	}
	fac.workerList = make([]*Worker, WORKERNUM, WORKERNUM)
	for i := 0; i < WORKERNUM; i++ {
		worker := &Worker{}
		worker.iterF = f
		worker.running = STATUS_WORKER_NOT_RUNNING
		worker.factory = fac
		fac.workerList[i] = worker
	}
	fac.runningNum = 0
	fac.wait = wait
	fac.bb = blackboardkit.NewBlockBorad()
	fac.bb.InitLogKit("worker_panic", "worker_start", "worker_end", "ajust_num")
	fac.bb.SetLogReadme("worker_panic", "工作goroutine panic退出")
	fac.bb.SetLogReadme("worker_start", "启动工作goroutine")
	fac.bb.SetLogReadme("worker_end", "关闭工作goroutine")
	fac.bb.SetLogReadme("ajust_num", "调整工作goroutine数量")
	fac.bb.InitCounterKit("worker_result_true", "worker_result_false")
	fac.bb.SetCounterReadme("worker_result_true", "工作goroutine返回true")
	fac.bb.SetCounterReadme("worker_result_false", "工作goroutine返回false")
	fac.bb.SetNoPrintToConsole(true)
	fac.bb.SetName("ratekit速度控制器里面的worker信息日志")
	fac.bb.Ready()
	return fac
}

//调整worker启动数量为num
func (fac *Factory) ajust(num uint32) {
	if num <= 0 {
		num = 1
	}
	if num >= WORKERNUM {
		num = WORKERNUM
	}
	runningNum := atomic.LoadUint32(&fac.runningNum)
	if num > runningNum {
		//需要增加
		var i uint32 = 0
		for i = 0; i < num-runningNum; i++ {
			fac.addOneRunning()
		}
	} else {
		var i uint32 = 0
		for i = 0; i < runningNum-num; i++ {
			fac.reduceOneRunning()
		}
	}
}

func (fac *Factory) addOneRunning() {
	if fac.runningNum == WORKERNUM {
		return
	}
	rand.Seed(time.Now().Unix())
	r := rand.Intn(10)
	//随机
	if r > 5 {
		//找到第一个没有运行的，并启动
		for i := 0; i < WORKERNUM; i++ {
			if atomic.CompareAndSwapInt32(&(fac.workerList[i].running), STATUS_WORKER_NOT_RUNNING, STATUS_WORKER_RUNNING) {
				//这里需要注意的是，检测到状态为stop，是无法保证go程真的stop了的
				// 特别是任务函数执行时间较长，有发生一个worker运行多个go程的危险
				//所以最好能避开这个worker是刚刚关闭的worker,目前的处理是不用太担心，关闭的时候会关掉多个
				//目前的处理是变换遍历顺序，以降低发生概率
				fac.bb.Log("worker_start", "启动 worker")
				go fac.workerList[i].run() //启动
				atomic.AddUint32(&fac.runningNum, 1)
				fac.bb.Log("ajust_num", "调整worker数量,增加一个worker,正序")
				break
			}
		}
	} else {
		for i := WORKERNUM - 1; i >= 0; i-- {
			if atomic.CompareAndSwapInt32(&(fac.workerList[i].running), STATUS_WORKER_NOT_RUNNING, STATUS_WORKER_RUNNING) {
				fac.bb.Log("worker_start", "启动 worker")
				go fac.workerList[i].run() //启动
				atomic.AddUint32(&fac.runningNum, 1)
				fac.bb.Log("ajust_num", "调整worker数量,增加一个worker,反序")
				break
			}
		}
	}
}
func (fac *Factory) reduceOneRunning() {
	if fac.runningNum == 0 {
		return
	}
	//找到第一个运行的，并关闭
	for i := 0; i < WORKERNUM; i++ {
		if atomic.CompareAndSwapInt32(&fac.workerList[i].running, STATUS_WORKER_RUNNING, STATUS_WORKER_NOT_RUNNING) {
			atomic.AddUint32(&fac.runningNum, ^uint32(0))
			fac.bb.Log("ajust_num", "调整worker数量,关闭一个worker")
			break
		}
	}
}

type Worker struct {
	running int32       //0关闭  1打开
	iterF   func() bool //这里面不必要再加个无限循环，仅仅写业务逻辑就可以了,注意这是一个可以重复多次执行的函数，里面类似从某个chan拿出
	//数据并且处理
	factory *Factory //所属的车间
}

func (w *Worker) run() {
	defer func() {
		if co := recover(); co != nil {
			w.factory.bb.Log("worker_panic", "worker 发生异常:", co)
			fmt.Println("panic", co)
			time.Sleep(time.Millisecond * time.Duration(getRandInt(200))) //如果挂了，等一点点时间再重启，防止无限挂跑死cpu
			w.run()
		}
	}()
	w.factory.wait.Wait()
	for {
		result := w.iterF()
		if result == true {
			w.factory.bb.Inc("worker_result_true")
		} else {
			w.factory.bb.Inc("worker_result_false")
		}
		if atomic.CompareAndSwapInt32(&w.running, STATUS_WORKER_NOT_RUNNING, STATUS_WORKER_NOT_RUNNING) {
			//状态为停止，关闭worker
			w.factory.bb.Log("worker_end", "关闭 worker")
			return
		}
	}
}

func getRandInt(n int) int {
	rand.Seed(time.Now().Unix())
	r := rand.Intn(n)
	return r
}
