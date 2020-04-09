package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/prologic/go-gopher"
	log "github.com/sirupsen/logrus"
	"jaytaylor.com/html2text"
)

type proxy struct{}

func (p *proxy) ServeGopher(w gopher.ResponseWriter, r *gopher.Request) {
	log.Infof("Selector: %s", r.Selector)
	url := strings.TrimPrefix(r.Selector, "/")
	if strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "http://") {
		// User already specified the protocol, so we
		// don't need to add it ourselves
	} else {
		// Default to https
		url = fmt.Sprintf("https://%s", url)
	}

	res, err := http.Get(url)
	if err != nil {
		msg := fmt.Sprintf("error fetching web resource %s: %s", url, err)
		log.WithError(err).WithField("url", url).Error(msg)
		w.WriteError(msg)
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		msg := fmt.Sprintf("error reading web resource body: %s", err)
		log.WithError(err).WithField("url", url).Error(msg)
		w.WriteError(msg)
		return
	}

	html := string(body)
	text, err := html2text.FromString(html, html2text.Options{PrettyTables: true})
	if err != nil {
		msg := fmt.Sprintf("error converting html to text: %s", err)
		log.WithError(err).WithField("url", url).Error(msg)
		w.WriteError(msg)
		return
	}

	// TODO: Handle links
	// TODO: Write Info items
	w.Write([]byte(text))
}

func main() {
	listenAddress := flag.String("listen-address", ":7000", ":port or address:port to listen on")
	noSecurity := flag.Bool("no-security", false, "Skip checking TLS certificates")
	flag.Parse()
	if *noSecurity {
		// Don't check HTTPS certificates if -no-security is set
		// (This is for if you want to intercept and alter HTTPS connections
		// in an upstream proxy, for rewrite experiments etc)
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	connectAddress := *listenAddress
	if strings.HasPrefix(connectAddress, ":") {
		connectAddress = "localhost" + connectAddress
	}
	fmt.Printf("Server starting, use (e.g.) gopher://%s/1www.wikipedia.org/\n", connectAddress)
	log.Fatal(gopher.ListenAndServe(*listenAddress, &proxy{}))
}
