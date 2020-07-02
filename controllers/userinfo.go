package controllers

import (
	"fresh/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"strconv"
)

type UserInfoController struct {
	beego.Controller
}

// 展示用户中心信息页面
func (this *UserInfoController)  UserInfoShow() {
	// 用户登录判断
	UserLoginCheck(&this.Controller)
	// 获取用户名
	username := this.GetSession("username")
	this.Data["username"] = username


	// 查询用户的地址，需要查询地址表
	var addr models.Address
	var user models.User
	user.Name = username.(string)
	db := orm.NewOrm()
	db.Read(&user,"Name")
	_,err := db.QueryTable("Address").RelatedSel("User").Filter("User__Name",username).Filter("Isdefault",true).All(&addr)

	if addr.Id == 0 {
		this.Data["addr"] = ""
	}else {
		this.Data["addr"] = addr
	}

	var goods []models.GoodsSKU
	// 连接redis
	conn, err := redis.Dial("tcp","172.19.36.69:6379")
	if err != nil {
		beego.Info("连接rediss失败",err)
	}
	defer conn.Close()
	//查询数据
	conn.Do("auth","admin123")
	reply ,err := conn.Do("lrange","histroy"+strconv.Itoa(user.Id),0,4)
	replyInts,_ := redis.Ints(reply,err)
	for _,val := range replyInts{
		var temp models.GoodsSKU
		db.QueryTable("GoodsSKU").Filter("Id",val).One(&temp)
		goods = append(goods, temp)
	}
	this.Data["goods"] = goods

	this.Layout = "UserInfoLayout.html"
	this.TplName = "user_center_info.html"
}

// 用户订单展示
func (this *UserInfoController) UserOrderShow() {
	// 用户登录判断
	UserLoginCheck(&this.Controller)
	db := orm.NewOrm()
	// 获取用户id
	username := this.GetSession("username")
	var user models.User
	user.Name = username.(string)
	db.Read(&user,"Name")
	// 获取全部的订单
	var order_infos []models.OrderInfo
	db.QueryTable("OrderInfo").RelatedSel("User").Filter("User__Id",user.Id).All(&order_infos)
	// 创建容器存放订单
	goodsbuffer := make([]map[string]interface{},len(order_infos))
	// 给容器中插入数据
	for index,order_info := range order_infos {
		// 查询所有的订单商品表
		var order_goods []models.OrderGoods
		temp := make(map[string]interface{})

		db.QueryTable("OrderGoods").RelatedSel("GoodsSKU","OrderInfo").Filter("OrderInfo__Id",order_info.Id).All(&order_goods)
		temp["ordergoods"] = order_goods
		temp["orderinfo"] = order_info
		temp["time"] = order_info.Time.Format("2006-01-02 15:04:05")
		goodsbuffer[index] = temp
	}

	// 传递给视图
	this.Data["goodsbuffer"] = goodsbuffer

	this.Layout = "UserInfoLayout.html"
	this.TplName = "user_center_order.html"

}

// 用户地址页展示
func (this *UserInfoController) UserSiteShow() {
	// 用户登录判断
	UserLoginCheck(&this.Controller)

	// 获取当前登录的用户名
	username := this.GetSession("username")
	this.Data["user"] = username
	// 关联查询
	db := orm.NewOrm()
	var addr models.Address

	db.QueryTable("Address").RelatedSel("User").Filter("User__Name",username).Filter("Isdefault",true).One(&addr)

	// 传递给视图
	this.Data["addr"] = addr


	this.Layout = "UserInfoLayout.html"
	this.TplName = "user_center_site.html"
}

// 用户添加地址
func (this *UserInfoController) UserSiteHandle() {
	// 获取数据
	receiver := this.GetString("receiver")
	addr := this.GetString("addr")
	zipCode := this.GetString("zipCode")
	phone := this.GetString("phone")
	// 校验数据
	if receiver == "" || addr == "" || zipCode == "" || phone == "" {
		this.Data["errmsg"] = "添加的信息不正确，请重试"
		this.Redirect("/user/usersite",302)
		return
	}
	beego.Info("数据库处理")
	// 处理数据
	db := orm.NewOrm()
	useraddr := models.Address{}
	// 查询用户是否有默认的收货地址
	useraddr.Isdefault = true
	err := db.Read(&useraddr,"Isdefault")
	if err == nil { // 如果二人为空说明查询到了，也就是有了默认的收货地址
		// 将默认的收货地址改为false
		useraddr.Isdefault = false
		db.Update(&useraddr)
	}

	// 如果 err 不为空，说明没有默认的地址
	// 关联user表
	username := this.GetSession("username")
	// 获取user对象
	var user models.User
	user.Name = username.(string)
	err =db.Read(&user,"Name")
	beego.Info("查询",err)

	// 如果直接插入的话，因为上面的useraddr已经存在了id会导致插入失败
	// 所以要新建一个对象
	var addNew models.Address
	addNew.Receiver = receiver
	addNew.Addr = addr
	addNew.Zipcode = zipCode
	addNew.Phone = phone
	addNew.Isdefault = true
	// 赋值
	addNew.User = &user

	// 插入数据库
	_, err = db.Insert(&addNew)
	if err != nil {
		this.Data["errmsg"] = "插入地址信息错误"
		this.Redirect("/user/usersite",302)
		return
	}

	// 返回视图
	this.Redirect("/user/usersite",302)
}