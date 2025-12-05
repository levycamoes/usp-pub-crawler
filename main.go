package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// --- CONFIG ---
const (
	URL_BASE       = "https://uspdigital.usp.br"
	URL_LOGIN      = "https://uspdigital.usp.br/jupiterweb/webLogin.jsp"
	URL_AUTH       = "https://uspdigital.usp.br/jupiterweb/autenticar"
	URL_LISTA_PAGE = "https://uspdigital.usp.br/jupiterweb/beneficioBolsaUnificadaListar?codmnu=6684"
	URL_DWR_API    = "https://uspdigital.usp.br/jupiterweb/dwr/call/plaincall/BeneficioBolsaUnificadaControleDWR.listarBeneficioBolsaUnificada.dwr"

	USER     = "username" // <--- Fill with your USP login
	PASSWORD = "password" // <--- Fill with your USP unique password
	YEAR     = "2015"     // <--- Year of the scholarship call
)

func main() {
	// Config CSV
	file, err := os.Create("bolsas_pub_" + YEAR + ".csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"Ano", "Unidade", "Título", "Vertente", "Bolsas"})

	// Config Collector with increased TIMEOUT
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36"),
		colly.AllowURLRevisit(),
	)

	// Set a timeout of 3 minutes (180 seconds).
	// The default is sometimes too short for the USP database.
	c.SetRequestTimeout(180 * time.Second)

	// Advanced transport configuration to avoid TCP breaks
	c.WithTransport(&http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   180 * time.Second,
			KeepAlive: 180 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	})

	// Session variables for DWR
	var scriptSessionId string

	// Error handling
	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("❌ Error in request:", err)
		if r.StatusCode != 200 {
			fmt.Println("Status Code:", r.StatusCode)
		}
	})

	// Login
	c.OnHTML("form[action='autenticar']", func(e *colly.HTMLElement) {
		fmt.Println("🔐 Trying to log in...")
		// Small pause to avoid looking too much like a robot
		time.Sleep(1 * time.Second)

		err := c.Post(URL_AUTH, map[string]string{
			"codpes": USER,
			"senusu": PASSWORD,
			"Submit": "Entrar",
		})
		if err != nil {
			log.Fatal("Erro no POST de login:", err)
		}
	})

	// Monitor Login and Redirections
	c.OnResponse(func(r *colly.Response) {
		currentURL := r.Request.URL.String()

		if strings.Contains(currentURL, "autenticar") {
			fmt.Println("✅ Login accepted. Initializing session...")
			c.Visit(URL_LISTA_PAGE)

		} else if strings.Contains(currentURL, "beneficioBolsaUnificadaListar") {
			fmt.Println("🛠️ Generating DWR security tokens...")
			scriptSessionId = prepararSessaoDWR(c, r)

			fmt.Println("📡 Sending data request (This may take up to 3 minutes)...")
			dispararDWR(c, writer, scriptSessionId)
		}
	})

	fmt.Println("🚀 Starting crawler (Timeout set to 3 min)...")
	c.Visit(URL_LOGIN)
}

// This function simulates what the DWR JavaScript would do in the browser
func prepararSessaoDWR(c *colly.Collector, r *colly.Response) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	fakeDwrSessionId := string(b)

	cookieDwr := &http.Cookie{
		Name:   "DWRSESSIONID",
		Value:  fakeDwrSessionId,
		Path:   "/jupiterweb",
		Domain: "uspdigital.usp.br",
	}
	c.SetCookies(URL_BASE, []*http.Cookie{cookieDwr})

	randToken := strconv.Itoa(rand.Intn(999999))
	return fakeDwrSessionId + "/" + randToken
}

func dispararDWR(c *colly.Collector, writer *csv.Writer, scriptSessionId string) {
	dwrCollector := c.Clone()

	// Apply the same timeout to the clone
	dwrCollector.SetRequestTimeout(180 * time.Second)

	dwrCollector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Content-Type", "text/plain")
		r.Headers.Set("Origin", "https://uspdigital.usp.br")
		r.Headers.Set("Referer", URL_LISTA_PAGE)
	})

	dwrCollector.OnResponse(func(r *colly.Response) {
		body := string(r.Body)
		if strings.Contains(body, "error") || strings.Contains(body, "Exception") {
			fmt.Println("❌ Error returned by server:", body[:min(len(body), 200)])
			return
		}

		parseDWRResponse(body, writer)
	})

	dwrCollector.OnError(func(r *colly.Response, err error) {
		fmt.Println("❌ FATAL error in DWR:", err)
	})

	payload := strings.Join([]string{
		"callCount=1",
		"nextReverseAjaxIndex=0",
		"c0-scriptName=BeneficioBolsaUnificadaControleDWR",
		"c0-methodName=listarBeneficioBolsaUnificada",
		"c0-id=0",
		"c0-param0=string:" + YEAR,
		"c0-param1=string:",
		"c0-param2=string:",
		"c0-param3=string:",
		"c0-param4=boolean:false",
		"batchId=1",
		"instanceId=0",
		"page=%2Fjupiterweb%2FbeneficioBolsaUnificadaListar%3Fcodmnu%3D6684",
		"scriptSessionId=" + scriptSessionId,
		"",
	}, "\n")

	err := dwrCollector.PostRaw(URL_DWR_API, []byte(payload))
	if err != nil {
		fmt.Println("❌ Error sending request:", err)
	}
}

func parseDWRResponse(body string, writer *csv.Writer) {
	start := strings.Index(body, "[{")
	end := strings.LastIndex(body, "}]")

	if start == -1 || end == -1 {
		fmt.Println("⚠️ No records found or empty response.")
		return
	}

	rawArray := body[start : end+2]
	reObj := regexp.MustCompile(`\{[\s\S]*?\}`)
	objetos := reObj.FindAllString(rawArray, -1)

	reTitulo := regexp.MustCompile(`titprjbnf:\s*"(.*?)"`)
	reUnidade := regexp.MustCompile(`nomabvclg:\s*"(.*?)"`)
	reVertente := regexp.MustCompile(`stavteprj:\s*"(.*?)"`)
	reAno := regexp.MustCompile(`anoofebnf:\s*"(.*?)"`)
	reBolsas := regexp.MustCompile(`numbolapr:\s*(\d+)`)

	count := 0
	for _, obj := range objetos {
		ano := unquoteUnicode(extract(reAno, obj))
		unidade := unquoteUnicode(extract(reUnidade, obj))
		titulo := unquoteUnicode(extract(reTitulo, obj))
		vertente := unquoteUnicode(extract(reVertente, obj))
		bolsas := extract(reBolsas, obj)

		writer.Write([]string{ano, unidade, titulo, vertente, bolsas})
		count++
	}
	writer.Flush()
	fmt.Printf("✅ SUCCESS! %d scholarships saved.\n", count)
}

// Utilities
func extract(re *regexp.Regexp, text string) string {
	match := re.FindStringSubmatch(text)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func unquoteUnicode(str string) string {
	if str == "" {
		return ""
	}
	s, err := strconv.Unquote(`"` + str + `"`)
	if err != nil {
		return str
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

