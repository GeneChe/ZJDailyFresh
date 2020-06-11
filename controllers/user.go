package controllers

import "github.com/astaxie/beego"

type UserController struct {
	beego.Controller
}

func (u *UserController) ShowRegister() {
	u.TplName = "register.html"
}

