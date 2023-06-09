package roddy

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"
)

var (
	serverIndexResponse = []byte("hello world\n")
	robotsFile          = `
User-agent: *
Allow: /allowed
Disallow: /disallowed
Disallow: /allowed*q=
`
)

func newUnstartedTestServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(serverIndexResponse)
	})

	mux.HandleFunc("/html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<title>Test Page</title>
</head>
<body>
<h1>Hello World</h1>
<p class="description">This is a test page</p>
<p class="description">This is a test paragraph</p>
</body>
</html>
		`))
	})

	mux.HandleFunc("/xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<page>
	<title>Test Page</title>
	<paragraph type="description">This is a test page</paragraph>
	<paragraph type="description">This is a test paragraph</paragraph>
</page>
		`))
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(r.FormValue("name")))
		}
	})

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(robotsFile))
	})

	mux.HandleFunc("/allowed", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("allowed"))
	})

	mux.HandleFunc("/disallowed", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("disallowed"))
	})

	mux.Handle("/redirect", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		destination := "/redirected/"
		if d := r.URL.Query().Get("d"); d != "" {
			destination = d
		}
		http.Redirect(w, r, destination, http.StatusSeeOther)
	}))

	mux.Handle("/redirected/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<a href="test">test</a>`)
	}))

	mux.HandleFunc("/set_cookie", func(w http.ResponseWriter, r *http.Request) {
		c := &http.Cookie{Name: "test", Value: "testv", HttpOnly: false}
		http.SetCookie(w, c)
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("/check_cookie", func(w http.ResponseWriter, r *http.Request) {
		cs := r.Cookies()
		if len(cs) != 1 || r.Cookies()[0].Value != "testv" {
			w.WriteHeader(500)
			w.Write([]byte("nok"))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("/500", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(500)
		w.Write([]byte("<p>error</p>"))
	})

	mux.HandleFunc("/user_agent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(r.Header.Get("User-Agent")))
	})

	mux.HandleFunc("/host_header", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(r.Host))
	})

	mux.HandleFunc("/custom_header", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(r.Header.Get("Test")))
	})

	mux.HandleFunc("/base", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<title>Test Page</title>
<base href="http://xy.com/" />
</head>
<body>
<a href="z">link</a>
</body>
</html>
		`))
	})

	mux.HandleFunc("/base_relative", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<title>Test Page</title>
<base href="/foobar/" />
</head>
<body>
<a href="z">link</a>
</body>
</html>
		`))
	})

	mux.HandleFunc("/tabs_and_newlines", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<title>Test Page</title>
<base href="/foo	bar/" />
</head>
<body>
<a href="x
y">link</a>
</body>
</html>
		`))
	})

	mux.HandleFunc("/foobar/xy", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
<title>Test Page</title>
</head>
<body>
<p>hello</p>
</body>
</html>
		`))
	})

	mux.HandleFunc("/100%25", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("100 percent"))
	})

	mux.HandleFunc("/large_binary", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		ww := bufio.NewWriter(w)
		defer ww.Flush()
		for {
			// have to check error to detect client aborting download
			if _, err := ww.Write([]byte{0x41}); err != nil {
				return
			}
		}
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		i := 0

		for {
			select {
			case <-r.Context().Done():
				return
			case t := <-ticker.C:
				fmt.Fprintf(w, "%s\n", t)
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				i++
				if i == 10 {
					return
				}
			}
		}
	})

	return httptest.NewUnstartedServer(mux)
}

func newTestServer() *httptest.Server {
	srv := newUnstartedTestServer()
	srv.Start()

	return srv
}
