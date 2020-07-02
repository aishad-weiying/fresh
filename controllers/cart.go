package controllers

import (
	"fresh/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"strconv"
)

type CartController struct {
	beego.Controller
}

// 封装函数获取购物车商品条目数
func GetCartNum(this *beego.Controller) int {
	// 购物车数据存储在redis中，key值为cart_用户id
	// 首先获取用户id
	username := this.GetSession("username")
	if username == nil {
		return 0
	}
	db := orm.NewOrm()
	var user models.User
	user.Name = username.(string)
	db.Read(&user, "Name")

	// 从redis获取购物车数量
	conn, err := redis.Dial("tcp", "172.19.36.69:6379")
	if err != nil {
		return 0
	}
	defer conn.Close()

	conn.Do("auth", "admin123")
	// 获取购物车数据条数
	rep, err := conn.Do("hlen", "cart_"+strconv.Itoa(user.Id))
	// do 返回的是interface类型的数据，借助回复助手函数
	num, _ := redis.Int(rep, err)

	return num

}

// 添加商品到购物车中
func (this *CartController) HandleAddCart() {
	// 获取数据
	id, err1 := this.GetInt("skuid")
	count, err2 := this.GetInt("count")

	// 返回json格式的数据给视图
	// beego 中map类型的数据就是json格式的，可以返回map类型的数据
	resp := make(map[string]interface{})

	// 发送json 数据
	defer this.ServeJSON()

	if err1 != nil || err2 != nil {
		// 传递标识，标识正确或者错误的状态
		resp["code"] = 400 // 错误
		// 传递信息
		resp["msg"] = "传递参数不正确"
		// 指定json数据
		this.Data["json"] = resp
		return
	}

	// 对用户登录做判断
	// 虽然已经有了路由过滤器做判断，但是ajax并不会跳转
	username := this.GetSession("username")

	if username == nil {
		// 传递标识，标识正确或者错误的状态
		resp["code"] = 302 // 错误
		// 传递信息
		resp["msg"] = "用户未登录"
		// 指定json数据
		this.Data["json"] = resp
		return
	}
	//获取用户id
	db := orm.NewOrm()
	var user models.User
	user.Name = username.(string)
	db.Read(&user, "Name")

	// 处理数据，存储到redis
	conn, err := redis.Dial("tcp", "172.19.36.69:6379")
	if err != nil {
		// 传递标识，标识正确或者错误的状态
		resp["code"] = 500 // 错误
		// 传递信息
		resp["msg"] = "redis连接失败"
		// 指定json数据
		this.Data["json"] = resp
		return
	}
	defer conn.Close()

	conn.Do("auth", "admin123")
	// 插入购物车之前，应该先获取到对应的原来的商品数量，如果存在累加，不存在直接插入
	Scount, _ := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), id))
	// 插入购物车数据
	conn.Do("hset", "cart_"+strconv.Itoa(user.Id), id, count+Scount)
	// 获取购物车数据条数
	num := GetCartNum(&this.Controller)

	// 传递标识，标识正确或者错误的状态
	resp["code"] = 200 // 正确
	// 传递信息
	resp["msg"] = "添加购物车成功"
	resp["num"] = num
	// 指定json数据
	this.Data["json"] = resp
}

