package routers

import (
	"beego/my-web-app/controllers"
	"github.com/astaxie/beego"
)

func init() {
    beego.Router("/", &controllers.MainController{})
    beego.Router("/k8s", &controllers.K8sController{})
}
