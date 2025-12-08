# learnings from building deisearch

stuff i learned building this search engine

## the weird ranking bug

was testing search for "database" and this random page "African Plant Database" showed up at 80% match. seemed odd because the page didn't really talk about databases - just had it in the title.

checked the db:

```sql
SELECT LENGTH(content), LENGTH(description), title
FROM pages WHERE url = 'https://africanplantdatabase.ch/en/nomen/129554';
-- content: 0 bytes wtf
```

turns out the spider only grabbed the title, no actual content. so the indexer saw "database" in the title and gave it max TF-IDF score (1.0). the math worked out to 80% even though the page was basically empty.

lesson: pages with only titles can mess up your rankings pretty bad

## why some pages had no content

made a test script to fetch the african plant db url directly:

- raw html: 2426 bytes
- text content: 1441 chars
- what spider got: 0 chars

ok so the content IS there, spider just couldn't see it. started checking other empty pages and found the same thing. tested a few:

- tiktok: 246K chars of content (all js rendered)
- some pages blocked by cloudflare
- most were modern sites using react/vue

realized the issue: modern websites don't put content in the html anymore. they do this:

```html
<div id="root"></div>
<script>
  // all the content loads here via javascript
</script>
```

standard http request doesn't run javascript, so you just get an empty div. found 4,621 pages with this problem.

## fixing it with two-phase crawling

didn't want to use a browser for everything (way too slow) but needed it for js sites. made it work in two steps:

1. try normal http fetch first
2. if content < 100 chars, retry with browser

code looks like:

```go
// phase 1: fast http
page, links := parser.Parse(httpResponse, url)
if page.HasSufficientContent() {
    return // done
}

// phase 2: slow browser
htmlContent := browserFetcher.FetchHTML(url)
browserPage, browserLinks := parser.ParseHTML(htmlContent, url)
if browserPage.HasSufficientContent() {
    page = browserPage  // use browser version
} else {
    return nil  // skip this page
}
```

results for african plant db:

- http fetch: 0 chars
- browser fetch: 3,769 chars (got the actual content about their plant database)

works pretty well. most pages are fast, only slow down for js-heavy sites.

## security stuff i didn't think about

was implementing the browser fetcher when i realized: wait, if i'm executing random javascript from the internet, what stops malicious pages from downloading malware or running cryptominers?

answer: nothing, unless you protect against it

added a bunch of flags to chromedp:

```go
chromedp.Flag("disable-downloads", true),     // no file downloads
chromedp.Flag("disable-plugins", true),       // no flash/java
chromedp.Flag("no-sandbox", false),           // keep sandbox on
chromedp.Flag("disable-background-networking", true), // no sneaky requests
```

also added checks before even fetching:

- content-type must be text/html
- size limit of 10MB
- already had extension filters (.exe, .zip, etc)

still probably not perfect but way better than just fetching everything
