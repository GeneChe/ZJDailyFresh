package controllers

import (
	"dailyFresh/models"
	"errors"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"github.com/smartwalle/alipay"
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

/*
 高并发核心技术 —— 订单与库存
	一、 什么时候进行预占库存？
	（1）方案一：加入购物车的时候去预占库存
	（2）方案二：下单的时候去预占库存
	（3）方案三：支付的时候去预占库存

	二、 分析
	（1）方案一：加入购物车并不代表用户一定会购买,如果这个时候开始预占库存，会导致想购买的无法加入购物车。而不想购买的人一直占用库存。显然这种做法是不可取的。
	（2）方案二：商品加入购物车后，选择下单，这个时候去预占库存。用户选择去支付说明了，用户购买欲望是比 方案一 要强烈的。订单也有一个时效，例如半个小时。超过半个小时后，系统自动取消订单，回退预占库存。
	（3）方案三：下单成功去支付的时候去预占库存。只有100个用户能支付成功，900个用户支付失败。用户体验很不好。而且支付流程也是一个比较复杂的流程，如果和减库存放在一起，将会变的更复杂。

	所以综上所述： 选择方案二比较合理。
	三、 重复下单问题
	 1. 在UI拦截，点击后按钮置灰，不能继续点击，防止用户，连续点击造成的重复下单
	 2. 在下单前获取一个下单的唯一token [可有uid跟ip或设备id等组成]，下单的时候需要这个token。后台系统校验这个token是否有效，才继续进行下单操作。
*/
// 处理创建订单 -- 还需改进
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

	// 一次购物车生成一笔订单, 生成订单后清空购物车对应商品.
	// 保证了购物车过来的和要生成订单中对应商品数量是一致的
	// 2.插入订单商品表
	cacheKey := AddCartCacheKey(userInfo["userId"])
	for _, v := range goodsSkus {
		var goodsOrder models.OrderGoods
		goodsOrder.OrderInfo = &orderInfo
		goodsOrder.GoodsSKU = v

		count, _ := redis.Int(conn.Do("hget", cacheKey, v.Id))
		// 需要判断库存是否存够
		if count == 0 || count > v.Stock {
			err = errors.New("库存不足或商品数量有误")
			break
		}
		goodsOrder.Count = count
		goodsOrder.Price = count * v.Price

		// insert不支持批量插入操作
		_, err = o.Insert(&goodsOrder)
		if err != nil {
			break
		}

		// 扩大并发现象
		time.Sleep(time.Second * 5)

		// todo -- 思考12306的购票排队问题 --
		// todo 注意mysql事物的隔离级别
		// 更新库存, 销量
		// ** 注意更新操作需要加上判断条件 stock >= count **
		// 因为 前面的判断 跟 这里的更新操作 中间隔着别的操作, 在并发时会导致两处 stock 值不同. 所以还需再判断一次
		updateNum, _ := o.QueryTable("GoodsSKU").Filter("Id", v.Id).Filter("Stock__gte", count).Update(orm.Params{
				"Stock":orm.ColValue(orm.ColMinus, count),
				"Sales":orm.ColValue(orm.ColAdd, count),
			})
		if updateNum < 1 {
			err = errors.New("库存不足2")
			break
		}

		// 清空购物车中对应商品数量
		_, _ = conn.Do("hdel", cacheKey, v.Id)
	}

	if err != nil {
		logs.Error("create order err", err)
		_ = o.Rollback()
		Response(&or.Controller, "1003", err.Error(), nil)
		return
	} else {
		_ = o.Commit()
	}

	Response(&or.Controller, "200", "创建成功", nil)
}

// 处理支付
func (or *OrderController) ShowPay() {
	orderId, err := or.GetInt("orderId")
	if err != nil {
		logs.Error("order--showPay--err: ", err)
		or.Redirect("/user/userorder", 302)
		return
	}

	// 1. 获取订单信息
	o := orm.NewOrm()
	orderInfo := &models.OrderInfo{Id:orderId}
	err = o.Read(orderInfo)
	if err != nil {
		logs.Error("order--showPay: order not exits ", err)
		or.Redirect("/user/userorder", 302)
		return
	}

	// 组合支付参数拼接成支付链接
	payUrl := alipayTradePagePay(orderInfo)
	or.Redirect(payUrl, 302) // 跳转至支付页面

	// 由于同步回调是支付宝直接get return_url到本地显示, 所以无需外网地址即可. 但这不稳定
	// 异步回调是需要外网可访问地址.
	// 这里模拟支付成功来处理订单支付状态
	orderInfo.OrderStatus = 1
	_, _ = o.Update(orderInfo, "OrderStatus")
}

// 处理支付同步回调结果 -- 不稳定
func (or *OrderController) ShowPaySyncResult() {
	or.Ctx.WriteString("hello")
}

