package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/prologic/go-gopher"
	log "github.com/sirupsen/logrus"
	"jaytaylor.com/html2text"
	"golang.org/x/text/encoding/charmap"
)

type proxy struct{}

var HostTabPort string // we'll need this for links

func ChunkString(s string, chunkSize int) []string {
    var chunks []string
    runes := []rune(s)

    if len(runes) == 0 {
        return []string{s}
    }

    for i := 0; i < len(runes); i += chunkSize {
        nn := i + chunkSize
        if nn > len(runes) {
            nn = len(runes)
        }
        chunks = append(chunks, string(runes[i:nn]))
    }
    return chunks
}


func (p *proxy) ServeGopher(w gopher.ResponseWriter, r *gopher.Request) {
	log.Infof("Selector: %s", r.Selector)
	requestedURL := strings.TrimPrefix(r.Selector, "/")
	requestedURL = strings.TrimPrefix(requestedURL, "\t");

	if requestedURL == "/" || requestedURL == "" {
		page, _ := ioutil.ReadFile("request.gopher")
		w.Write(page)
		
		return
	}

	if strings.HasPrefix(requestedURL,"https://") ||
		strings.HasPrefix(requestedURL,"http://") {
		// User already specified the protocol, so we
		// don't need to add it ourselves
	} else {
		// Default to https
		requestedURL = fmt.Sprintf("https://%s", requestedURL)
	}

	res, err := http.Get(requestedURL)
	if err != nil {
		msg := fmt.Sprintf("error fetching web resource %s: %s", requestedURL, err)
		log.WithError(err).WithField("url", requestedURL).Error(msg)
		w.WriteError(msg)
		return
	}

	mime_type := res.Header.Get("Content-Type")

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		msg := fmt.Sprintf("error reading web resource body: %s", err)
		log.WithError(err).WithField("url", requestedURL).Error(msg)
		w.WriteError(msg)
		return
	}

	baseURL, err := url.Parse(requestedURL)
	if err != nil {
		// should never happen if got this far
		log.Fatal(err)
	}

	encoder := charmap.CodePage866.NewEncoder()
	html := encoder.String(string(body))

	// but it might not be HTML, if a link was followed to a
	// plain-text document or to a binary file
	if mime_type != "" && ! strings.Contains(mime_type,"html") {
		if strings.HasPrefix(mime_type, "text/") {
			// We can serve it as plain text.
			// Still need "info" lines, because we're a gophermap
			w.Write([]byte(strings.ReplaceAll("i"+html,"\n","\ni")))
		} else {
			// A binary file or something.
			// As we said we'd be a gophermap, we'd better not
			// serve this.  TODO: could create a selector of another type
			// and embed something in the selector to say we've done so
			msg := fmt.Sprintf("Not an HTML or text document: %s (MIME type is %s)", requestedURL, mime_type)
			log.Error(msg)
			w.Write([]byte(msg))
		}
		return
	}

	// If we get this far, it's HTML
	
	// Before trying to parse links, remove scripts.
	// Better do this here (not wait for html2text), because
	// some scripts write half-links which confuses us.
	// TODO: we may want to take out comments etc as well.
	html = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(html,"")

	// While we're at it, no-break space had better become a
	// normal space (or html2text may just remove the entity)
	html = strings.ReplaceAll(html,"&nbsp;"," ")

	// Now parse links
	html = regexp.MustCompile(`(?is)<a ([^>]* )?href=[^>]*>.*?</a>`).ReplaceAllStringFunc(html,func (m string) string {
		// We want the absolute URLs of links to survive html2text,
		// so we can turn them into Gopher selectors afterwards.
		// Use Markdown-style first, which html2text leaves alone.
		href := regexp.MustCompile(`(?i) href="?([^>" ]*)`).FindStringSubmatch(m)[1]
		u, err := url.Parse(href)
		
		if err != nil {
			// not a valid URL: ignore it
			return m
		}

		href2 := baseURL.ResolveReference(u)
		text := regexp.MustCompile(`(?is)[^>]*>(.*?)</a>`).FindStringSubmatch(m)[1]
		return fmt.Sprintf("[%s](%s)",text,href2) // Markdown-style, to go through html2text unchanged
	})

	text, err := html2text.FromString(html, html2text.Options{PrettyTables: true})
	if err != nil {
		msg := fmt.Sprintf("error converting html to text: %s", err)
		log.WithError(err).WithField("url", requestedURL).Error(msg)
		w.WriteError(msg)
		return
	}
	
	lines := strings.Split(text,"\n")
	outputLines := []string{};

	for _, s := range lines {
		re := regexp.MustCompile(`[[]([^]]*)[]][(]([^)]*)[)]`)

		if len(s) <= 59 || re.FindString(s) != "" {
			outputLines = append(outputLines, s)
		} else {
			tmpStrings := ChunkString(s, 59)
			
			for _, s := range tmpStrings {
				outputLines = append(outputLines, s)
			}
		}
	}

	for _,s := range outputLines {
		// Convert our Markdown links to Gopher selectors
		w.Write([]byte(strings.ReplaceAll(regexp.MustCompile(`[[]([^]]*)[]][(]([^)]*)[)]`).ReplaceAllString("\r\ni"+s+"\t-\t-\t-\r\n","\r\n1$1\t$2\t"+HostTabPort+"\r\ni"),"\ni\n","\n")[1:]))
		// TODO: wrap "i" lines at 67 characters?
		// (but beware the formatting of pre, blockquote etc)
	}
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

	i := strings.LastIndex(connectAddress, ":")
	HostTabPort = connectAddress[:i] + "\t" + connectAddress[i+1:]
	fmt.Printf("Server starting, use (e.g.) gopher://%s/1www.wikipedia.org/\n",connectAddress)
	log.Fatal(gopher.ListenAndServe(*listenAddress, &proxy{}))
}
