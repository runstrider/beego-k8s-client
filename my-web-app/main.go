package main

import (
	_ "beego/my-web-app/routers"
	"github.com/astaxie/beego"
)

func main() {
	//关闭模板输出
	beego.AutoRender = false
	//开启session
	beego.BConfig.WebConfig.Session.SessionOn = true
	beego.Run()
}

