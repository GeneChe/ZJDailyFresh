package routers

import (
	"dailyFresh/controllers"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

func init() {
    // beego.Router("/", &controllers.MainController{})
	beego.InsertFilter("/user/*", beego.BeforeExec, filterFunc)

    // 用户模块
	// 1. 登录注册
	beego.Router("/register", &controllers.UserController{}, "get:ShowRegister;post:HandleRegister")
    beego.Router("/active", &controllers.UserController{}, "get:ActiveUser")
    beego.Router("/login", &controllers.UserController{}, "get:ShowLogin;post:HandleLogin")
	// 严谨的写法: 只有是已登录的状态才让logout, 所以这里使用/user/logout 路径
	beego.Router("/user/logout", &controllers.UserController{}, "get:Logout")
	// 2. 用户中心
	beego.Router("/user/usercenter", &controllers.UserController{}, "get:ShowUserInfo")
	beego.Router("/user/userorder", &controllers.UserController{}, "get:ShowUserOrder")
	beego.Router("/user/useraddress", &controllers.UserController{}, "get:ShowUserAddr;post:HandleUserAddr")

    // 商品模块
    beego.Router("/", &controllers.GoodsController{}, "get:ShowHomePage")

    // 购物车模块

    // 订单模块

    // 后台模块
}

func filterFunc(c *context.Context) {
	// Handler crashed with error runtime error: invalid memory address or nil pointer dereference
	// 一般是没配置app.conf中sessionon字段
	userInfo := c.Input.Session("userInfo")
	if userInfo == nil {
		c.Redirect(302, "/login")
		return
	}
}