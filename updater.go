package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	goURL string = "https://go.dev/dl/"
)

//this is completely wrong looking at this now
//TODO: Fix to stat poss. installed Go dir,
//not TLDir. that makes no sense judas 
//priest
func delete_current_install(extraction_destination string) {
	path_exists, path_error := os.Stat(extraction_destination)
	if path_error != nil {
		return
	}
	switch path_exists {
	default:
		os.RemoveAll(extraction_destination + "go")
	}
}

func delete_tarball(user_download_directory string) {
	directory_listing, dir_error := os.ReadDir(user_download_directory)
	re := regexp.MustCompile(`go(\d{1,2}\.){1,3}.*`)
	if dir_error != nil {
		log.Fatalf("Failed to read directory: %q\n", dir_error)
	}
	for _, file := range directory_listing {
		if re.FindString(file.Name()) != "" {
			os.Remove(user_download_directory + file.Name())
		}
	}
}

// Untar file source string -> target. Returns err if fail
func extract_tar(source, target string) {
	reader, open_err := os.Open(source)
	if open_err != nil {
		log.Fatal(open_err)
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)
	defer func() {
		for {

			header, tar_err := tarReader.Next()
			if tar_err == io.EOF {
				break
			} else if tar_err != nil {
				log.Fatal(tar_err)
			}

			path := filepath.Join(target, header.Name)
			info := header.FileInfo()
			if info.IsDir() {
				if tar_err = os.MkdirAll(path, info.Mode()); tar_err != nil {
					log.Fatal(tar_err)
				}
				continue
			}

			WriteFile, wf_err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
			if wf_err != nil {
				log.Fatal(wf_err)
			}

			_, copy_err := io.Copy(WriteFile, tarReader)
			if copy_err != nil {
				log.Fatal(copy_err)
			}
			if close_err := WriteFile.Close(); close_err != nil {
				log.Fatal(close_err)
			}
		}
	}()
}

// Decompress source to target, returns err if fail
func decompress_gzip(source, target string) {
	defer func() {
		reader, open_err := os.Open(source)
		if open_err != nil {
			log.Fatal(open_err)
		}
		defer reader.Close()

		archive, ext_err := gzip.NewReader(reader)
		if ext_err != nil {
			log.Fatal(ext_err)
		}
		defer archive.Close()

		target = filepath.Join(target, archive.Name)
		writer, write_err := os.Create(target)
		if write_err != nil {
			log.Fatal(write_err)
		}
		defer writer.Close()

		_, copy_error := io.Copy(writer, archive)
		if copy_error != nil {
			log.Fatal(copy_error)
		}
	}()

}

func get_user_directory() (string, string) {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("I dont know how this happened: %q\n", err)
	}
	user_download_folder := dirname + "/Downloads/"
	if _, err := os.Stat(user_download_folder); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(user_download_folder, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
	return user_download_folder, dirname
}

// Regquery downloads on go.dev/dl
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

func extract_and_cleanup(download_location, user_download_directory, download_ver, extraction_destination string) {
	delete_current_install(extraction_destination)

	decompress_gzip(download_location, user_download_directory+download_ver+".linux-amd64.tar")

	extract_tar(user_download_directory+download_ver+".linux-amd64.tar", extraction_destination)

	delete_tarball(user_download_directory)
	fmt.Printf("Extracted to %s\n", extraction_destination)
	fmt.Printf("Double check your path statement, verify its pointing to %sgo/bin\n", extraction_destination)
	os.Exit(1)
}

// Download from new web request
func golang_download(download_ver string) {
	var extraction_destination string
	linux_download := download_ver + ".linux-amd64.tar.gz"
	user_download_directory, user_home := get_user_directory()
	download_location := user_download_directory + linux_download
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
	fmt.Println("Would you like to extract and install now?")
	fmt.Printf("Options: Yes or No\n")
	user_choice := bufio.NewReader(os.Stdin)
	choice, err := user_choice.ReadString('\n')
	if err != nil {
		log.Fatalf("How did you do this? %q\n", err)
	}
	current_user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	switch strings.ToLower(choice) {
	case "yes\n":
		if current_user.Uid != "0" {
			extraction_destination = user_home + "/.local/"
			fmt.Printf("Extracting %s to %s\n", linux_download, extraction_destination)
			extract_and_cleanup(download_location, user_download_directory, download_ver, extraction_destination)
		} else {
			extraction_destination = "/usr/local/"
			fmt.Printf("Extracting %s to /usr/local/\n", linux_download)
			extract_and_cleanup(download_location, user_download_directory, download_ver, extraction_destination)
		}
	case "no\n":
		fmt.Printf("GoLang download is sitting at: %s\n", download_location)
	default:
		fmt.Println("Gonna take that as a no. Goodbye!")

	}
}

// Web request to go.dev
func update_golang(goVer string) {
	fmt.Println("Checking", goURL, "for new version")
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

	goWebsiteVersion := golang_website_langver(htmlWebPage)
	if goWebsiteVersion == goVer {
		fmt.Println("Your current version is up to date.")
		return
	}

	fmt.Printf("GoLang is currently version %s, you are behind.\n", goWebsiteVersion)
	golang_download(goWebsiteVersion)
}

// Entry Point
func main() {
	goCurrentVersion, goNotFound := exec.Command("go", "version").Output()

	if goNotFound != nil {
		fmt.Println("Verify you have added your /go/bin instance to your path. Would you like to install now?")
		user_choice := bufio.NewReader(os.Stdin)
		fmt.Printf("Yes or No?: ")
		choice, err := user_choice.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		switch strings.ToLower(choice) {
		case "yes\n":
			fmt.Printf("\nInstalling GoLang\n")
			update_golang("nil")
			os.Exit(1)
		default:
			fmt.Println("Goodbye!")
			os.Exit(127)
		}

	}

	goVer := strings.Split(string(goCurrentVersion), " ")
	fmt.Println("Current GoLang version:", goVer[2])
	update_golang(goVer[2])
}
