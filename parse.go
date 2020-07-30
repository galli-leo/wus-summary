package main

import (
	"path"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

var plog = newLogger("parse")

func ParseLinks(sel *goquery.Selection) []string {
	return sel.Find("a").Map(func(i int, n *goquery.Selection) string {
		ret, _ := n.Attr("href")
		ret = path.Base(ret)
		return ret
	})
}

const probTitle = "Probability distribution"

const LinkRegex = `(?m)\[\[((?P<type>\w+):)?(?P<link>[^|\]]*)(\|(?P<options>[^|\]]*))*\]\]`

type LinkTag struct {
	Type    string
	Link    string
	Options string
}

func ParseLinkTag(txt string) []*LinkTag {
	ret := []*LinkTag{}

	re := regexp.MustCompile(LinkRegex)
	matches := re.FindAllStringSubmatch(txt, -1)
	for _, match := range matches {
		t := &LinkTag{}
		t.Link = match[3]
		t.Type = match[2]
		t.Options = match[5]
		ret = append(ret, t)
	}

	return ret
}

func ParseTemplateValue(txt string) string {
	re := regexp.MustCompile(`(?mi)<br\s*/?>`)
	txt = re.ReplaceAllString(txt, ", ")
	bold := regexp.MustCompile(`(?mi)'''(.*?)'''`)
	txt = bold.ReplaceAllString(txt, "\\textbf{$1}")
	italics := regexp.MustCompile(`(?mi)''(.*?)''`)
	txt = italics.ReplaceAllString(txt, "\\textit{$1}")
	sub := regexp.MustCompile(`(?mi)<\s*?sub\s*?>(.*?)</\s*?sub\s*?>`)
	txt = sub.ReplaceAllString(txt, "\\textsubscript{$1}")
	sup := regexp.MustCompile(`(?mi)<\s*?sup\s*?>(.*?)</\s*?sup\s*?>`)
	txt = sup.ReplaceAllString(txt, "\\textsuperscript{$1}")
	txt = html.UnescapeString(txt)
	return txt
}

func matchLen(match []int, idx int) int {
	return match[2*idx+1] - match[2*idx]
}

func CleanTemplateValue(txt string) string {
	re := regexp.MustCompile(LinkRegex)
	result := []byte{}
	currPos := 0
	for _, match := range re.FindAllStringSubmatchIndex(txt, -1) {
		tmpl := "${options}"
		if matchLen(match, 5) < 1 {
			tmpl = "${link}"
		}
		matchStart := match[0]
		result = append(result, []byte(txt[currPos:matchStart])...)
		result = re.ExpandString(result, tmpl, txt, match)
		currPos = match[1]
	}
	result = append(result, []byte(txt[currPos:len(txt)])...)
	return string(result)
}

func ParseImage(txt string) *Image {
	links := ParseLinkTag(txt)

	for _, link := range links {
		if link.Type == "Image" || link.Type == "File" {
			plog.Infof("Parsed Image: %v", link.Link)
			return &Image{
				Filename: link.Link,
				Caption:  link.Options,
			}
		}
	}

	return nil
}

func StripEmptyNewLines(txt string) string {
	split := strings.Split(txt, "\n")
	ret := []string{}
	for _, line := range split {
		if strings.TrimSpace(line) != "" {
			ret = append(ret, line)
		}
	}

	return strings.Join(ret, "\n")
}

func ParseTemplate(sel *goquery.Selection) map[string]interface{} {
	parts := sel.Find("part")
	ret := map[string]interface{}{}
	isbeta := false
	for i, _ := range parts.Nodes {
		psel := parts.Eq(i)
		names := psel.Find("name").First()
		name := strings.TrimSpace(names.Text())

		values := psel.Find("value").First().Contents()
		if name == "name" && strings.Contains(values.Text(), "Beta") {
			isbeta = true
		}
		if name == "pdf_image" {
			plog.Infof("Value for pdf_image: %v", values.Text())
		}
		val := ""
		if values.Length() < 1 {
			continue
		}

		for idx, vn := range values.Nodes {
			vsel := values.Eq(idx)
			if vn.Type == html.TextNode {
				txtVal := strings.TrimSpace(vsel.Text())
				txtVal = ParseTemplateValue(txtVal)

				img := ParseImage(txtVal)
				if img != nil {
					ret[name] = img
					goto end_outer
				}

				txtVal = CleanTemplateValue(txtVal)

				val += txtVal
			} else {
				name := vsel.Find("ext name")
				if strings.TrimSpace(name.Text()) != "math" {
					continue
				}
				inner := vsel.Find("ext inner")
				mathText := inner.Text()
				if strings.Contains(mathText, "\n") {
					val += `$$` + strings.TrimSpace(StripEmptyNewLines(mathText)) + `$$`
				} else if strings.Contains(mathText, "\\begin{align}") {
					val += mathText
				} else if len(mathText) > 0 {
					val += "$" + mathText + "$"
				}
			}
			val += " "
		}

		ret[name] = strings.TrimSpace(val)

	end_outer:
	}

	if isbeta {
		plog.Infof("why you do this: %v", ret)
	}

	return ret
}

func GetStr(props map[string]interface{}, key string) string {
	if ret, ok := props[key].(string); ok {
		return ret
	}

	return ""
}

func ParseDistribution(doc *goquery.Document) *Distribution {
	templates := doc.Find("template")
	for i, _ := range templates.Nodes {
		tsel := templates.Eq(i)
		title := tsel.Find("title")
		if strings.TrimSpace(title.Text()) == probTitle {
			distr := &Distribution{}
			props := ParseTemplate(tsel)
			if params, ok := props["parameters"].(string); ok {
				distr.Parameters = params
			}
			distr.Support = GetStr(props, "support")
			distr.Notation = GetStr(props, "notation")
			distr.Mean = GetStr(props, "mean")
			distr.Variance = GetStr(props, "variance")
			distr.PDF = GetStr(props, "pdf")
			distr.CDF = GetStr(props, "cdf")
			if img, ok := props["pdf_image"].(*Image); ok {
				distr.Image = img
				err := img.Download()
				if err != nil {
					plog.Errorf("Failed to download image %s: %v", img.Filename, err)
				}
			}
			return distr
		}
	}
	return nil
}

func ParseDistrLinks(doc *goquery.Document) []*Section {
	sections := []*Section{}

	tables := doc.Find("table")
	for idx, node := range tables.Nodes {
		sel := tables.Eq(idx)
		hdr := sel.Find("th div[id='Probability_distributions_(List)']").Nodes
		if len(hdr) > 0 {
			html, _ := goquery.OuterHtml(sel)
			plog.Infof("Found table: %v, %v", node, html)

			rows := sel.Find("tr")
			for idx, _ := range rows.Nodes {
				if idx == 0 {
					continue
				}
				section := &Section{}
				rsel := rows.Eq(idx)

				th := rsel.Children().First()
				section.title = th.Text()
				section.title = strings.Replace(section.title, "univariate", "univariate ", 1)

				plog.Infof("Found section: %s", section.title)

				dl := rsel.Find("dl").First()
				if dl.Length() > 0 {
					plog.Infof("Has subsections!")
					subsec := &Section{}
					children := dl.Children()
					for i, dd := range children.Nodes {
						dsel := children.Eq(i)
						if dd.Data == "dt" {
							if i != 0 {
								section.Subsections = append(section.Subsections, subsec)
								subsec = &Section{}
							}
							subsec.title = dsel.Find("span").Text()
						} else {
							link := ParseLinks(dsel)[0]
							plog.Infof("Found link: %s", link)
							subsec.Links = append(subsec.Links, link)
						}
					}
					section.Subsections = append(section.Subsections, subsec)
				} else {
					section.Links = ParseLinks(rsel.Find("td"))
				}

				sections = append(sections, section)
			}
		}
	}

	return sections
}
