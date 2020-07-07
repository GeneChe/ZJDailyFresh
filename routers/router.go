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
	beego.Router("/goodsDetail", &controllers.GoodsController{}, "get:ShowGoodsDetail")
	beego.Router("/goodsList", &controllers.GoodsController{}, "get:ShowGoodsList")
	beego.Router("/goodsSearch", &controllers.GoodsController{}, "post:HandleSearch")

    // 购物车模块
    // 增加到购物车请求: 当未登录时, 设置了user前缀, 请求会被过滤器拦截, 返回一个login视图,
    // 		--- 而ajax是后台请求不会进入login视图, 且ajax发送和接收的都是json格式数据.
    // 更新购物车请求: 设置前缀user是因为更新操作是在登录后才能看到的页面触发的ajax操作, 不存在路由过滤问题
	beego.Router("/user/addCart", &controllers.CartController{}, "post:HandleAddCart")
	beego.Router("/user/userCart", &controllers.CartController{}, "get:ShowUserCart")
	beego.Router("/user/updateCart", &controllers.CartController{}, "post:HandleUpdateCart")
	beego.Router("/user/deleteCart", &controllers.CartController{}, "post:HandleDeleteCart")

    // 订单模块
    beego.Router("/user/placeOrder", &controllers.OrderController{}, "post:HandlePlaceOrder")
	beego.Router("/user/createOrder", &controllers.OrderController{}, "post:HandleCreateOrder")
	beego.Router("/user/pay", &controllers.OrderController{}, "get:ShowPay")
	beego.Router("/user/paySyncResult", &controllers.OrderController{}, "get:ShowPaySyncResult")

    // 后台模块 -- 项目dailyFreshManage
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