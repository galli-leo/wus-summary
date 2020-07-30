package main

import (
	"fmt"
	"strings"
	"sync"
)

type Section struct {
	title       string
	Links       []string
	Subsections []*Section
	Distrs []*Distribution
}

func (s *Section) Section(num int) string {
	sub := num-1
	subs := strings.Repeat("sub", sub)
	return fmt.Sprintf("\\%ssection{%s}", subs, s.title)
}

func (s *Section) LoadDistrs(wg *sync.WaitGroup, sec int) error {
	for _, link := range s.Links {
		wg.Add(1)
		go func (link string)  {
			distr := GetDistr(link)
			if distr != nil {
				distr.Sec = sec
				s.Distrs = append(s.Distrs, distr)
			}
			wg.Done()
		}(link)
	}
	LoadDistrs(s.Subsections, wg, sec + 1)
	return nil
}

func LoadDistrs(sections []*Section, wg *sync.WaitGroup, s int) {
	for _, sec := range sections {
		sec.LoadDistrs(wg, s)
	}
}

func LoadAllDistrs(sections []*Section) {
	var wg sync.WaitGroup
	LoadDistrs(sections, &wg, 1)
	wg.Wait()
}