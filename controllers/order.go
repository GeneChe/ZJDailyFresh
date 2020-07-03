package controllers

import (
	"dailyFresh/models"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"strconv"
	"strings"
	"time"
)

type OrderController struct {
	beego.Controller
}

// 展示提交订单页
func (or *OrderController) HandlePlaceOrder() {
	// 获取提交过来的数组
	skuIds := or.GetStrings("skuId")
	// 校验提交的是否为空
	if len(skuIds) == 0 {
		logs.Error("place order err sku id is empty")
		or.Redirect("/user/userCart", 302)
		return
	}
	or.Data["skuIds"] = skuIds

	ormObj := orm.NewOrm()
	userInfo := GetUserInfo(&or.Controller)
	// 1. 根据用户id获取地址信息
	var addrs []*models.Address
	_, _ = ormObj.QueryTable("Address").Filter("User", userInfo["userId"]).OrderBy("-IsDefault").All(&addrs)
	or.Data["addrs"] = addrs

	// 2. 根据商品id从数据库中获取商品信息 和 从redis中获取商品数量
	var goodsSkus []*models.GoodsSKU
	_, _ = ormObj.QueryTable("GoodsSKU").Filter("Id__in", skuIds).All(&goodsSkus)

	conn := GetRedisConn()
	if conn == nil {
		or.Redirect("/user/userCart", 302)
		return
	}
	defer conn.Close()

	cacheKey := AddCartCacheKey(userInfo["userId"])
	// 组合页面需要的数据结构
	goodsList := make([]map[string]interface{}, len(goodsSkus))
	var totalNum int
	var totalPrice float64
	for k, v := range goodsSkus {
		temp := make(map[string]interface{})
		temp["index"] = k+1
		temp["goods"] = v

		// 获取数量
		count, _ := redis.Int(conn.Do("hget", cacheKey, v.Id))
		temp["count"] = count

		// 计算小计
		tempPrice := float64(count * v.Price)
		temp["rowPrice"] = fmt.Sprintf("%.2f", tempPrice)

		// 计算总金额/总件数
		totalNum += count
		totalPrice += tempPrice

		goodsList[k] = temp
	}

	or.Data["goodsList"] = goodsList
	var transPrice float64 = 10 	// 运费先固定10元
	or.Data["totalNum"] = totalNum
	or.Data["totalPrice"] = totalPrice
	or.Data["transPrice"] = transPrice
	or.Data["actualPrice"] = totalPrice + transPrice
	or.Data["pageTitle"] = "提交订单"
	or.Layout = "cart_layout.html"
	or.TplName = "place_order.html"
}

// 处理创建订单
func (or *OrderController) HandleCreateOrder() {
	addrId, _ := or.GetInt("addrId")
	payType, _ := or.GetInt("payType")
	// **注意 页面传过来的是字符串 "[1 2 3]"
	skuIds := or.GetString("skuIds")
	totalC, _ := or.GetInt("totalC")
	transP, _ := or.GetFloat("transP")
	actualP, _ := or.GetFloat("actualP")
	if addrId <= 0 || payType <= 0 || len(skuIds) <= 0 {
		Response(&or.Controller, "1001", "请求参数错误", nil)
		return
	}

	conn := GetRedisConn()
	if conn == nil {
		Response(&or.Controller, "1002", "conn redis err", nil)
		return
	}
	defer conn.Close()

	userInfo := GetUserInfo(&or.Controller)
	userId, _ := strconv.Atoi(userInfo["userId"])
	o := orm.NewOrm()

	// ** 不要将查询放在事物中间, 不然在commit之前不会执行查询, 查询结果是空, 就会影响需要查询结果的后续操作 **
	skuIds = skuIds[1:len(skuIds)-1] // 去除首位
	skuIdArr := strings.Fields(skuIds)
	var goodsSkus []*models.GoodsSKU
	_, _ = o.QueryTable("GoodsSKU").Filter("Id__in", skuIdArr).All(&goodsSkus)

	// 开始事物
	err := o.Begin()
	// 1.插入订单表
	var orderInfo models.OrderInfo
	orderInfo.OrderId = time.Now().Format("20060102150405") + userInfo["userId"]
	orderInfo.PayWay = payType
	orderInfo.TotalCount = totalC
	orderInfo.TransitPrice = int(transP)
	orderInfo.TotalPrice = int(actualP)
	orderInfo.User = &models.User{Id: userId}
	orderInfo.Addr = &models.Address{Id: addrId}
	_, err = o.Insert(&orderInfo)

	// 2.插入订单商品表
	cacheKey := AddCartCacheKey(userInfo["userId"])
	for _, v := range goodsSkus {
		var goodsOrder models.OrderGoods
		goodsOrder.OrderInfo = &orderInfo
		goodsOrder.GoodsSKU = v
		// ?? 问题 不同订单商品同种商品数量不一样, 存的时候未区分订单id, 直接拿的是这个用户这个商品总的数量. ??
		// 即使付完款后清除redis中这个商品的个数, 在同批商品创建多个订单时也会出问题??
		count, _ := redis.Int(conn.Do("hget", cacheKey, v.Id))
		// 需要判断库存是否存够
		goodsOrder.Count = count
		goodsOrder.Price = count * v.Price

		// insert不支持批量插入操作
		_, err = o.Insert(&goodsOrder)
		if err != nil {
			break
		}
	}

	if err != nil {
		logs.Error("create order err", err)
		_ = o.Rollback()
		Response(&or.Controller, "1003", "数据错误", nil)
		return
	} else {
		_ = o.Commit()
	}

	Response(&or.Controller, "200", "创建成功", nil)
}