package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"golang.org/x/net/html"
)

type CrawlJob struct {
	URL   string
	Depth int
}

// extractLinks parses the HTML response body and return a slice of all href URLs found.
func extractLinks(resp *http.Response, baseURL string) []string {
	var links []string

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		fmt.Printf("Error parsing base URL: %v\n", err)
		return links
	}

	//html.Parse reads the response body and builds the node tree
	doc, err := html.Parse(resp.Body)
	if err != nil {
		fmt.Printf("Error parsing HTML: %v\n", err)
		return links
	}

	// This is an anonymous, recursive function .It checs the current node, then calls itself for every child node
	var visitNode func(n *html.Node)
	visitNode = func(n *html.Node) {
		//If this is an HTML element and the tag name is "a"
		if n.Type == html.ElementNode && n.Data == "a" {
			//Loop through the attributes (like class, id, href)
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					parsedlink, err := url.Parse(attr.Val)
					if err == nil {
						//ResolveReference handles relative paths (eg., "/about" -> "http://example.com/about")
						absoluteLink := parsedBaseURL.ResolveReference(parsedlink)

						//Only keep http and https links (ignore mailto:, javascript:, etc.)
						if absoluteLink.Scheme == "http" || absoluteLink.Scheme == "https" {
							//Remove fragmenrts (#) so we don't treat "page.com#top" and "page.com#bottom" as different pages
							absoluteLink.Fragment = ""
							links = append(links, absoluteLink.String())
						}
					}
					break // Found the href, move on
				}
			}
		}
		// Recursivelt visit all child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visitNode(c)
		}
	}
	// Start walking the tree from the top document node
	visitNode(doc)
	return links
}

// crawl manages the logic of visiting a page, extracting links, and dive deeper
/*func crawl(targetURL string, depth int, maxDepth int, visited map[string]bool) {
	if depth > maxDepth {
		return
	}
	// IF we have already visited this URL, skip it to prevent infinite loops.
	if visited[targetURL] {
		return
	}
	//Mark as visited BEFORE fetching to prevent other recurssive calls from hitting it
	visited[targetURL] = true

	//Print with indentation based on depth
	fmt.Printf("%s Crawling: %s\n", strings.Repeat("-", depth), targetURL)

	resp, err := http.Get(targetURL)
	if err != nil {
		fmt.Printf("Error fetching %s: %v\n", targetURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return // returning quitly on a bad status rather than exiting the wholw process
	}
	links := extractLinks(resp, targetURL)

	// Recursively crawl all found links
	for _, link := range links {
		crawl(link, depth+1, maxDepth, visited)
	}
}*/

func workers(id int, jobs <-chan CrawlJob, wg *sync.WaitGroup, mu *sync.Mutex, visited map[string]bool, maxDepth int, results chan<- CrawlJob) {
	//Loops forever, waiting for jobs to arrive on the channel
	for job := range jobs {
		//1. LOCK HTE DOOR: We need to safely check/update the visited map
		mu.Lock()
		if visited[job.URL] || job.Depth > maxDepth {
			mu.Unlock() // UNLOCK immediately if weare skipping
			wg.Done()   //Cross this job off the manager's clipboard
			continue    // Skip to the next job in the channel
		}
		//Mark as visited while we still have the lock
		visited[job.URL] = true
		mu.Unlock() //UNLOCK so other workers can use the map

		//2. Do the heavy lifting (crawling)
		fmt.Printf("Worker %d Crawling (Depth %d: %s\n", id, job.Depth, job.URL)

		resp, err := http.Get(job.URL)
		if err != nil {
			fmt.Printf("Worker %d Error fetching %s : %v\n", id, job.URL, err)
			wg.Done()
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			wg.Done()
			continue
		}

		//3. Extract the new links
		links := extractLinks(resp, job.URL)
		resp.Body.Close()

		//4. Push the new links BACK to the manager via a separate channel
		for _, link := range links {
			wg.Add(1)
			results <- CrawlJob{URL: link, Depth: job.Depth + 1}
		}

		//5. Cross this completed job off the managers clipboard
		wg.Done()
	}
}

func main() {
	startURL := "https://www1.iitp.ac.in/"
	maxDepth := 1 // Be careful setting this too high!

	// Initialize our visited map.
	// In Go, maps must be initialized with 'make' before you can write to them.
	visited := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	//Create our two channels:
	// 'jobs' is the queue of the URLs waiting to be crawled
	// 'results' is where workers dump the NEW URLs they find

	jobs := make(chan CrawlJob, 1000)
	results := make(chan CrawlJob, 1000)

	fmt.Printf("Starting sequential crawl at %s...\n", startURL)

	// Start 5 concurrnet workers
	for i := 1; i <= 10; i++ {
		go workers(i, jobs, &wg, &mu, visited, maxDepth, results)
	}

	//Prime the pump: Send the seed URL into the jobs queue
	wg.Add(1)
	jobs <- CrawlJob{URL: startURL, Depth: 0}

	//Coordinator loop: Listen for new URL's found by the workers
	//We run this in a background goroutine so the main function can move on and Wait()
	go func() {
		for result := range results {
			go func(job CrawlJob) {
				jobs <- job
			}(result)
		}
	}()

	//Wait for the clipboard to reach exactly zero (all workers are idle)
	wg.Wait()
	close(results)
	fmt.Printf("\nCrawl complete! Visited %d unique pages.\n", len(visited))
}
