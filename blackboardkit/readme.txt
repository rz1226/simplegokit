如何使用
典型的应用场景是：想记录一些日志，计时，计数等，可以在web的某个端口方便的查看最近的信息



//创建监控黑板 黑板一共存在三种kit， 分别是logkit counterkit timerkit 用于记录日志，计数器，计时器
bb = blackboardkit.NewBlockBorad()
//初始化logkit,多个参数对应多个日志分类
bb.InitLogKit("worker_panic", "worker_start", "worker_end", "ajust_num")
//初始化counterkit ，多个参数对应多个计数器
bb.InitCounterKit("worker_result_true", "worker_result_false")
//初始化timerkit,多个参数对应多个计时器
bb.InitTimerKit("dosomething","dosomething2")
//给初始化的kit加个说明，这个说明的作用是显示的时候可读性更高。如果不加也可以。
//三种方法分别添加三种kit说明
bb.SetLogReadme("worker_panic", "工作goroutine panic退出")
bb.SetLogReadme("worker_start", "启动工作goroutine")
bb.SetLogReadme("worker_end", "关闭工作goroutine")
bb.SetLogReadme("ajust_num", "调整工作goroutine数量")
bb.SetCounterReadme("worker_result_true", "工作goroutine返回true")
bb.SetCounterReadme("worker_result_false", "工作goroutine返回false")
bb.SetTimerReadme("dosomething","说明")
//设置名字
bb.SetName("给黑板一个名字")
//最终初始化完成后，调用ready说明就绪，这个就可以进入数据以及获取监控数据了
bb.Ready()


//如何监控
import "github.com/rz1226/simplegokit/monitorkit"
func main(){
    monitorkit.StartMonitorBB("9091","/look") // 用浏览器看9091/look端口查看监控数据
}



