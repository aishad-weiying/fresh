package routers

import (
	"github.com/astaxie/beego/context"
	"fresh/controllers"
	"github.com/astaxie/beego"
)

func init() {

	// 设置路由过滤器
	var BeferExecFunc = func(ctx *context.Context) {
		username := ctx.Input.Session("username")
		if username == nil {
			ctx.Redirect(302,"/login")
		}
	}
	beego.InsertFilter("/user/*",beego.BeforeRouter,BeferExecFunc)
	// 用户注册的路由
	beego.Router("/register",&controllers.UserController{},"get:UserRegiterShow;post:UserRegisterHandle")
	// 用户激活的路由
    beego.Router("/active",&controllers.UserController{},"get:UserActive")
	// 用户登录的路由
	beego.Router("/login",&controllers.UserController{},"get:UserLogin;post:UserLoginHandle")
	// 首页展示的路由
	beego.Router("/",&controllers.IndexController{},"get:IndexShow")
	// 用户退出登录的路由
	beego.Router("/user/logout",&controllers.UserController{},"get:UserLogout")
	// 用户中心展示的路由
	beego.Router("/user/userinfo",&controllers.UserInfoController{},"get:UserInfoShow")
	// 用户订单页面展示的路由
	beego.Router("/user/userorder",&controllers.UserInfoController{},"get:UserOrderShow")
	// 用户地址页面的路由
	beego.Router("/user/usersite",&controllers.UserInfoController{},"get:UserSiteShow;post:UserSiteHandle")
	// 商品详情页
	beego.Router("/goodsinfo",&controllers.IndexController{},"get:ShowGoodsInfo")
	// 商品列表页
	beego.Router("/list",&controllers.ListController{},"get:ShowList")

	// 商品搜索
	beego.Router("/goodssearch",&controllers.IndexController{},"post:HandleGoodsSearch")

	// 添加购物车
	beego.Router("/user/addcart",&controllers.CartController{},"post:HandleAddCart")

	// 展示购物车数据
	beego.Router("/user/mycart",&controllers.CartController{},"get:ShowCart")

	// 更新购物车数据
	beego.Router("/user/cartUpdate",&controllers.CartController{},"post:HandleCartUpdate")

	// 删除购物车数据
	beego.Router("/user/cartdelete",&controllers.CartController{},"post:HandleCartDelete")

	// 订单页面
	beego.Router("/user/order",&controllers.OrderController{},"post:ShowOrder")

	// 添加订单
	beego.Router("/user/addorder",&controllers.OrderController{},"post:HandleAddOrder")

	// 付款
	beego.Router("/user/gopay",&controllers.OrderController{},"get:HandleGoPay")
}
