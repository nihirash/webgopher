# webgopher

Quick 'n Dirty prototype of a Web to Gopher proxy service that allows you to
access the greater World-Wide-Web via the GOpher protocol by proxying the URL
selected by the selector to the web and converting the content so something
legible for Gopher clients.

**NB:** This is very much work-in-progress.

## Install

```#!sh
$ go get githubcom/prologic/webgopher
```

or to run locally from the Git repo:

```#!sh
$ go run main.go
```

## Usage

Run the `webgopher` daemon:

```#!sh
$ webgopher
```

Use your favorite Gopher client and pass in the URL you wish to browse on the
WEB as the selector:

```#!sh
$ lynx gopher://localhost:7000/1www.wikipedia.org/
```

![Screenshot](/screenshot.png)

## Using an upstream proxy

If you need the HTTP or HTTPS requests from `webgopher` to
go through a proxy, set the `http_proxy` environment
variable before running `webgopher`.  This can be used for
example to make [Web Adjuster](http://ssb22.user.srcf.net/adjuster)
modify the pages first (you might want to use the Web
Adjuster parameters `--real-proxy` and `--just-me`, and
perhaps `--js-interpreter` to collect output from Javascript),
but if you want to adjust HTTPS pages in this way, then you
must run `webgopher` with `-no-security` so that the TLS
certificates will not be checked.  Do not do this by default.

## License

webgopher is licensed under the terms of the MIT License.