// alipay 拼接支付页
func alipayTradePagePay(orderInfo *models.OrderInfo) string {
	var privateKey = `MIIEpAIBAAKCAQEA9V7/Sdov7ycQoZFJRwcwSRYp1NdfHAl/cUYYax9ZapMiAhnK
bF8miEzGGYYpRU5nD397g54s4iVDzAdeW/sU2pxMM7mBR4uBCud/7lI+ScvLESEX
rj1Wl5+tsYfoX+yVVYCJ4+VI49YJhSrSD+uDPHsW4Zg5RYT2IP3dCVjT3uWRmDx5
pcFbaQmcMc4DGAjI7yEoftsZXboYNqs5twtb8pc+S6CafZg+FATYO1ZP865Jz1aU
oh5+OH31uks+v1T+2WWtgliMm/60e9RZ8SYO7hbguG4bTAM2j1bt8f6EkKFvjaKq
eM63+ILmhKsUfDykg3pf5+UPwihl81FXl0bmKwIDAQABAoIBAQDAHcZCm8wmMu8J
oci/DTjYMLtGA+9a83DOTvS1gxEuqc7Z2Dmuyn1QANSmjW3o7u8wqj8aGZHI6yZ/
LFHMMPXuCKx9X0SCsQ6za/i1r71HaIIxgjiZWzteck68Ds55tLJkBMVyI0cD5MUF
eDaK8nqJs1KCBf7pmKZhxIL5W4xgGtR5iRHAbQlGsWn6r/r9PEkoKPlXWsansoXQ
UmrgvAIRO2Uu6Y/bOWsSssLdUxJIWFeMyZwC+45f4M1FyBTnK9czXqU23UVtWEvI
BXO9XGnbb8KWJ9caHDt7O39CqM6PdE0iS9RI8kv4l12zve9oxxMkeSIhdcdknqvE
Bvbsp1qJAoGBAP2VNvRMZAdkX/y0mahibL02nByhmMrBBGJF72L9Ak7ReyvPG1ED
cen4RUyeWCdZRslM18aDWkD2n46RhJHg1ndWlZhkZ7KnTJcpwqaGrLR36qPM1aJh
TcN5bWJS3eHtibQFHAbEIXoDuN10C2lNwZY+ooK4iKN+dj8tI14pYXUNAoGBAPe1
vpDpPPN+Gyn/dQ2bxnNCkcLCTYwWzzeD4gDoaJiuHbtB5C3CZ2S549jxMTpU+XNN
oZbiVkuO/h9I9VGjoqw2xDSuA/KiH9d7rl8LBiBnznHiQlI49FUzANEK537MxDQ2
xsuDl5cYVD5xDDaeyRBgZBtBSrpKTQ7CNvWxxmoXAoGAenO2oMvOtd8blu0jEjPN
LKWVRyIlpSsF0erRiWyB08vGfcY5+6n9NS1lUXVZPk8XJpfLzpmZWKt/KxpL+SGo
juIpxPgfNx8glhJdY4q/FTqe/NAqDYqNQap+Tq+TY8kP6PVark3BmKj5eT7TT9tz
cvj2AsfXe5PSx/klDhBPdnUCgYBUNyTvzWwceE4x7BjWpJRGkWZO6ZJFw2d1v0+x
8VHGPsP66v7xk7tlIlHVasLKyyL30XfTfWXLUHUTG9HTjKd8ly4DnvnWnsnmj7UL
uQq/L6ufSkY0AAsJgEqRx3xGvsUh31Gc1UNPakUR6Ys2cqt29t5x6bPHPAWQs/TN
eUA0xwKBgQCukr6lh8NVMxabu3/27+kr5N7VzidHT5/PY++JrA0mRN8wmvl8Tv9i
lgVIsp1hNkXAnVSFV/Ici36b8AQwEeJQ4RxOG0YTug0JQOrJrzEb80zO3odKrt5s
h219Qs3OBqv2ZtPtrQnrZT6F+Tnlm0p9CWPw7M8TwQ/WEs3WpWp1LA==`
	var aliPublicKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAr03reLNAt6E7/lSfbGqiFoRD8R9Y5c7CahYKu2G9qrRD5DUrpcF0tVEzcyvUHDG7kS58QNisnEYdFnO9UDf3sb/PI780PyANeFDDI6ogK1UYEJVtq8zR2v85MngWv+QE8+UDnHQeJd3I45zEwLAsAYFOwhmikGrBlxKe5AZmbYXWEtfeJYCLEWAnWoWOZXVe4HkbBMPhSfCKUw7402MXrUWpW0pwZeaP+pNeYqpguLpkul2Zl7TMxXWu/mrXMHORDO8ByTQO3GOFdRaFWVx80Sc+cN3ioEA9lUvl80OdvUNafoost531NzSaDsRuXy7GXOvSsGc/WPQlXDWQZS8NXQIDAQAB"
	var client, err = alipay.New("2016102600764047", privateKey, false)
	if err != nil {
		logs.Error("alipay err: ", err)
		return "/user/userorder"
	}
	err = client.LoadAliPayPublicKey(aliPublicKey)
	if err != nil {
		logs.Error("alipay load key err: ", err)
		return "/user/userorder"
	}

	var p = alipay.TradePagePay{}
	p.NotifyURL = "http://xxx"
	p.ReturnURL = "http://192.168.3.11:8080/user/paySyncResult"
	p.Subject = "gene支付测试"
	p.OutTradeNo = orderInfo.OrderId
	p.TotalAmount = strconv.Itoa(orderInfo.TotalPrice)
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	payUrl, err := client.TradePagePay(p)
	if err != nil {
		logs.Error("alipay page pay err: ", err)
		return "/user/userorder"
	}

	return payUrl.String()
}