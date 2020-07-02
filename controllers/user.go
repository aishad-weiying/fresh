package controllers

import (
	"encoding/base64"
	"fresh/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/utils"
	"regexp"
	"strconv"
)

type UserController struct {
	beego.Controller
}

// 用户注册界面显示
func (this *UserController) UserRegiterShow() {
	this.TplName = "register.html"
}

// 处理用户的注册请求
func (this *UserController) UserRegisterHandle() {
	// 获取用户发送的数据
	username := this.GetString("user_name")
	passwd := this.GetString("pwd")
	cpasswd := this.GetString("cpwd")
	email := this.GetString("email")
	// 对获取的数据进行校验
	if username == "" || passwd == "" || cpasswd == "" || email == "" {
		this.Data["errmsg"] = "输入的信息不完整，请重新输入"
		this.TplName = "register.html"
		return
	}
	// 判断用户两次输入的密码是否一致
	if passwd != cpasswd {
		this.Data["errmsg"] = "两次密码输入不一致，请重新输入"
		this.TplName = "register.html"
		return
	}
	// 判断用户的邮箱，使用正则
	rex_email := `\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*`
	// 解析正则表达式
	reg , _ := regexp.Compile(rex_email)
	res := reg.FindString(email)
	if res == "" {
		this.Data["errmsg"] = "邮箱格式不正确"
		this.TplName = "register.html"
		return
	}

	// 处理数据，插入数据库
	db := orm.NewOrm()

	user := models.User{}

	user.Name = username
	user.PassWord = passwd
	user.Email = email
	// 校验用户名是否重名
	err := db.Read(&user,"Name")
	if err != orm.ErrNoRows {
		this.Data["errmsg"] = "用户以存在，请重新注册！"
		this.TplName = "register.html"
		return
	}
	// 如果用户不存在的话，插入数据库
	_ ,err = db.Insert(&user)
	if err != nil {
		this.Data["errmsg"] = "数据插入失败，请重新注册"
		this.TplName = "register.html"
		return
	}
	// 发送用于用户激活的邮件
	// 指定发送邮件的配置信息
	config := `{"username":"2286416563@qq.com","password":"授权码","host":"smtp.qq.com","port":587}`
	// 根据配置信息，创建指定的email 对象
	temail := utils.NewEMail(config)
	// 通过EMAIL对象中的属性。指定，发件人邮箱，收件人邮箱，邮件标题，以及邮件的内容。
	temail.From = "2286416563@qq.com"
	temail.To = []string{email}
	temail.Subject = "天天生鲜用户激活"
	temail.HTML = "复制该连接到浏览器中激活: 127.0.0.1/active?id="+strconv.Itoa(user.Id)
	// 发送邮件
	err = temail.Send()
	if err != nil{
		this.Data["errmsg"] = "发送激活邮件失败，请重新注册！"
		this.TplName = "register.html"
		return
	}
	this.Ctx.WriteString("注册成功，请前往邮箱激活!")

}

// 用户激活
func (this *UserController) UserActive() {
	// 获取用户id
	id ,err := this.GetInt("id")
	if err !=nil{
		this.Data["errmsg"] = "激活路径不正确，请重新确定之后登陆！"
		this.TplName = "login.html"
		return
	}
	// 根据获取的id 查询数据库
	db := orm.NewOrm()

	user := models.User{}

	user.Id = id

	err = db.Read(&user,"Id")
	if err != nil{
		this.Data["errmsg"] = "激活路径不正确，请重新确定之后登陆！"
		this.TplName = "login.html"
	}
	// 将active字段改为激活，并更新
	user.Active = true
	_ , err = db.Update(&user)
	if err != nil{
		this.Data["errmsg"] = "激活失败，请重新确定之后登陆！"
		this.TplName = "login.html"
	}
	this.Redirect("/login",302)
}

// 用户登录界面展示
func (this *UserController) UserLogin() {
	// 实现查询是否存在cookie
	username := this.Ctx.GetCookie("username")
	// 解码
	temp , _:= base64.StdEncoding.DecodeString(username)

	if string(temp) == ""{
		this.Data["username"] = ""
		this.Data["checked"] = ""
	}else {
		this.Data["username"] = string(temp)
		this.Data["checked"] = "checked"
	}

	this.TplName = "login.html"
}

// 用户登录处理
func (this *UserController) UserLoginHandle() {
	// 获取数据
	username := this.GetString("username")
	passwd := this.GetString("pwd")
	// 判断数据
	if username == "" || passwd == "" {
		this.Data["errmsg"] = "用户名和密码不能为空，请重新输入"
		this.TplName = "login.html"
		return
	}

	// 到数据库中查询用户名是否存在
	db := orm.NewOrm()
	user := models.User{}

	user.Name = username
	// 首先判断用户名是否存在
	err := db.Read(&user,"Name")
	if err != nil {
		this.Data["errmsg"] = "用户名或密码错误，请重新输入"
		this.TplName = "login.html"
		return
	}
	// 再判断密码是否正确
	if user.PassWord != passwd {
		this.Data["errmsg"] = "用户名或密码错误，请重新输入"
		this.TplName = "login.html"
		return
	}
	// 判断用户是否激活
	if user.Active == false {
		this.Data["errmsg"] = "用户尚未激活，请先到邮箱中激活"
		this.TplName = "login.html"
		return
	}

	// 用户登录成功，创建cookie
	// 首先判断是否点击了记住用户名
	remember := this.GetString("remember")
	if remember == "on" {
		// 使用base64加密 实现可以使用cookie存储中文
		temp := base64.StdEncoding.EncodeToString([]byte(username))
		this.Ctx.SetCookie("username",temp,3600 * 24 * 30)
	}else {
		this.Ctx.SetCookie("username",username,-1)
	}

	//登录成功后添加session
	this.SetSession("username",user.Name)
	// 返回视图
	this.Redirect("/",302)

}

// 用户退出登录
func (this *UserController) UserLogout() {
	// 删除session
	this.DelSession("username")
	// 返回视图
	this.Redirect("/",302)
}
