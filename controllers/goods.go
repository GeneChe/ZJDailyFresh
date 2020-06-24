package controllers

import (
	"dailyFresh/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"os"
)

type GoodsController struct {
	beego.Controller
}

// 展示首页
func (g *GoodsController) ShowHomePage() {
	// 获取类型数据
	var goodsTypes []*models.GoodsType
	o := orm.NewOrm()
	_, _ = o.QueryTable("GoodsType").All(&goodsTypes)
	// 获取轮播图数据
	var scrollBanner []*models.HomeScrollBanner
	_, _ = o.QueryTable("HomeScrollBanner").OrderBy("-Index").All(&scrollBanner)
	// 获取推广数据
	var promoteBanner []*models.HomePromotionBanner
	_, _ = o.QueryTable("HomePromotionBanner").OrderBy("-Index").All(&promoteBanner)

	// 获取首页展示商品数据
	// 定义首页数据格式 -- 字典切片
	homeData := make([]map[string]interface{}, len(goodsTypes))
	for k, v := range homeData { // slice and map 是值传递 改变临时变量, 原值也会修改
		// 1.类型数据
		v = make(map[string]interface{})
		v["type"] = goodsTypes[k]
		// 2.首页展示商品数据
		var textGoods, imgGoods []*models.HomeShowGoods
		_, _ = o.QueryTable("HomeShowGoods").
			RelatedSel("GoodsSKU").Filter("GoodsSKU__GoodsType__Id", goodsTypes[k].Id).
			Filter("DisplayType", 0).OrderBy("-Index").All(&textGoods)
		_, _ = o.QueryTable("HomeShowGoods").
			RelatedSel("GoodsSKU").Filter("GoodsSKU__GoodsType__Id", goodsTypes[k].Id).
			Filter("DisplayType", 1).OrderBy("-Index").All(&imgGoods)

		logs.Info(os.Stderr)	// 打印sql信息

		v["textGoods"] = textGoods
		v["imgGoods"] = imgGoods
		// 覆盖原来的值
		homeData[k] = v
	}

	g.Data["homeData"] = homeData
	g.Data["types"] = goodsTypes
	g.Data["scrollBanner"] = scrollBanner
	g.Data["promoteBanner"] = promoteBanner
	GetUserInfo(&g.Controller)
	g.TplName = "index.html"
}

// 展示商品详情
// 历史记录添加规则: 什么时候添加?  什么时候访问?  用什么存储及存储哪些内容?
// 1. 在登录后访问商品详情, 添加一条记录
// 2. 在用户中心页获取
// 3. 使用redis存储, 存储用户id对应的商品id, 而且有顺序要求
func (g *GoodsController) ShowGoodsDetail() {
	goodsId, err := g.GetInt("id")
	if err != nil {
		logs.Error("get goods detail err", err)
		g.Redirect("/", 302)
		return
	}

	o := orm.NewOrm()
	// 1.商品详情数据
	var goodsSKU models.GoodsSKU
	// goodsSKU.Id = goodsId
	// _ = o.Read(&goodsSKU)	缺少详情信息
	_ = o.QueryTable("GoodsSKU").RelatedSel("GoodsType", "GoodsSPU").
		Filter("Id", goodsId).One(&goodsSKU)

	// 2.新品推荐数据 -- 获取同类型时间靠前的前两条数据
	var newGoods []*models.GoodsSKU
	_, _ = o.QueryTable("GoodsSKU").RelatedSel("GoodsType").
		Filter("GoodsType", goodsSKU.GoodsType).
		OrderBy("-AddTime").Limit(2, 0).All(&newGoods)

	// 3.添加历史游览记录数据
	userInfo := GetUserInfo(&g.Controller)
	if userInfo != nil { // 用户已经登录
		conn := GetRedisConn()
		if conn != nil {
			defer conn.Close()
			cacheKey := GoodsHistoryCacheKey(userInfo["userId"])
			// 删除缓存中已有相同的商品记录 -- lrem key count value -- count为0表示删除全部value
			_, _ = conn.Do("lrem", cacheKey, 0, goodsId)
			// 存储记录 -- lpush key value
			_, _ = conn.Do("lpush", cacheKey, goodsId)
		}
	}

	showLayout(&g.Controller, "商品详情")
	g.Data["newGoods"] = newGoods
	g.Data["goodsSKU"] = goodsSKU
	g.TplName = "detail.html"
}

// 展示商品列表页
func (g *GoodsController) ShowGoodsList() {
	typeId, err := g.GetInt("typeId")
	if err != nil {
		logs.Error("get goods list err with wrong type id", err)
		g.Redirect("/", 302)
		return
	}

	o := orm.NewOrm()
	// 1.商品类型数据
	var goodsType models.GoodsType
	goodsType.Id = typeId
	_ = o.Read(&goodsType)

	// 2.新品推荐数据
	var newGoods []*models.GoodsSKU
	_, _ = o.QueryTable("GoodsSKU").Filter("GoodsType", typeId).
		OrderBy("-AddTime").Limit(2, 0).All(&newGoods)

	// 3.商品列表数据
	var goods []*models.GoodsSKU
	_, _ = o.QueryTable("GoodsSKU").Filter("goods_type_id", typeId).All(&goods)

	g.Data["typeInfo"] = goodsType
	g.Data["newGoods"] = newGoods
	g.Data["goods"] = goods
	showLayout(&g.Controller, "商品列表")
	g.TplName = "list.html"
}

// 用户游览商品记录redis缓存key
func GoodsHistoryCacheKey(uid string) string {
	return "user:goods:history:" + uid
}

// 商品模块layout视图
func showLayout(c *beego.Controller, title string) {
	var types []*models.GoodsType
	_, _ = orm.NewOrm().QueryTable("GoodsType").All(&types)

	c.Data["types"] = types
	c.Data["title"] = title
	GetUserInfo(c)
	c.Layout = "goods_layout.html"
}
