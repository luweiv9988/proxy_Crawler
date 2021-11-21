package main

import (
	"log"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/gocolly/colly"
	rds "github.com/luweiv9988/go_redis"
	cron "github.com/robfig/cron/v3"
)

// Agent 为模拟浏览器客户端
const Agent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.69 Safari/537.36 Edg/95.0.1020.44"

var (
	// TargetURI 固定请求参数
	TargetURI = "https://www.kuaidaili.com/free/inha"

	// DomainName 限定域名区域
	DomainName = "www.kuaidaili.com"
)

func main() {

	// Colly 执行顺序
	// OnRequest 请求发出之前调用
	// OnError 请求过程中出现Error时调用
	// OnResponse 收到response后调用
	// OnHTML 如果收到的内容是HTML，就在onResponse执行后调用
	// OnXML 如果收到的内容是HTML或者XML，就在onHTML执行后调用
	// OnScraped OnXML执行后调用

	// Redis连接属性:
	storage := &rds.Storage{
		Address:  "127.0.0.1:6379",
		Password: "",
		DB:       0,
	}

	// 限定访问域名
	c := colly.NewCollector(
		// colly.Async(true), //异步抓取
		colly.AllowedDomains(DomainName),
		colly.UserAgent(Agent),
		colly.MaxDepth(2), //爬取深度
	)

	c.OnHTML("#listnav ul li a", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		log.Printf("Link found: %q -> %s\n", e.Text, link)
		// c.Visit(e.Request.AbsoluteURL(link))
		e.Request.Visit(link)
	})

	// 请求频率控制
	c.Limit(&colly.LimitRule{
		DomainGlob: "*.kuaidaili.*",
		// DomainRegexp: `kuaidaili\.com`,
		RandomDelay: 10 * time.Second,
		// 控制并发
		Parallelism: 1,
	})

	// 发起请求
	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL.String())
	})

	// 错误处理:
	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
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
			// fmt.Println(htmlquery.InnerText(ipaddr), htmlquery.InnerText(port))
			_ = storage.Insert(htmlquery.InnerText(ipaddr), htmlquery.InnerText(port), 0)

		}
	})

	// 关闭Redis连接
	defer storage.Close()

	c.OnScraped(func(r *colly.Response) {
		log.Println("Finished", r.Request.URL)
	})

	crontab := cron.New()
	task := func() {
		c.Visit(TargetURI)
	}
	crontab.AddFunc("*/1 * * * *", task)

	crontab.Start()

	select {}
}
