package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/gomodule/redigo/redis"
)

type MainController struct {
	beego.Controller
}

func (c *MainController) Get() {
	c.Data["Website"] = "beego.me"
	c.Data["Email"] = "astaxie@gmail.com"
	c.TplName = "index.tpl"
}

// 返回redis链接接口
func GetRedisConn() redis.Conn {
	dialOp := redis.DialPassword("zj2fighting")
	conn, err := redis.Dial("tcp", "192.168.3.99:6379", dialOp)
	if err != nil {
		logs.Error("connect to redis err", err)
		return nil
	}
	return conn
}