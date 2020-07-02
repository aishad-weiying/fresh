package controllers

import (
	"fmt"
	"fresh/models"
	"github.com/astaxie/beego"
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

func (this *OrderController) ShowOrder() {
	// 判断用户是否登录
	UserLoginCheck(&this.Controller)
	// 获取数据,因为在传递的时候，是循环传递的，那么得到的应该是一个切片
	skuids := this.GetStrings("skuid")

	// 校验数据
	if len(skuids) == 0 {
		beego.Info("获取数据失败")
		this.Redirect("/user/mycart", 302)
		return
	}
	// 获取用户
	db := orm.NewOrm()
	username := this.GetSession("username")
	var user models.User
	user.Name = username.(string)
	db.Read(&user, "Name")
	// 价钱总计
	Zsum := 0
	// 计算商品总件数
	Zcount := 0
	// 获取所有的商品和数量
	// 1. 创建容器来存储商品和数量
	goodsbuffer := make([]map[string]interface{}, len(skuids))

	// 2. 循环获取id和数量
	for index, skuid := range skuids {
		temp := make(map[string]interface{}, len(skuids))

		id, _ := strconv.Atoi(skuid)
		// 查询商品数据
		var goodssku models.GoodsSKU
		goodssku.Id = id
		db.Read(&goodssku, "Id")
		//存放商品
		temp["goodssku"] = goodssku
		// 获取商品数量
		conn, err := redis.Dial("tcp", "172.19.36.69:6379")
		if err != nil {
			beego.Info("redis error", err)
			return
		}
		defer conn.Close()
		conn.Do("auth", "admin123")
		// 查询数量，返回值是interface类型，使用回复助手函数
		count, _ := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), id))
		// 存放数量
		temp["count"] = count
		// 存放小计
		temp["sum"] = count * goodssku.Price
		// 存放总计和总件数
		Zsum += count * goodssku.Price
		Zcount += count
		// 加入到容器中
		goodsbuffer[index] = temp
	}

	// 将数据传递给视图
	this.Data["goods"] = goodsbuffer
	this.Data["Zsum"] = Zsum
	this.Data["Zcount"] = Zcount
	// 实际价钱，加上运费
	this.Data["Ssum"] = Zsum + 10
	// 获取用户地址
	var add []models.Address
	db.QueryTable("Address").RelatedSel("User").Filter("User", user).All(&add)
	this.Data["add"] = add
	// 把所有的商品id传递给视图
	this.Data["skuids"] = skuids
	this.TplName = "place_order.html"
}

