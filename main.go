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
	_ "embed"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"github.com/tebeka/selenium"
)

type HatsSite struct {
	DomainName    string
	Available     bool
	FetchTime     time.Time
	DomainInfo    *whoisparser.Domain
	Registrar     *whoisparser.Contact
	Registrant    *whoisparser.Contact
	ScreenshotURL template.URL
	Title         string
	Notes         template.HTML
}

func getSites(largest int, wd selenium.WebDriver) (sites []HatsSite, err error) {
	// TODO: 1hat.com 0hats.com, hats.com; possibly onehat.com, twohats.com, etc
	for i := 2; i <= largest; i++ {
		hatsSite := HatsSite{
			DomainName: fmt.Sprintf("%dhats.com", i),
			FetchTime:  time.Now(),
		}
		log.Printf("Retrieving info for %v\n", hatsSite.DomainName)

		// Check if domain is registered
		query_result, err := whois.Whois(hatsSite.DomainName)
		if err != nil {
			return sites, err
		}
		result, err := whoisparser.Parse(query_result)
		if err == whoisparser.ErrNotFoundDomain {
			hatsSite.Available = true
			hatsSite.Notes = template.HTML(fmt.Sprintf("Register at <a href='https://www.namecheap.com/domains/registration/results/?domain=%v'>NameCheap</a>", hatsSite.DomainName))
			sites = append(sites, hatsSite)
			continue
		} else if err != nil {
			return sites, err
		}
		hatsSite.Available = false
		hatsSite.DomainInfo = result.Domain
		hatsSite.Registrar = result.Registrar
		hatsSite.Registrant = result.Registrant

		_, err = net.LookupHost(hatsSite.DomainName)
		if err != nil {
			hatsSite.Notes = template.HTML(fmt.Sprintf("DNS Error: <code>%v</code>", err.Error()))
			sites = append(sites, hatsSite)
			continue
		}

		// Get web page, take screenshot
		err = wd.Get(fmt.Sprintf("http://%v/", hatsSite.DomainName))
		if err != nil {
			return sites, err
		}

		hatsSite.Title, err = wd.Title()
		if err != nil {
			return sites, err
		}

		screenshot, err := wd.Screenshot()
		if err != nil {
			return sites, err
		}
		hatsSite.ScreenshotURL = template.URL(fmt.Sprintf("data:image/png;base64,%v", base64.StdEncoding.EncodeToString(screenshot)))

		sites = append(sites, hatsSite)
	}
	return sites, nil
}

//go:embed template.html
var rawtmpl string

func generateHTML(sites []HatsSite, w io.Writer) error {
	funcs := template.FuncMap{
		"parseTime": func(s string) time.Time { time, _ := time.Parse(time.RFC3339, s); return time },
	}
	tmpl, err := template.New("main").Funcs(funcs).Parse(rawtmpl)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, sites)
}

func main() {
	serve := flag.Bool("serve", false, "Serve HTTP rather than writing a file")
	filename := flag.String("path", "index.html", "Output filename (if -serve=false, default)")
	port := flag.Int("port", 8080, "Port to serve on")
	debug := flag.Bool("debug", false, "Enable debug logging for the selenium package")
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
	selenium.SetDebug(*debug)

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
