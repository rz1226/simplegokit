如何使用
典型的应用场景，项目需要一定数量的goroutine处理后台任务，同时想了解他们的运行情况，例如是否还在运行，是否panic了。
一组goroutine还有几个在运行等。


以上设置好一个全局的变量来控制本项目中所有的相关常驻goroutine

这是个任务，一般来说是个无限循环任务，不会主动退出。这个例子会退出，实际使用一般是无限循环

func s(){
	for i:=1;i<10000000;i++{
		time.Sleep( time.Millisecond*1)
		//fmt.Println("running2")
		if i%40 == 0{
			panic("some error oh no")
			return

		}
	}
}

func main(){

    //启动任务 1 名称  2 数量  3 函数名 4 panic是否重启
    //注意：不建议动态的添加任务，应该是项目开始运行添加固定数量的goroutine。否则只能在测试环境使用这个库。
    fmt.Println(coroutinekit.Start("正在 进行推送任务", 30, s , true ))

    //查看信息
    coroutinekit.Show()
}
