package settings

import (
      "os"
      "fmt"
)

var Storage = "/var/lib/cucurbita/"
var Address = ":80"

func init() {

        args := os.Args

	// 遍历命令行参数
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-d":
			// 如果命令行参数包含 "-d"，则设置环境变量 CUCURBITA_STORAGE
			if i+1 < len(args) {
				os.Setenv("CUCURBITA_STORAGE", args[i+1])
				i++ 
			} else {
				fmt.Println("缺少 -d 参数的数据库存储目录,例如：/var/lib/cucurbita/ ")
				os.Exit(1)
			}
		case "-p":
			// 如果命令行参数包含 "-p"，则设置环境变量 CUCURBITA_ADDRESS
			if i+1 < len(args) {
				os.Setenv("CUCURBITA_ADDRESS", args[i+1])
				i++ 
			} else {
				fmt.Println("缺少 -p 参数的端口值例如： :8888")
				os.Exit(1)
			}
		}
	}
	if value := os.Getenv("CUCURBITA_STORAGE"); len(value) != 0 {
		Storage = value
	}
	if value := os.Getenv("CUCURBITA_ADDRESS"); len(value) != 0 {
		Address = value
	}
}