// 添加订单
func (this *OrderController) HandleAddOrder() {
	// 获取数据
	addrId, _ := this.GetInt("addrId")
	payId, _ := this.GetInt("payId")
	skuid := this.GetString("skuid")
	// skuid 之前传递的是切片，但是jquery获取到的只能是常规的值，将切片变为了字符串
	// 那么在这里获取成了字符串类型，需要转换为切片
	// 获取到的字符串为 [id id id] ，先切割，只要id数据
	ids := skuid[1 : len(skuid)-1]
	// 以空格为分隔符转换成切片
	skuids := strings.Split(ids, " ")

	totalCount, _ := this.GetInt("totalCount")
	transferPrice, _ := this.GetFloat("transferPrice")
	transfer, _ := this.GetFloat("transfer")
	beego.Info(addrId, payId, skuid, totalCount, transferPrice, transfer)

	// 定义json格式的数据
	resp := make(map[string]interface{})
	defer this.ServeJSON()

	// 校验数据
	if len(skuid) == 0 {
		// 传递标识，标识正确或者错误的状态
		resp["code"] = 400 // 错误
		// 传递信息
		resp["msg"] = "传递参数不正确"
		// 指定json数据
		this.Data["json"] = resp
		return
	}
	// 处理数据
	db := orm.NewOrm()
	username := this.GetSession("username")
	var user models.User
	user.Name = username.(string)
	db.Read(&user, "Name")
	db.Begin() // 开始事务，如果出现问题，都不能插入数据
	// 向订单表中插入数据
	var order models.OrderInfo
	// 订单id 时间 + 用户id
	order.OrderId = time.Now().Format("20060102150405") + strconv.Itoa(user.Id)
	// 插入用户信息
	order.User = &user
	// 订单状态 1 为未支付，2为支付成功
	order.Orderstatus = 1
	// 支付方式,1为货到付款 ，2为微信支付 ，3为支付宝支付 ，4为银行卡支付
	order.PayMethod = payId
	// 总件数 ,运费和总钱数
	order.TotalCount = totalCount
	order.TransitPrice = int(transferPrice)
	order.TotalPrice = int(transfer)
	// 查询地址
	var addr models.Address
	addr.Id = addrId
	db.Read(&addr)
	// 插入地址
	order.Address = &addr
	// 执行插入
	db.Insert(&order)
	// 连接redis
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
	conn.Do("auth", "admin123")
	// 向订单商品表中插入数据
	for _, val := range skuids {
		id, _ := strconv.Atoi(val)
		// 根据id查询商品表
		var goods models.GoodsSKU
		goods.Id = id
		i := 3
		for i > 0 {
			db.Read(&goods)
			// 将商品信息插入到订单商品表
			var order_goods models.OrderGoods
			order_goods.GoodsSKU = &goods
			// 将订单信息插入到订单商品表
			order_goods.OrderInfo = &order
			// 将单一商品数量插入到订单商品表,在redis中获取
			conn.Do("auth", "admin123")
			count, _ := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), id))
			order_goods.Count = count
			order_goods.Price = goods.Price
			// 获取库存量
			precount := goods.Stock
			// 判断添加的商品数量是否大于库存
			if count > goods.Stock {
				resp["code"] = 404 // 错误
				// 传递信息
				resp["msg"] = "商品库存不足"
				// 指定json数据
				this.Data["json"] = resp
				db.Rollback() // 如果出现问题，事务回滚
				return
			}
			// 更新库存和销量
			goods.Stock -= count
			goods.Sales += count
			// 更新数据的时候，判断库存量
			// 返回更新数据的条数和错误信息
			updateCount, _ := db.QueryTable("GoodsSKU").Filter("Id", goods.Id).Filter("Stock", precount).Update(orm.Params{"Stock": goods.Stock, "Sales": goods.Sales})
			if updateCount == 0 {
				if i > 0 {
					// 如果更新失败，但是i> 0 继续执行下一次循环判断
					i -= i
					continue
				}
				// 没有更新，代表两次判断的库存不一致
				resp["code"] = 404 // 错误
				// 传递信息
				resp["msg"] = "商品库存改变，订单提交失败"
				// 指定json数据
				this.Data["json"] = resp
				db.Rollback() // 如果出现问题，事务回滚
				return
			} else {
				// 插入
				db.Insert(&order_goods)
				// 如果更新成功了，那么添加订单成功，退出循环并清空购物车
				conn.Do("hdel", "cart_"+strconv.Itoa(user.Id), goods.Id)
			}
		}
	}
	// 如果正确，结束事务
	db.Commit()
	// 返回数据
	// 传递标识，标识正确或者错误的状态
	resp["code"] = 200 // 错误
	// 传递信息
	resp["msg"] = "添加订单成功"
	// 指定json数据
	this.Data["json"] = resp
}

