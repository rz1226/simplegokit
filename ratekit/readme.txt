如何使用
注意，不适用于执行时间非常久的函数执行，会造成限速不准。
典型的应用场景，例如用固定的速度，如1000个每秒的速度访问某接口推送数据，或者获取数据。

var rk *ratekit.RateKit

func init(){
	rk   = ratekit.NewRateKit(300     ) //参数表示每秒钟最大执行次数
}
初始化限速器，

func testx(){
    for i:=0;i<10000;i++{
		f := func(a int )func()bool{
			return func()bool{
                //fmt.Println("闭包", a   )
				time.Sleep(time.Millisecond*30)
				return false
			}

		}(i )
        //f 是一个闭包。因为要适应 func()bool格式

		time.Sleep(time.Microsecond*20)
		//进入限速器  , 1 函数  2 最大执行次数，一般如果第一次成功就执行一次，如果失败重试
		rk.Go( f, 3   )

	}

}


