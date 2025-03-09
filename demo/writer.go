package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/nipuntalukdar/rollingwriter"
)

func main() {
	// writer 实现了 io.Writer 的全部接口
	// 使用配置方式生成一个 writer 或者 Option 都可以
	config := rollingwriter.Config{
		LogPath:       "./log",        //日志路径
		TimeTagFormat: "060102150405", //时间格式串
		FileName:      "test",         //日志文件名
		MaxRemain:     5,              //配置日志最大存留数

		// 目前有2中滚动策略: 按照时间滚动按照大小滚动
		// - 时间滚动: 配置策略如同 crontable, 例如,每天0:0切分, 则配置 0 0 0 * * *
		// - 大小滚动: 配置单个日志文件(未压缩)的滚动大小门限, 如1G, 500M
		RollingPolicy:      rollingwriter.TimeRolling, //配置滚动策略 norolling timerolling volumerolling
		RollingTimePattern: "* * * * * *",             //配置时间滚动策略
		RollingVolumeSize:  "20k",                      //配置截断文件下限大小

		// Compress will compress log file with gzip
		Compress: true,
	}

	// 创建一个 writer
	writer, err := rollingwriter.NewWriterFromConfig(&config)
	if err != nil {
		// 应该处理错误
		panic(err)
	}

	// 并发读写即可
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(w *sync.WaitGroup) {
			for j := 0; j < 10000 ; j++{
				fmt.Fprintf(writer, "now the time is given here :%s \n", time.Now())
				time.Sleep(10 * time.Millisecond)
			}
			w.Done()
		}(&wg)
	}
	wg.Wait()
}
