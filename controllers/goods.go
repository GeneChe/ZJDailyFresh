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
func (g *GoodsController) ShowGoodsDetail() {
	goodsId, err := g.GetInt("id")
	if err != nil {
		logs.Error("get goods detail err", err)
		g.Redirect("/", 302)
		return
	}

	o := orm.NewOrm()
	// 商品详情数据
	var goodsSKU models.GoodsSKU
	// goodsSKU.Id = goodsId
	// _ = o.Read(&goodsSKU)	缺少详情信息
	_ = o.QueryTable("GoodsSKU").RelatedSel("GoodsType", "GoodsSPU").
		Filter("Id", goodsId).One(&goodsSKU)

	// 新品推荐数据 -- 获取同类型时间靠前的前两条数据
	var newGoods []*models.GoodsSKU
	_, _ = o.QueryTable("GoodsSKU").RelatedSel("GoodsType").
		Filter("GoodsType", goodsSKU.GoodsType).
		OrderBy("-AddTime").Limit(2, 0).All(&newGoods)

	showLayout(&g.Controller, "商品详情")
	g.Data["newGoods"] = newGoods
	g.Data["goodsSKU"] = goodsSKU
	g.TplName = "detail.html"
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
