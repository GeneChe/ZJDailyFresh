package models

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type User struct { // 用户表
	Id         int
	Name       string	`orm:"size(20)"`
	Password   string	`orm:"size(20)"`
	Email      string	`orm:"size(50)"`
	IsActive   bool		`orm:"default(false)"`
	Permission int		`orm:"default(0)"`	// 权限设置 0 普通用户 1 管理员

	Addresses []*Address `orm:"reverse(many)"`
	Orders	  []*OrderInfo	`orm:"reverse(many)"`
}

type Address struct { // 地址表
	Id 			int
	Receiver	string	`orm:"size(20)"`
	Addr		string	`orm:"size(50)"`
	Phone 		string	`orm:"size(20)"`
	PostCode 	string	`orm:"size(20)"`
	IsDefault 	bool	`orm:"default(false)"`	// 是否默认

	User		*User	`orm:"rel(fk)"`			// 用户id
	Orders		[]*OrderInfo `orm:"reverse(many)"`
}

type GoodsSPU struct { // 商品SPU表
	Id 		int
	Name 	string		`orm:"size(20)"`
	GoodsDetail	string	`orm:"size(200)"`

	GoodsSKUs []*GoodsSKU `orm:"reverse(many)"`
	GoodsType *GoodsType  `orm:"rel(fk)"`
}

type GoodsType struct { // 商品类型表
	Id 		int
	Name 	string		`orm:"size(20)"`
	Logo	string
	typeImg string

	GoodsSPUs []*GoodsSPU `orm:"reverse(many)"`
	HomeShowGoods []*HomeShowGoods `orm:"reverse(many)"`
}

type GoodsSKU struct {	// 商品SKU表
	Id 			int
	Name 		string	`orm:"size(20)"`
	Desc		string
	Price		int
	GoodsUnit 	string
	GoodsImg	string
	Stock		int		`orm:"default(1)"`
	Sales		int		`orm:"default(0)"`
	Status 		int		`orm:"default(1)"`	// 商品状态 0 下线 1上线
	AddTime		time.Time	`orm:"auto_now_add"`	// 添加时间

	GoodsSPU 	*GoodsSPU	`orm:"rel(fk)"`
	GoodsImages []*GoodsImage	`orm:"reverse(many)"`
	HomeShowGoods []*HomeShowGoods	`orm:"reverse(many)"`
	HomeScrollBanner []*HomeScrollBanner `orm:"reverse(many)"`
	OrderGoods	[]*OrderGoods	`orm:"reverse(many)"`
}

type GoodsImage struct {	// 商品图片表
	Id 		int
	Image 	string

	GoodsSKU	*GoodsSKU	`orm:"rel(fk)"`	// 商品sku
}

type HomeScrollBanner struct {	// 首页轮播商品表
	Id 		int
	Image 	string
	Index 	int		`orm:"default(0)"`	// 展示顺序

	GoodsSKU *GoodsSKU	`orm:"rel(fk)"`	// 商品sku
}

type HomePromotionBanner struct {	// 首页推广商品表
	Id 		int
	Name 	string	`orm:"size(20)"`
	Url 	string	`orm:"size(100)"`
	Image	string
	Index 	int		`orm:"default(0)"`
}

type HomeShowGoods struct {	// 首页展示商品表
	Id 			int
	DisplayType	int		`orm:"default(1)"`	// 展示类型	0 图片 1 文字
	Index 		int		`orm:"default(0)"`

	GoodsType	*GoodsType	`orm:"rel(fk)"`
	GoodsSKU	*GoodsSKU	`orm:"rel(fk)"`
}

type OrderInfo struct {	// 订单表
	Id 			int
	OrderId		string	`orm:"unique"`
	PayWay		int		// 支付方式: 0 支付宝, 1 微信, 2 银行卡
	TotalCount	int		// 商品总数量
	TotalPrice	int		// 商品总价格
	TransitPrice int	// 运费
	OrderStatus int		`orm:"default(0)"`	// 订单状态 0 未付款, 1 付款成功, 2 付款失败
	TradeNo		string	// 支付编号
	OrderTime	time.Time	`orm:"auto_now_add"`	// 订单创建时间
	OrderFinishTime time.Time `orm:"auto_now"`		// 订单完成时间
	OrderCommentTime time.Time `orm:"auto_now_add"`	// 订单评论时间
	Comment 	string

	User		*User		`orm:"rel(fk)"`
	Addr		*Address	`orm:"rel(fk)"`
	OrderGoods	[]*OrderGoods	`orm:"reverse(many)"`
}

type OrderGoods struct {	// 订单商品表
	Id 		int
	Count	int		`orm:"default(1)"`

	OrderInfo	*OrderInfo	`orm:"rel(fk)"`
	GoodsSKU	*GoodsSKU	`orm:"rel(fk)"`
}

func init() {
	// RegisterDataBase
	_ = orm.RegisterDataBase("default", "mysql", "root:zj2fighting@(192.168.3.99:3306)/dailyFresh?charset=utf8&loc=local")

	// RegisterModel
	orm.RegisterModel(
		new(User), new(Address), new(OrderGoods),
		new(OrderInfo), new(HomeShowGoods), new(HomeScrollBanner),
		new(HomePromotionBanner), new(GoodsSKU), new(GoodsSPU),
		new(GoodsType), new(GoodsImage),
	)

	// create table
	_ = orm.RunSyncdb("default", false, true)
}
