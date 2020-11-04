package kits

const MAXSIZE = 1000

type CounterKit struct {
	data    map[string]*Counter
	names   []string
	readmes map[string]string //名称的注释
	ready   *Ready            //是否就绪，就绪意味着可以开始放入数据，拿出数据的操作。意味着再也能进行初始化阶段的动作
}

func NewCounterKit(strs ...string) *CounterKit {
	if len(strs) > MAXSIZE {
		panic("counterkit size too big")
	}
	c := &CounterKit{}
	c.ready = NewReady()
	c.readmes = make(map[string]string)
	c.data = make(map[string]*Counter, 0)
	//限制资源

	for _, v := range strs {
		c.data[v] = NewCounter()
	}
	c.names = strs
	return c
}
func (c *CounterKit) Names() []string {
	return c.names
}

func (c *CounterKit) SetReadme(name string, readme string) {
	if c.ready.IsReady() == true {
		return
	}
	c.readmes[name] = readme
}

func (c *CounterKit) Ready() {
	c.ready.SetTrue()
}

func (c *CounterKit) Show(bbname string) string {
	if c.ready.IsReady() == false {
		return bbname + "计数器没有就绪"
	}
	counterNames := c.Names()
	str := ""
	for _, name := range counterNames {
		readme, ok := c.readmes[name]
		if !ok {
			readme = ""
		}
		str += "\n----------------------\n计数器名称:" + name + " : \n计数器信息:" + readme + "\n"
		str += c.Str(name)
		str += "\n"
	}
	return str
}

func (c *CounterKit) Inc(name string) {
	if c.ready.IsReady() == false {
		return
	}
	counter, ok := c.data[name]
	if !ok {
		return
	}
	counter.Add(1)
}
func (c *CounterKit) IncBy(name string, num int64) {
	if c.ready.IsReady() == false {
		return
	}
	counter, ok := c.data[name]
	if !ok {
		return
	}
	counter.Add(num)
}
func (c *CounterKit) Get(name string) int64 {
	if c.ready.IsReady() == false {
		return 0
	}
	counter, ok := c.data[name]
	if !ok {
		return 0
	}
	return counter.Get()
}

func (c *CounterKit) Str(name string) string {
	if c.ready.IsReady() == false {
		return "countkit not ready"
	}
	counter, ok := c.data[name]
	if !ok {
		return "找不到计数器:" + name + "\n"
	}
	return counter.Str()
}
