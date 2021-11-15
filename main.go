package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/gocolly/colly"
	"github.com/gocolly/redisstorage"
)

// Agent 为模拟浏览器客户端
const Agent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36 Edg/95.0.1020.44"

var (
	// TargetURI 固定请求参数
	TargetURI = "https://www.kuaidaili.com/free/inha/1"

	// DomainName 限定域名区域
	DomainName = "www.kuaidaili.com"
)

func main() {

	// 限定访问域名
	c := colly.NewCollector(
		colly.AllowedDomains(DomainName),
		colly.UserAgent(Agent),
	)

	// 错误处理:
	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	// 发起请求
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	// 请求频率控制
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
	})

	//响应处理
	c.OnResponse(func(r *colly.Response) {
		doc, err := htmlquery.Parse(strings.NewReader(string(r.Body)))
		if err != nil {
			log.Fatal(err)
		}
		nodes := htmlquery.Find(doc, `//tbody/tr`)
		for _, node := range nodes {
			ipaddr := htmlquery.FindOne(node, "./td[1]")
			port := htmlquery.FindOne(node, "./td[2]")
			fmt.Println(htmlquery.InnerText(ipaddr), htmlquery.InnerText(port))

		}
	})

	// Redis连接属性:
	storage := &redisstorage.Storage{
		Address:  "127.0.0.1:6379",
		Password: "",
		DB:       0,
		// Prefix:   "httpbin_test",
	}

	err := c.SetStorage(storage)
	if err != nil {
		panic(err)
	}

	// 关闭Redis连接
	defer storage.Client.Close()

	c.Visit(TargetURI)
}
