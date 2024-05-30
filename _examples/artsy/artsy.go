package main

import (
	"fmt"
	"github.com/gocolly/colly/v2"
	"log"
	"os"
)

var characters = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "visitArtistsList", "w", "x", "y", "z"}

type ArtistsScrapper struct {
	*colly.Collector
	nextPage, nextChar chan struct{}
	file               *os.File
}

type ArtScrapper struct {
	*colly.Collector
	nextPage chan struct{}
	file     *os.File
}

func main() {
	fName := "./artsy.csv"
	file, _ := os.Create(fName)
	defer file.Close()
	w := initializeCSV(file)
	w.Write([]string{"Artist Name", "Art Name", "Price", "Link"})
	w.Flush()

	c := colly.NewCollector(
		colly.Async(true),
		colly.AllowedDomains("www.artsy.net", "artsy.net"),
		colly.CacheDir("./artsy_cache"),
	)

	artistSCrapper := &ArtistsScrapper{
		c,
		make(chan struct{}),
		make(chan struct{}),
		file,
	}

	artistSCrapper.parseArtistsList()
	for _, char := range characters {
		pageNum := 1
		artistsBaseAddress := fmt.Sprintf("https://www.artsy.net/artists/artists-starting-with-%s?page=%d", char, pageNum)
		err := artistSCrapper.Visit(artistsBaseAddress)
		if err != nil {
			log.Print(err)
		}
		for {
			select {
			case <-artistSCrapper.nextChar:
				break
			case <-artistSCrapper.nextPage:
				pageNum++
				artistsBaseAddress = fmt.Sprintf("https://www.artsy.net/artists/artists-starting-with-%s?page=%d", char, pageNum)
				err = artistSCrapper.Visit(artistsBaseAddress)
				if err != nil {
					log.Print(err)
				}
			default:
			}
		}
	}

}

func (s *ArtistsScrapper) parseArtistsList() {
	s.OnHTML(".jJWpXK", func(artistsBox *colly.HTMLElement) {
		if len(artistsBox.Text) == 0 {
			//pages on specific characters ends, let's go for next character
			s.nextChar <- struct{}{}
		}
		artistsBox.ForEach(".Box-sc-15se88d-0", func(_ int, artistBox *colly.HTMLElement) {
			artistBox.ForEach("a", func(_ int, artistUrl *colly.HTMLElement) {
				pageNum := 1
				artScrapper := &ArtScrapper{
					s.Clone(),
					make(chan struct{}),
					s.file,
				}

				artScrapper.parseArtistArtWorks()
				url := fmt.Sprintf("https://www.artsy.net%s?page=%d", artistUrl.Attr("href"), pageNum)
				err := artScrapper.Visit(url)
				if err != nil {
					log.Print(err)
				}

				for {
					select {
					case <-artScrapper.nextPage:
						pageNum++
						url = fmt.Sprintf("https://www.artsy.net%s?page=%d", artistUrl.Attr("href"), pageNum)
						err = artScrapper.Visit(url)
						if err != nil {
							log.Print(err)
						}
					default:
					}
				}
			})
		})
	})
}

func (s *ArtScrapper) parseArtistArtWorks() {

	art := &Art{}
	s.OnHTML(".idOkRo h1", func(artist *colly.HTMLElement) {
		art.ArtistName = artist.Text
		s.writeToCSV(art)
	})

	s.OnHTML(".drwzqv .fresnel-at-xs", func(parent *colly.HTMLElement) {
		parent.ForEach(".eilryE", func(_ int, artWorkMetaData *colly.HTMLElement) {
			artWorkMetaData.ForEach(".ilQWRL", func(_ int, artName *colly.HTMLElement) {
				art.Name = artName.Text
				s.writeToCSV(art)
			})

			artWorkMetaData.ForEach(".eXbAnU", func(_ int, price *colly.HTMLElement) {
				art.Link = artWorkMetaData.Attr("href")
				art.Price = price.Text
				s.writeToCSV(art)
			})
		})
	})
}

func (s *ArtScrapper) writeToCSV(a *Art) {
	if len(a.ArtistName) > 0 &&
		len(a.Name) > 0 &&
		len(a.Price) > 0 &&
		len(a.Link) > 0 {
		w := initializeCSV(s.file)
		err := w.Write([]string{a.ArtistName, a.Name, a.Price, a.Link})
		if err != nil {
			fmt.Println(err)
		}
		w.Flush()
		s.nextPage <- struct{}{}
	}
}
