httpclient用法
典型的应用场景，访问一个http url

var client *httpkit.HttpClient

func init(){
    1 超时时间秒 2 最大空闲连接池容量
	client = httpkit.NewHttpClient(1,100)
}
以上初始化一个全局的httpclient

执行post
url := "http://127.0.0.1:9030"
var buf io.Reader
buf = strings.NewReader("abc=sdfsdfsdfsdfsdf&t=234")
str, err := client.Post(url, "application/x-www-form-urlencoded;charset=utf-8", buf )

执行get
url := "http://127.0.0.1:9030"
str, err := client.Get(url )


//启动监控
monitorkit.StartMonitorBB("9092","/look")
