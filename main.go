package main

// https://pkg.go.dev/github.com/tebeka/selenium#example-package
// https://github.com/domainr/whois
// https://golangexample.com/dns-lookup-using-go/

/*
chandler@xenon ~/projects/hats-domains % for i in {1..50}; do                                                                                                             [21:17:08]
  dig ${i}hats.com | grep -i nxdomain >/dev/null && echo ${i}hats.com is available
done
5hats.com is available
16hats.com is available
26hats.com is available
28hats.com is available
30hats.com is available
35hats.com is available
36hats.com is available
37hats.com is available
41hats.com is available
43hats.com is available
44hats.com is available
46hats.com is available
48hats.com is available
49hats.com is available
[1] chandler@xenon ~/projects/hats-domains %
*/

import (
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tebeka/selenium"
)

type HatsSite struct {
	DomainName    string
	Owner         string
	Since         time.Time
	ScreenshotURL template.URL
	Title         string
	FetchTime     time.Time
	HTTPOpen      bool
	HTTPSOpen     bool
}

func getSites(largest int, wd selenium.WebDriver) (sites []HatsSite, err error) {
	for i := 2; i <= largest; i++ {
		domainName := fmt.Sprintf("%vhats.com", i)

		err := wd.Get(fmt.Sprintf("http://%v/", domainName))
		if err != nil {
			return sites, err
		}

		title, err := wd.Title()
		if err != nil {
			return sites, err
		}

		screenshot, err := wd.Screenshot()
		if err != nil {
			return sites, err
		}

		sites = append(sites, HatsSite{
			DomainName:    domainName,
			Owner:         "unknown",
			ScreenshotURL: template.URL(fmt.Sprintf("data:image/png;base64,%v", base64.StdEncoding.EncodeToString(screenshot))),
			Title:         title,
			FetchTime:     time.Now(),
			HTTPOpen:      false, // TODO
			HTTPSOpen:     false, // TODO
		})
	}
	return sites, nil
}

func generateHTML(sites []HatsSite, w io.Writer) error {
	tmpl, err := template.New("main").Parse(`<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>{n}hats.com domains</title>
		<style>
			body { max-width: 800px; font-family: sans-serif; margin: auto; }
			dt { font-weight: bold; }
			img { max-width: 100%; }
		</style>
	</head>
	<body>
		<h1>{n}hats.com domains</h1>
		<h3>Summary</h3>
		<ol>
			{{range .}}
			<li><a href="#{{.DomainName}}">{{.DomainName}}</a> &ndash; Registered since TODO</li>
			{{end}}
		</ol>
		{{range .}}
		<h3 id="{{.DomainName}}">{{.DomainName}}</h3>
		<dl>
			<dt>Fetched</dt>
			<dd>{{.FetchTime.Format "Mon Jan 2 15:04:05 -0700 MST 2006" }}</dd>
			<dt>Title</dt>
			<dd>{{.Title}}</dd>
			<dt>Owner</dt>
			<dd>{{.Owner}}</dd>
			<dt>Since</dt>
			<dd>{{.Since.Format "Mon Jan 2 2006" }}</dd>
		</dl>
		<img src="{{.ScreenshotURL}}" alt="screenshot of {{.DomainName}} as of {{.FetchTime.Format "Mon Jan 2 15:04:05 -0700 MST 2006" }}">
		{{end}}
	</body>
	</html>`)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, sites)
}

func main() {
	serve := flag.Bool("serve", false, "Serve HTTP rather than writing a file")
	filename := flag.String("path", "index.html", "Output filename (if -serve=false, default)")
	port := flag.Int("port", 8080, "Port to serve on")
	largest := flag.Int("largest", 50, "largest n for {n}hats.com")
	flag.Parse()

	const (
		geckoDriverPath = "deps/geckodriver"
		geckoDriverPort = 8080
	)

	opts := []selenium.ServiceOption{
		selenium.StartFrameBuffer(),
		selenium.Output(nil),
	}
	selenium.SetDebug(true)

	service, err := selenium.NewGeckoDriverService(geckoDriverPath, geckoDriverPort, opts...)
	if err != nil {
		panic(err)
	}
	defer service.Stop()

	wd, err := selenium.NewRemote(nil, fmt.Sprintf("http://localhost:%d", geckoDriverPort))
	if err != nil {
		panic(err)
	}
	defer wd.Quit()

	if *serve {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// err := generateHTML(w, *largest, wd)
			// if err != nil {
			// 	w.Write([]byte(err.Error()))
			// }
		})
		fmt.Printf("Serving on %v\n", port)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", *port), nil))
	} else {
		file, err := os.Create(*filename)
		if err != nil {
			panic(err)
		}
		sites, err := getSites(*largest, wd)
		if err != nil {
			fmt.Println(err)
		}
		err = generateHTML(sites, file)
		if err != nil {
			fmt.Println(err)
		}
	}
}
