package main


type data struct{
	Id  int `orm:"id"`
	Name string `orm:"name"`
}


func test1(){
	var st []*data
	err := kit.Query("select * from names where id in (?,?,?) limit 3",17466,17468,17469).GetStrus( &st)
	fmt.Println( err )
	for _, v := range st{
		fmt.Println(v.Id,v.Name)
	}

	var st2 data
	err2 := kit.Query("select * from names where id in (?,?,?) limit 3",17466,17468,17469).GetStru( &st2)
	fmt.Println( err2 )
	fmt.Println( st2.Id, st2.Name )

	var st3 []int
	err3 := kit.Query("select id from names where id in (?,?,?) limit 3",17466,17468,17469).GetNs( &st3)
	fmt.Println( err3 )
	fmt.Println( st3 )


	var st4  int
	err4 := kit.Query("select count(*) from names where id in (?,?,?) limit 3",17466,17468,17469).GetN( &st4)
	fmt.Println( err4 )
	fmt.Println( st4 )

}