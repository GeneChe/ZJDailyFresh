package controllers

import (
	"dailyFresh/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"strconv"
)

type CartController struct {
	beego.Controller
}

// 处理添加购物车请求
// 该请求是通过ajax发出, ajax是页面的后台请求
// 给ajax请求返回视图时, 发请求的页面并不会加载这个视图
// 		-- 如这里 用户未登录时, filter要跳转login页给ajax, 发送ajax这个页面并不会跳转到login页, 而是无变化
// 所以ajax一般用于页面的局部刷新
func (c *CartController) HandleAddCart() {
	// 获取数据
	skuId, err1 := c.GetInt("skuId")
	number, err2 := c.GetInt("number")
	// 校验数据
	if err1 != nil || err2 != nil {
		Response(&c.Controller, "1001", "请求参数错误", nil)
		return
	}

	userInfo := GetUserInfo(&c.Controller)
	if userInfo == nil {
		Response(&c.Controller, "1002", "获取用户信息失败", nil)
		return
	}

	// 处理数据
	conn := GetRedisConn()
	if conn == nil {
		Response(&c.Controller, "1002", "链接redis错误", nil)
		return
	}
	defer conn.Close()
	// 添加到购物车
	cacheKey := AddCartCacheKey(userInfo["userId"])
	oldNumber, _ := redis.Int(conn.Do("hget", cacheKey, skuId)) // 原来的数量
	_, _ = conn.Do("hset", cacheKey, skuId, number+oldNumber)

	// 返回json
	Response(&c.Controller, "200", "添加成功", GetCartCount(&c.Controller))
}

// 展示购物车页面
func (c *CartController) ShowUserCart() {
	conn := GetRedisConn()
	if conn == nil {
		c.TplName = "cart.html"
		return
	}
	defer conn.Close()

	userInfo := GetUserInfo(&c.Controller)
	cacheKey := AddCartCacheKey(userInfo["userId"])
	// 获取redis中购物车数据
	rep, err := conn.Do("hgetall", cacheKey) // rep是map[string]int
	goods, _ := redis.IntMap(rep, err)
	// 商品ids
	var ids []int
	for k := range goods {
		skuId, _ := strconv.Atoi(k)
		ids = append(ids, skuId)
	}
	// 查询商品数据
	var goodsSkus []*models.GoodsSKU
	_, _ = orm.NewOrm().QueryTable("GoodsSKU").Filter("Id__in", ids).All(&goodsSkus)
	// 拼接最终数据
	cartContents := make([]map[string]interface{}, len(goodsSkus))
	var totalPrice, totalCount int
	for k, v := range goodsSkus {
		temp := make(map[string]interface{})
		tempCount := goods[strconv.Itoa(v.Id)]
		rowPrice := v.Price * tempCount
		totalPrice += rowPrice
		totalCount += tempCount
		temp["goods"] = v
		temp["count"] = tempCount
		temp["rowPrice"] = rowPrice
		cartContents[k] = temp
	}

	c.Data["totalPrice"] = totalPrice
	c.Data["totalCount"] = totalCount
	c.Data["cartContents"] = cartContents
	c.TplName = "cart.html"
}

// 处理更新购物车中商品数量
func (c *CartController) HandleUpdateCart() {
	skuId, err1 := c.GetInt("skuId")
	count, err2 := c.GetInt("count")	// 这里是某个商品的总数, 不是单次增加的数
	if err1 != nil || err2 != nil {
		Response(&c.Controller, "1001", "请求参数错误", nil)
		return
	}

	conn := GetRedisConn()
	if conn == nil {
		Response(&c.Controller, "1002", "链接redis错误", nil)
		return
	}
	defer conn.Close()

	userInfo := GetUserInfo(&c.Controller)
	cacheKey := AddCartCacheKey(userInfo["userId"])
	_, _ = conn.Do("hset", cacheKey, skuId, count)

	Response(&c.Controller, "200", "请求成功", nil)
}

// 删除购物车商品操作
func (c *CartController) HandleDeleteCart() {
	skuId, err := c.GetInt("skuId")
	if err != nil {
		Response(&c.Controller, "1001", "商品id错误", nil)
		return
	}

	conn := GetRedisConn()
	if conn == nil {
		Response(&c.Controller, "1002", "redis connect error", nil)
		return
	}
	defer conn.Close()

	userInfo := GetUserInfo(&c.Controller)
	cacheKey := AddCartCacheKey(userInfo["userId"])
	_, _ = conn.Do("hdel", cacheKey, skuId)

	Response(&c.Controller, "200", "请求成功", nil)
}

// 返回购物车的数量函数 -- 种类数量
func GetCartCount(c *beego.Controller) (count int) {
	userInfo := GetUserInfo(c)
	if userInfo == nil {
		return
	}

	conn := GetRedisConn()
	if conn == nil {
		return
	}
	defer conn.Close()

	cacheKey := AddCartCacheKey(userInfo["userId"])
	// Int需要两个参数, Do返回两个参数, 所以可以直接赋值
	count, _ = redis.Int(conn.Do("hlen", cacheKey))

	return
}

// 添加到购物车的缓存key
func AddCartCacheKey(uid string) string {
	return "cart:addCart:" + uid
}

// 请求返回json函数
func Response(c *beego.Controller, code, msg string, data interface{}) {
	resp := make(map[string]interface{})
	resp["result"] = code
	resp["message"] = msg
	resp["data"] = data

	c.Data["json"] = resp
	c.ServeJSON()
}
