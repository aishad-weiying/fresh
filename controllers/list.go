package controllers

import (
	"fresh/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"math"
)

type ListController struct {
	beego.Controller
}

// 分页函数
func PageTools(pageCount, pageIndex int) []int {
	// 创建切片，用来保存要显示的页码
	var pageIndexBuffer []int
	// 以 pageCount = 10 页为例
	if pageCount <= 5 {
		pageIndexBuffer = make([]int, pageCount)
		for index, _ := range pageIndexBuffer {
			pageIndexBuffer[index] = index + 1
		}
		// 页数小于5的时候，有几页就显示几页
	} else if pageIndex < 3 {
		// 前三页的时候,显示的都是 1 2 3 4 5
		pageIndexBuffer = make([]int, 5)
		for index, _ := range pageIndexBuffer {
			pageIndexBuffer[index] = index + 1
		}
		// 页数大于5，但是当前页小于3的时候，依旧显示的都是前5页
		//for 循环结果为 pageIndexBuffer = []int{1,2,3,4,5}
	} else if pageIndex > pageCount-3 {
		// 页数大于5的时候，当前页为后三页的时候，显示的都是后5页

		pageIndexBuffer = make([]int, pageCount)
		//for index,_ := range  pageIndexBuffer{
		//	pageIndexBuffer[index] = pageCount - 5 + index
		//}
		pageIndexBuffer = []int{pageCount - 4, pageCount - 3, pageCount - 2, pageCount - 1, pageCount}
	} else {
		pageIndexBuffer = make([]int, 5)
		//for index,_ := range pageIndexBuffer{
		//	pageIndexBuffer[index] = pageIndex - 3 + index
		//}
		pageIndexBuffer = []int{pageIndex - 2, pageIndex - 1, pageIndex, pageIndex + 1, pageIndex + 2}
	}
	return pageIndexBuffer
}

// 商品列表页展示
func (this *ListController) ShowList() {
	// 获取传递的id
	id, err := this.GetInt("id")
	if err != nil {
		beego.Info("传递的参数错误", err)
		this.Redirect("/", 302)
		return
	}
	//根据类型获取新品
	db := orm.NewOrm()
	// 获取同类型商品的最新两条信息
	var goodsSKU2 []models.GoodsSKU
	db.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", id).OrderBy("Time").Limit(2, 0).All(&goodsSKU2)
	this.Data["new2"] = goodsSKU2

	// 获取商品
	var goods []models.GoodsSKU
	// 使用ShowLayout 结合goodslayout.html 页面，获取所有商品类型以及用户登录判断
	ShowLayout(&this.Controller)
	this.TplName = "list.html"

	// 分页处理，定义每页显示的条目
	pageSize := 3 // 定义每页显示多少条数据
	// 获取总页数
	count, _ := db.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", id).Count()
	pageCount := math.Ceil(float64(count) / float64(pageSize))
	// 获取当前页
	pageIndex, err := this.GetInt("pageindex")
	if err != nil {
		pageIndex = 1
	}
	// 定义函数处理,获取要显示的页数的切片
	pages := PageTools(int(pageCount), int(pageIndex))
	// 传递给视图这个切片
	this.Data["pages"] = pages
	// 传递给视图商品类型的id
	this.Data["id"] = id
	// 传递给视图当前页，用来标记高亮显示当前页页码
	this.Data["pageindex"] = pageIndex
	// 根据每页显示的数据个数，查询每页应该显示的数据
	start := (pageIndex - 1) * pageSize
	// 上一页和下一页的判断
	if pageIndex <= 1 {
		preIndex := 0
		this.Data["preIndex"] = preIndex
	} else {
		preIndex := pageIndex - 1
		this.Data["preIndex"] = preIndex
	}
	if pageIndex >= int(pageCount) {
		nextIndex := -1
		this.Data["nextIndex"] = nextIndex
	} else {
		nextIndex := pageIndex + 1
		this.Data["nextIndex"] = nextIndex
	}
	//根据不同的选项获取不同排序规则
	sort := this.GetString("sort")
	if sort == "" {
		db.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", id).Limit(pageSize, start).All(&goods)
		// 将本页要显示的数据传递给视图
		this.Data["goods"] = goods
		// 把当前的排序规则传递给视图
		this.Data["sort"] = ""
	} else if sort == "price" {
		db.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", id).OrderBy("Price").Limit(pageSize, start).All(&goods)
		// 将本页要显示的数据传递给视图
		this.Data["goods"] = goods
		// 把当前的排序规则传递给视图
		this.Data["sort"] = "price"
	} else {
		db.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", id).OrderBy("Sales").Limit(pageSize, start).All(&goods)
		// 将本页要显示的数据传递给视图
		this.Data["goods"] = goods
		// 把当前的排序规则传递给视图
		this.Data["sort"] = "sale"
	}
	// 获取购物商品数量
	num := GetCartNum(&this.Controller)
	this.Data["num"] = num
}
