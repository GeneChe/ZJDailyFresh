package controllers

import "github.com/astaxie/beego"

type GoodsController struct {
	beego.Controller
}

func (g *GoodsController) ShowHomePage() {
	GetUserInfo(&g.Controller)
	g.TplName = "index.html"
}

