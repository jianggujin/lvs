package main

import (
	"fmt"
	"time"
)

func main() {
	timeZone, _ := time.LoadLocation("Asia/Shanghai")
	fmt.Print(time.Now().In(timeZone).Format("2006-01-02"))
}
