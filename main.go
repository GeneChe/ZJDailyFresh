package main

import (
	_ "dailyFresh/models"
	_ "dailyFresh/routers"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

func main() {
	// 可以输出orm对应sql
	orm.Debug = true
	// runtime.GOOS 获取当前系统
	beego.Run()
}

