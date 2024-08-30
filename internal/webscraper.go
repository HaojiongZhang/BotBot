package util


import (
    "fmt"
    "net/http"
    "strings"
	"io"
	"unicode"

    "github.com/PuerkitoBio/goquery"
)


func scrapeArxiv(url string) (string, string, string, error) {
	url = strings.TrimSpace(url)
	url = strings.TrimSuffix(url, ",")

    resp, err := http.Get(url)
    if err != nil {
        return "", "", "", fmt.Errorf("error fetching data: %v", err)
    }
    defer resp.Body.Close()

    // Check if the response status is OK (200)
    if resp.StatusCode != http.StatusOK {
        return "", "", "", fmt.Errorf("error: status code %d", resp.StatusCode)
    }

    // Load the HTML document
    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        return "", "", "", fmt.Errorf("error loading document: %v", err)
    }

    // headers := []string{}
    paragraphs := []string{}
    abstract := ""

    // Extract headers (h1, h2, h3, h4, h5, h6)
    // doc.Find("h1").Each(func(i int, s *goquery.Selection) {
    //     headers = append(headers, s.Text())
    // })
	title := doc.Find("title").First().Text()
    if title == "" {
        title = doc.Find("h1").First().Text()
    }
    if title == "" {
        title = doc.Find("h2").First().Text()
    }

    // Extract paragraphs
    doc.Find("p").EachWithBreak(func(i int, s *goquery.Selection) bool {
        paragraphs = append(paragraphs, s.Text())
        return i < 9 // Stop after 10 paragraphs (0-9)
    })

    // Extract the abstract specifically
    doc.Find("blockquote.abstract").Each(func(i int, s *goquery.Selection) {
        abstract = s.Text()
    })

    // combinedHeaders := strings.Join(headers, "\n\n")
    combinedParagraphs := strings.Join(paragraphs, "\n\n")

	PrintDebug(strings.TrimSpace(abstract))
	title = strings.TrimLeftFunc(title, func(r rune) bool {
        return !unicode.IsLetter(r)
    })

    return title, strings.TrimSpace(abstract), combinedParagraphs, nil
}

func stringDiff(a, b string) string {
    if len(a) > len(b) {
        a, b = b, a
    }
    var diff strings.Builder
    for i := 0; i < len(a); i++ {
        if a[i] != b[i] {
            diff.WriteString(fmt.Sprintf("Pos %d: '%c' vs '%c'\n", i, a[i], b[i]))
        }
    }
    if len(a) != len(b) {
        diff.WriteString(fmt.Sprintf("Length difference: %d vs %d\n", len(a), len(b)))
    }
    return diff.String()
}

// WebScraper function (to be implemented)
func WebScraper(url string) (string, string, error) {
	PrintDebug("Input URL: "+ url)
	title, abstract, paragraph, err := scrapeArxiv(url)
	if err != nil{
		return "", "", fmt.Errorf("error scraping arXiv: %w", err)
	}
	if len(abstract) == 0{
		return title, paragraph, nil
	}
	return title, abstract, nil
}

func jinaScrapper(url string) (string, error) {

	fullURL := "https://r.jina.ai/" + url

	response, err := http.Get(fullURL)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	PrintDebug("jina response: " + string(body))
	return string(body), nil
}