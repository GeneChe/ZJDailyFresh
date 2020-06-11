package routers

import (
	"dailyFresh/controllers"
	"github.com/astaxie/beego"
)

func init() {
    beego.Router("/", &controllers.MainController{})
	// 注册登录
	beego.Router("/register", &controllers.UserController{}, "get:ShowRegister")
}
