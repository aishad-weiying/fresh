package main

import (
	_ "fresh/routers"
	_ "fresh/models"
	"github.com/astaxie/beego"
)

func main() {
	beego.Run()
}

