package controllers

import (
	"dailyFresh/models"
	"encoding/base64"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/utils"
	"regexp"
	"strconv"
)

type UserController struct {
	beego.Controller
}

// 展示注册页
func (u *UserController) ShowRegister() {
	u.TplName = "register.html"
}

// 处理注册操作
func (u *UserController) HandleRegister() {
	// 1. 获取数据
	userName := u.GetString("user_name")
	userPwd := u.GetString("pwd")
	userCpwd := u.GetString("cpwd")
	userEmail := u.GetString("email")
	// 2. 检验数据
	if userName == "" || userPwd == "" || userCpwd == "" || userEmail == "" {
		u.Data["errMsg"] = "数据不完整, 请重新注册"
		u.TplName = "register.html"
		return
	}
	if userPwd != userCpwd {
		u.Data["errMsg"] = "两次输入密码不一致, 请重新注册"
		u.TplName = "register.html"
		return
	}

	//reg, _ := regexp.Compile(`^[A-Za-z0-9\u4e00-\u9fa5]+@[a-zA-Z0-9_-]+(\\.[a-zA-Z0-9_-]+)+$`)  ?? 异常panic ??
	reg, _ := regexp.Compile("^[A-Za-z0-9\u4e00-\u9fa5]+@[a-zA-Z0-9_-]+(\\.[a-zA-Z0-9_-]+)+$")
	isCorrect := reg.FindString(userEmail)
	if isCorrect == "" {
		u.Data["errMsg"] = "邮箱格式错误!"
		u.TplName = "register.html"
		return
	}

	// 3. 处理数据
	var user models.User
	user.Name = userName
	user.Password = userPwd
	user.Email = userEmail
	_, err := orm.NewOrm().Insert(&user)
	if err != nil {
		u.Data["errMsg"] = "注册失败!"
		u.TplName = "register.html"
		return
	}

	// 发送邮件
	emailConfig := `{
		"username":"854848162@qq.com",
		"password":"uuqutaqslhgrbbie",
		"host":"smtp.qq.com",
		"port":587
	}`
	emailConn := utils.NewEMail(emailConfig)
	emailConn.From = "854848162@qq.com"
	emailConn.To = []string{userEmail}
	emailConn.Subject = "天天生鲜用户注册"
	// 注意这里我们发送给用户的是激活请求地址
	// Text和HTML只会显示其中一个
	emailConn.Text = "欢迎注册天天生鲜系统"
	emailConn.HTML = "复制该链接到浏览器中激活：192.168.3.11:8080/active?id=" + strconv.Itoa(user.Id)
	err = emailConn.Send()
	if err != nil {
		u.Data["errMsg"] = "发送激活邮件失败, 请重新注册 " + err.Error()
		u.TplName = "register.html"
		return
	}

	// 4. 返回视图
	u.Ctx.WriteString("注册成功, 请去邮箱中激活用户!")
}

// 激活处理
func (u *UserController) ActiveUser() {
	id, err := u.GetInt("id")
	if err != nil {
		u.Data["errMsg"] = "激活用户不存在"
		u.TplName = "register.html"
		return
	}

	o := orm.NewOrm()
	var user models.User
	user.Id = id
	err = o.Read(&user)
	if err != nil {
		u.Data["errMsg"] = "激活用户不存在"
		u.TplName = "register.html"
		return
	}

	user.IsActive = true
	_, err = o.Update(&user, "IsActive")
	if err != nil {
		u.Data["errMsg"] = "激活失败! " + err.Error()
		u.TplName = "register.html"
		return
	}

	u.Redirect("/login", 302)
}

// 展示登录页
func (u *UserController) ShowLogin() {
	temp := u.Ctx.GetCookie("userName")
	// 解密
	userName, _ := base64.StdEncoding.DecodeString(temp)
	if string(userName) == "" {
		u.Data["userName"] = ""
		u.Data["checked"] = ""
	} else {
		u.Data["userName"] = string(userName)
		u.Data["checked"] = "checked"
	}
	u.TplName = "login.html"
}

// 处理登录操作
func (u *UserController) HandleLogin() {
	userName := u.GetString("username")
	userPwd := u.GetString("pwd")
	if userName == "" || userPwd == "" {
		u.Data["errMsg"] = "登录数据不完整, 请重新输入!"
		u.TplName = "login.html"
		return
	}

	var user models.User
	user.Name = userName
	err := orm.NewOrm().Read(&user, "Name")
	if err != nil {
		u.Data["errMsg"] = "用户名或密码错误, 请重新输入!"
		u.TplName = "login.html"
		return
	}
	if user.Password != userPwd {
		u.Data["errMsg"] = "用户名或密码错误, 请重新输入!"
		u.TplName = "login.html"
		return
	}

	if !user.IsActive {
		u.Data["errMsg"] = "用户未激活, 请前往邮箱激活!"
		u.TplName = "login.html"
		return
	}

	remember := u.GetString("remember")
	if remember == "on" {
		// cookie不能存中文, 硬存对应字段为空, 遇到中文将其转换成base64再存储
		temp := base64.StdEncoding.EncodeToString([]byte(userName))
		u.Ctx.SetCookie("userName", temp, 3600 * 24)
	} else {
		u.Ctx.SetCookie("userName", userName, -1)
	}

	// 保存到session中
	userInfo := map[string]string{"userName":user.Name, "userId":strconv.Itoa(user.Id)}
	u.SetSession("userInfo", userInfo)

	// 跳转homePage
	// u.Ctx.WriteString("登录成功")
	u.Redirect("/", 302)
}

// 退出操作
func (u *UserController) Logout() {
	u.DelSession("userInfo")
	u.Redirect("/", 302)
}

// 用户中心
func (u *UserController) ShowUserInfo() {
	u.Layout = "user_center_layout.html"
	u.TplName = "user_center_info.html"
}

// 用户订单信息
func (u *UserController) ShowUserOrder() {
	u.Layout = "user_center_layout.html"
	u.TplName = "user_center_order.html"
}

// 用户地址信息
func (u *UserController) ShowUserAddr() {
	u.Layout = "user_center_layout.html"
	u.TplName = "user_center_site.html"
}

// 获取session中的用户信息
func GetUserInfo(c *beego.Controller) (u map[string]string) {
	userInfo := c.GetSession("userInfo")
	u, ok := userInfo.(map[string]string)
	if userInfo == nil || !ok {
		c.Data["userName"] = ""
	} else {
		c.Data["userName"] = u["userName"]
	}

	return
}