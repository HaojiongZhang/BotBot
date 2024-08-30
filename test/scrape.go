package main

import (
    "fmt"
    "io"
    "io/ioutil"
    "net/http"
    "os"
    "strings"

    "github.com/PuerkitoBio/goquery"
    "github.com/pdfcpu/pdfcpu/pkg/api"
    "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Function to scrape headers, paragraphs, and abstracts from a given arXiv URL
func scrapeArxiv(url string) (string, string, string, error) {
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

    return title, strings.TrimSpace(abstract), combinedParagraphs, nil
}

// Function to download a PDF file from a given URL
func downloadPDF(pdfURL string, filename string) error {
    // Make an HTTP GET request to download the PDF
    resp, err := http.Get(pdfURL)
    if err != nil {
        return fmt.Errorf("error downloading PDF: %v", err)
    }
    defer resp.Body.Close()

    // Check if the response status is OK (200)
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("error: status code %d", resp.StatusCode)
    }

    // Create the file
    out, err := os.Create(filename)
    if err != nil {
        return fmt.Errorf("error creating file: %v", err)
    }
    defer out.Close()

    // Copy the response body to the file
    _, err = io.Copy(out, resp.Body)
    if err != nil {
        return fmt.Errorf("error copying response to file: %v", err)
    }

    return nil
}

// Function to read the first page of a PDF file
func readFirstPagePDF(filename string) error {
    // Read the PDF file and extract the first page
    file, err := os.Open(filename)
    if err != nil {
        return fmt.Errorf("error opening PDF: %v", err)
    }
    defer file.Close()

    config := model.NewDefaultConfiguration()
    outputPath := "firstpage.pdf"
    pages := []string{"1"}
    err = api.ExtractPages(file, filename, outputPath, pages, config)
    if err != nil {
        return fmt.Errorf("error extracting first page from PDF: %v", err)
    }
    // Read the extracted first page
    text, err := ioutil.ReadFile("firstpage.pdf")
    if err != nil {
        return fmt.Errorf("error reading first page PDF: %v", err)
    }

    // Print the extracted text
    fmt.Println("First Page Text:")
    fmt.Println(string(text))

    return nil
}

func main() {
    // Example arXiv URL
    arxivURL := "https://arxiv.org/abs/2305.14314" // Replace with your paper URL
    // pdfURL := "https://arxiv.org/pdf/2406.18665.pdf" // Replace with the PDF URL

    // Scrape the arXiv page
    headers, abstract, paragraphs, err := scrapeArxiv(arxivURL)
    if err != nil {
        fmt.Println(err)
        return
    }

    // Print the scraped data
    fmt.Println("Headers:")
    fmt.Println(headers)
    fmt.Println(len(headers))

    fmt.Println("\nAbstract:")
    // fmt.Println(abstract)
    fmt.Println(len(abstract))

    fmt.Println("\nParagraphs:")
    fmt.Println(paragraphs)

    // Download the PDF
    // err := downloadPDF(pdfURL, "paper.pdf")
    // if err != nil {
    //     fmt.Println(err)
    //     return
    // }

    // // Read the first page of the downloaded PDF
    // err = readFirstPagePDF("paper.pdf")
    // if err != nil {
    //     fmt.Println(err)
    //     return
    // }
}