// 展示购物车数据
func (this *CartController) ShowCart() {
	// //获取用户是否登录的信息
	UserLoginCheck(&this.Controller)

	username := this.GetSession("username")
	// 获取用户id
	db := orm.NewOrm()
	var user models.User
	user.Name = username.(string)
	db.Read(&user, "Name")

	// 从redis中获取数据
	conn, err := redis.Dial("tcp", "172.19.36.69:6379")
	if err != nil {
		beego.Info("redis 连接失败", err)
		return
	}
	defer conn.Close()
	conn.Do("auth", "admin123")

	// 获取所有的数据,conn.Do返回值为[]map[string]int,redis.IntMap获取到的返回值为map[string]int
	goodsMap, _ := redis.IntMap(conn.Do("hgetall", "cart_"+strconv.Itoa(user.Id)))
	// 购物车每一行存储的都是 商品对象和商品数量，那么应该定义一个容器来存储
	goods := make([]map[string]interface{}, len(goodsMap))
	//循环查询商品
	i := 0
	// 定义所有商品总价
	totalPrice := 0
	// 定义所有商品总数量
	totalNum := 0
	for index, num := range goodsMap {
		id, _ := strconv.Atoi(index)
		// 查询
		var goodsSKU models.GoodsSKU
		goodsSKU.Id = id
		db.Read(&goodsSKU)
		// 临时容器，用来存放单条的数据
		temp := make(map[string]interface{})
		temp["goodsSKU"] = goodsSKU
		temp["num"] = num
		// 获取单一个商品的总价
		temp["addprice"] = goodsSKU.Price * num
		// 所有商品的总价和数量
		totalPrice += goodsSKU.Price * num
		totalNum += num
		// 将临时的容器添加到大的容器中
		goods[i] = temp
		i += 1
	}

	//  传递给视图
	this.Data["goods"] = goods
	this.Data["totalPrice"] = totalPrice
	this.Data["totalNum"] = totalNum

	this.Layout = "cartlayout.html"
	this.TplName = "cart.html"
}

// 更新购物车数据
func (this *CartController) HandleCartUpdate() {
	// 获取数据
	id, err1 := this.GetInt("skuid")
	count, err2 := this.GetInt("count")

	// 返回json格式的数据给视图
	// beego 中map类型的数据就是json格式的，可以返回map类型的数据
	resp := make(map[string]interface{})

	// 发送json 数据
	defer this.ServeJSON()

	if err1 != nil || err2 != nil {
		// 传递标识，标识正确或者错误的状态
		resp["code"] = 400 // 错误
		// 传递信息
		resp["msg"] = "传递参数不正确"
		// 指定json数据
		this.Data["json"] = resp
		return
	}

	username := this.GetSession("username")
	//获取用户id
	db := orm.NewOrm()
	var user models.User
	user.Name = username.(string)
	db.Read(&user, "Name")

	// 处理数据，存储到redis
	conn, err := redis.Dial("tcp", "172.19.36.69:6379")
	if err != nil {
		// 传递标识，标识正确或者错误的状态
		resp["code"] = 500 // 错误
		// 传递信息
		resp["msg"] = "redis连接失败"
		// 指定json数据
		this.Data["json"] = resp
		return
	}
	defer conn.Close()

	conn.Do("auth", "admin123")
	// 插入购物车数据
	conn.Do("hset", "cart_"+strconv.Itoa(user.Id), id, count)

	// 传递标识，标识正确或者错误的状态
	resp["code"] = 200 // 正确
	// 传递信息
	resp["msg"] = "添加购物车成功"
	// 指定json数据
	this.Data["json"] = resp
}

// 删除购物车数据
func (this *CartController) HandleCartDelete() {
	// 获取用户传递的skuid
	skuid, err := this.GetInt("skuid")
	// 返回json格式的数据给视图
	// beego 中map类型的数据就是json格式的，可以返回map类型的数据
	resp := make(map[string]interface{})
	defer this.ServeJSON()
	// 数据校验
	if err != nil {
		// 传递标识，标识正确或者错误的状态
		resp["code"] = 400 // 错误
		// 传递信息
		resp["msg"] = "传递参数不正确"
		// 指定json数据
		this.Data["json"] = resp
		return
	}

	// 处理数据
	// 获取用户id
	username := this.GetSession("username")
	db := orm.NewOrm()
	var user models.User
	user.Name = username.(string)
	db.Read(&user, "Name")

	// 连接数据库
	conn, err := redis.Dial("tcp", "172.19.36.69:6379")
	if err != nil {
		// 传递标识，标识正确或者错误的状态
		resp["code"] = 500 // 错误
		// 传递信息
		resp["msg"] = "redis连接错误"
		// 指定json数据
		this.Data["json"] = resp
		return
	}
	defer conn.Close()

	// 删除数据
	conn.Do("auth", "admin123")
	conn.Do("hdel", "cart_"+strconv.Itoa(user.Id), skuid)
	// 返回数据

	// 传递标识，标识正确或者错误的状态
	resp["code"] = 200 // 正确
	// 传递信息
	resp["msg"] = "删除购物车成功"
	// 指定json数据
	this.Data["json"] = resp
}
