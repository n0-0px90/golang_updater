package main

//TODO: Write extraction function for tarballs
//TODO: Write recursive delete function for uneeded directories
//TODO: Write switch case for if user is root
//TODO: Write switch case for if user installs local
//TODO: Write

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	goURL string = "https://go.dev/dl/"
)

func get_user_directory() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("I dont know how this happened: %q\n", err)
	}
	user_download_folder := dirname + "/Downloads/"
	return user_download_folder
}

func golang_website_langver(htmlWebPage *goquery.Document) string {
	re := regexp.MustCompile(`go(\d{1,3}\.){2}\d{1,3}`)

	var matches []string

	htmlWebPage.Find("div, a").Each(func(index int, item *goquery.Selection) {

		if item.HasClass("download downloadBox") {

			match, err := regexp.MatchString(`go(\d{1,3}\.?){1,3}`, item.Text())

			if match {
				matches = append(matches, re.FindString(item.Text()))
			}

			if err != nil {
				log.Fatal(err)
			}
		}
	})

	return matches[0]
}

func golang_download(download_ver string) {
	linux_download := download_ver + ".linux-amd64.tar.gz"
	download_location := get_user_directory() + linux_download
	tarball, err := os.Create(download_location)

	if err != nil {
		log.Fatalf("Failed to create file: %s", download_location)
	}

	defer tarball.Close()

	fmt.Printf("Downloading %s\nYour download will be at %s\n", linux_download, download_location)

	download, err := http.Get(goURL + linux_download)
	if err != nil {
		log.Fatal(err)
	}

	defer download.Body.Close()

	if download.StatusCode != 200 {
		log.Fatalf("Status code wasn't 200: %d %s\n", download.StatusCode, download.Status)
	}
	_, err = io.Copy(tarball, download.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Downloaded %s to %s!\n", linux_download, download_location)
	fmt.Println("Would you like to extract and install according to GoLang documentation?")
	user_choice := bufio.NewReader(os.Stdin)
	choice, err := user_choice.ReadString('\n')
	if err != nil {
		log.Fatalf("How did you do this? %q\n", err)
	}
	fmt.Println(choice)
	switch strings.ToLower(choice) {
	case "yes\n":
		fmt.Println("Just kidding, this isn't implemented yet.")
	case "no\n":
		fmt.Println("Goodbye.")
	default:
		fmt.Println("Gonna take that as a no. Goodbye!")

	}
}

func update_golang(goVer string) {
	fmt.Println("Checking", goURL, "for current version")
	resp, err := http.Get(goURL)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("Status code wasn't 200: %d %s\n", resp.StatusCode, resp.Status)
	}

	htmlWebPage, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var goWebsiteVersion string = golang_website_langver(htmlWebPage)
	if goWebsiteVersion == goVer {
		fmt.Println("Your current version is up to date.")
		return
	}

	fmt.Printf("GoLang is currently version %s, you are behind.\n", goWebsiteVersion)
	golang_download(goWebsiteVersion)
}

func main() {
	goCurrentVersion, goNotFound := exec.Command("go", "version").Output()

	if goNotFound != nil {
		fmt.Println("Read and install GoLang manually first.\nIf you have installed it, fix your path.")
		log.Fatal(goNotFound)
	}

	goVer := strings.Split(string(goCurrentVersion), " ")
	fmt.Println("Current GoLang version:", goVer[2])
	update_golang(goVer[2])
}
