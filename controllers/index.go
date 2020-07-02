package controllers

import (
	"fresh/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"strconv"
)

type IndexController struct {
	beego.Controller
}
func UserLoginCheck(this *beego.Controller)  {
	username := this.GetSession("username")

	if username == "" {
		this.Data["username"] = ""
	}else {
		this.Data["username"] = username
	}
}

// 首页显示
func (this *IndexController) IndexShow() {
	// 判断用户是否登录
	UserLoginCheck(&this.Controller)
	db := orm.NewOrm()
	// 获取商品类型
	var goodsType []models.GoodsType
	db.QueryTable("GoodsType").All(&goodsType)
	this.Data["types"] = goodsType

	// 获取轮播图数据
	var banner []models.IndexGoodsBanner
	db.QueryTable("IndexGoodsBanner").OrderBy("Index").All(&banner)
	this.Data["banner"] = banner

	// 获取促销商品
	var IndexPromotionBanner []models.IndexPromotionBanner
	db.QueryTable("IndexPromotionBanner").OrderBy("Index").All(&IndexPromotionBanner)
	this.Data["IndexPromotionBanner"] = IndexPromotionBanner

	// 获取首页商品
	goods := make([]map[string]interface{},len(goodsType))

	// 存储商品类型到容器中
	for index , value := range goodsType {
		// value 为现有的类型数据，但是为了标示具体是什么数据，在容器中定义标识
		temp := make(map[string]interface{})
		temp["type"] = value
		goods[index] = temp
	}

	// 存储文字商品和图片商品
	var goodsImage []models.IndexTypeGoodsBanner
	var goodsText []models.IndexTypeGoodsBanner

	for _ , temp := range goods {
		// 查询图片商品，赋值给结构体对象
		db.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsSKU","GoodsType").Filter("GoodsType",temp["type"]).Filter("DisplayType",1).OrderBy("Index").All(&goodsImage)
		// 查询文字商品，赋值给结构体对象
		db.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsSKU","GoodsType").Filter("GoodsType",temp["type"]).Filter("DisplayType",0).OrderBy("Index").All(&goodsText)
		// 将结构体对象添加到容器中
		temp["goodsText"] = goodsText
		temp["goodsImage"] = goodsImage
	}

	// 把容器传递给视图
	this.Data["goods"] = goods
	ShowLayout(&this.Controller)
	// 获取购物商品数量
	num := GetCartNum(&this.Controller)
	this.Data["num"] = num
	this.TplName = "index.html"
}

// 封装函数用来获取商品类型,传递给goodslayout
func ShowLayout(this *beego.Controller)  {
	// 获取所有的商品类型
	db := orm.NewOrm()
	var types []models.GoodsType
	db.QueryTable("GoodsType").All(&types)
	this.Data["types"] = types
	//获取用户是否登录的信息
	UserLoginCheck(this)
	// 指定layout
	this.Layout = "goodslayout.html"
}

// 展示商品的详情
func (this *IndexController) ShowGoodsInfo() {
	//获取id
	id ,err := this.GetInt("id")
	// 判断数据
	if err != nil {
		beego.Info("传递的参数错误",err)
		this.Redirect("/",302)
		return
	}

	// 根据id查询数据库
	db := orm.NewOrm()
	var goodsSKU models.GoodsSKU
	goodsSKU.Id = id
	db.QueryTable("goodsSKU").RelatedSel("GoodsType","Goods").Filter("Id",id).One(&goodsSKU)
	// 传递数据给视图
	this.Data["goods"] = goodsSKU

	// 获取同类型商品的最新两条信息
	var goodsSKU2 []models.GoodsSKU
	db.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType",goodsSKU.GoodsType).OrderBy("Time").Limit(2,0).All(&goodsSKU2)
	this.Data["new2"] = goodsSKU2

	// 判断用户是否登录
	username := this.GetSession("username")

	if username != nil {
		// 查询用户信息
		var user models.User
		user.Name = username.(string)
		db := orm.NewOrm()
		db.Read(&user, "Name")
		// 添加历史记录
		// 连接redis
		conn, err := redis.Dial("tcp", "172.19.36.69:6379")
		if err != nil {
			beego.Info("redis连接失败", err)
			this.Redirect("/", 302)
			return
		}
		defer conn.Close()
		//插入历史
		conn.Do("auth", "admin123")
		// 如果多次浏览一个商品，只添加一次
		// 那么在插入之前，先把这个商品之前的记录在list中移除
		reply, err := conn.Do("lrem", "histroy"+strconv.Itoa(user.Id), 0, id)
		reply, _ = redis.Bool(reply, err)
		if reply == false {
			beego.Info("清除浏览历史失败",err)
		}
		_, err = conn.Do("lpush", "histroy"+strconv.Itoa(user.Id), id)
		if err != nil {
			beego.Info("插入失败2", err)
		}
	}
	// 调用封装的函数
	ShowLayout(&this.Controller)
	// 获取购物商品数量
	num := GetCartNum(&this.Controller)
	this.Data["num"] = num
	this.TplName = "detail.html"
}

// 商品搜索
func (this *IndexController) HandleGoodsSearch() {
	/// 获取数据
	goodsname := this.GetString("goodsname")

	// 定义容器存储查找的商品
	var goods []models.GoodsSKU
	db := orm.NewOrm()
	// 校验数据
	// 如果 goodsname 为空，显示所有的商品
	if goodsname == "" {
		db.QueryTable("GoodsSKU").All(&goods)
		this.Data["goods"] = goods
		ShowLayout(&this.Controller)
		this.TplName = "search.html"
	}

	db.QueryTable("GoodsSKU").Filter("Name__icontains",goodsname).All(&goods)
	this.Data["goods"] = goods
	ShowLayout(&this.Controller)
	this.TplName = "search.html"
}