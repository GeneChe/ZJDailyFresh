package main

import (
	_ "dailyFresh/models"
	_ "dailyFresh/routers"
	"github.com/astaxie/beego"
)

func main() {
	beego.Run()
}