// 付款
func (this *OrderController) HandleGoPay() {
	privateKey := "MIIEpQIBAAKCAQEArTpk9s6iUcBL5bt0KUHeohb8A8OFhNutFrPzqDDag9IyBFrq"+
		"D7E0A5uPu8DrRgDreRmm3z95aYe9gHnPI5eVOaa3FsoqNziGk2P2oerpIwIpyXb4"+
		"oT3j+qDH9MSW1VomoqoBigWQv57/VDGJ0/APtjqOpgIxyPNwLyNx88mwAsDwWcti"+
		"4tOc1JuioDGl8MDSEK/5b6RVlAtqZgsyj2fFPRW2u9ETgcdakG4ufum9y/B2Erz2"+
		"8EBqAjusS0U1cmPOxzhAwil6oS8ibYwgX5f9qOeqkTQzqRD2T63StjrDgvYhvTui"+
		"nLmvU6UeuU+c3xMvCrrhqWXkJNGLhbA5GXj+xQIDAQABAoIBAQCplqGBfooqvreP"+
		"ERWHzpTG2vmeNaxhhS7PKx0/a5SBuSZ+XQMImdLyNTlh9lxfDEd7J0HXDa0vQ1Si"+
		"kp3Xmm7dIfnctc7egNg/M34gxnm3bEa8lVyTfqPSmdUoK83+0WNBnG4lUY2EC4Ss"+
		"SFNGyZ6YKHu+yiczNCCABZNpt+o4xk4o2HwOY5MOJkfnlwYyIFCMVfz3WxRt2sw1"+
		"nvjQPl+JIWBf0oU9SAxTs9GflJ3fQaWXrfUBU8W/0LLZwLXLUXPethwCKRNpEqQ4"+
		"hu2k/nnyQQ6I0of25UJo+bbm1rXOlX3oV9ewPTlJH2RiYHFfhP7BI8Q71io4b0Mf"+
		"ZlLXPk+BAoGBANWXsiceV8AXPwxxHCfY49IhDQeqO6h8oyqcv0e1EzDaqR2WRk0I"+
		"G6XM7ggMXAA6AuAHpwXC9JH95jvX3GpU8PchECJHrjcd6Xf58/sTa0UP1G4ajaFr"+
		"xWfy+GpE8aC9JOz5LrF7FnJdau5IIDND+AXpJO4kW5xhebPsXua9Yks/AoGBAM+f"+
		"Fmepi279vUfcOTAr4hin2pN9TQzQ/o2bKL9hc6u4WYkuiDOUTZEoXip0swiRKfWz"+
		"7e1pyC06w42l5JTuLYPTcY58P62ruQlGGHC1lzYKmnJu256WPGOIQ2O3VACIfD+g"+
		"mJ/k5+R0JlhFf8XKpEv2rMjgOgI/F4I/50YcsMj7AoGBAKe4dLZvByzZlDKq5xcL"+
		"Iuov9dFdBXeqV13ws+sU7zrfmQiYph97DGrHXuqG+f9bjkJo/+hwTCgPnajEOlps"+
		"1MLZ/ZdNfindnSUO61zuxL74TTEgPLLSs7KKgjLAbJRxsfs7OEU5iEjJvlvZ2x8m"+
		"ci4CA3PUrPNBP5XfOC4r7HF1AoGASDiRpZuPehtflTig2AXbzzHMUZO7kqK8eWuo"+
		"n/H5N5mX46VBEZgb50uAfgo8INXGH8boE7bBQCJ51bMIMVoskPejP6ouyG28nuI4"+
		"LDSulcjYcsfnM2IVPZYvwucJnGndtpBZpv0MQSa6E+iRCq9zuUzkS7fb1d42gkNS"+
		"YswmHrMCgYEAh4FMi5/0JH+LRiTlpcxxYHAgZHSbmlu2uhphMLDuuSV7YfngGGty"+
		"XL9QEyogCS4uo8EB53D+d+06sJNE9AJ+kUIeGsf6DNdAm9Dze8Y6DWlunqdIk8FZ"+
		"5oBTn34v6eLG5Q7E3LqgtTOGS2R2vGJ3ulernsW6oL0LwBaROSX29Wo="
	var client, err1 = alipay.New("2016102600762715", privateKey, false)

	//client.LoadAppPublicCertFromFile("appCertPublicKey_2017011104995404.crt") // 加载应用公钥证书
	//client.LoadAliPayRootCertFromFile("alipayRootCert.crt") // 加载支付宝根证书
	//client.LoadAliPayPublicCertFromFile("alipayCertPublicKey_RSA2.crt") // 加载支付宝公钥证书
	client.LoadAliPayPublicKey("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAp3ckRnizjjColOyIVyZ1OTF/jTEjPSeytfu/+m6oezkQ+ne2dRDjpWTAILuZJ/dJcyObIfgXb+u/6mtrbjmKTy2QzKtpWHWsXCWBNIgWCbPfVxhkZ0/kRP4UNxeOf8eXo7MUxPezjfwXKJas4plr8t7yOQTY9e6Ru+DMUBQKLV/QTg6PKKrU4gIDjapAoPpCe74IQn93bGwdy4m6PQXPp0wCgbmCF7yPzcxFdQa8BDvvfV9yLPqhBeFqR3wV2XENcRMVvzk3oSXAaOAiMEw6X4Yr0ShMHcktv6kh3n74vx0Ta6lv+UilQpwX4zc/7o9wvK0Me3x/9bzLMWSOEfflYQIDAQAB")

	// 将 key 的验证调整到初始化阶段
	if err1 != nil {
		fmt.Println("err1111",err1)
		return
	}

	var p = alipay.TradePagePay{}
	//p.NotifyURL = "http://xxx"
	p.ReturnURL = "http://172.19.36.69/user/userorder"
	p.Subject = "沙箱测试"
	p.OutTradeNo = "202007021050"
	p.TotalAmount = "100"
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"
	// ksrynb1451@sandbox.com

	var url, err = client.TradePagePay(p)
	if err != nil {
		fmt.Println("errrrr",err)
	}

	var payURL = url.String()
	this.Redirect(payURL,302)
	// 这个 payURL 即是用于支付的 URL，可将输出的内容复制，到浏览器中访问该 URL 即可打开支付页面。

}
