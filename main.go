// using goquery : https://github.com/PuerkitoBio/goquery

package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// 불러올 정보에 대한 struct
type jobInfo struct {
	id string
	title string
	company string
	location string
	summary string
}

var baseURL string = "https://kr.indeed.com/jobs?q=python&limit=50"

func main() {
	c := make(chan []jobInfo)
	var jobs []jobInfo // jobInfo(struct)의 slice
	totalPages := getPages() // get the number of pages

	for i:=0; i<totalPages; i++{
		go getPage(i, c) // 해당 page의 정보(jobInfo)를 가져온다. 50개의 array
	}

	for i:=0; i<totalPages; i++ {
		extractedJobs := <-c
		jobs = append(jobs, extractedJobs...)
	}


	wirteJobs(jobs) 
	fmt.Println("Done, extracted", len(jobs))
}

func getPage(page int, mainC chan<- []jobInfo) {
	// make slice - 한 페이지의 직업 정보를 담을 slice
	var jobs []jobInfo

	// make channel for using goroutines
	c := make(chan jobInfo)

	// Page마다의 url을 변경하기 위함
	pageURL := baseURL + "&start=" + strconv.Itoa(page*50) // Itoa : integer -> string
	fmt.Println("Requesting", pageURL)

	// page err checing
	res, err := http.Get(pageURL)
	checkErr(err)
	checkCode(res)
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	// 정보 추출
	// .sponTapItem class를 갖는 tag를 모두 찾음
	searchJobCards := doc.Find(".sponTapItem")
	// Each : 각 tag마다 실행
	searchJobCards.Each(func(i int, card *goquery.Selection) {
		go extractJob(card, c) // job의 디테일한 정보 50개를 동시에 실행 
	})

	for i:=0; i<searchJobCards.Length(); i++ {
		job := <-c // channel을 통해 오는 메세지를 받음
		jobs = append(jobs, job)
	}

	// struct의 slice
	mainC <- jobs
}

// job의 디테일한 정보 추출
func extractJob(card *goquery.Selection, c chan<- jobInfo)  {
	id, _ := card.Attr("data-jk")
	title := cleanString(card.Find(".jobTitle>span").Text())
	company := card.Find(".companyName").Text()
	location := card.Find(".companyLocation").Text()
	summary := card.Find(".job-snippet").Text()
	
	// 하나의 직업정보(struct) 반환
	c <- jobInfo{
		id: id, 
		title: title,
		company: company,
		location: location,
		summary: summary,
	}
}

func cleanString(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}
// strings.TrimSpace(str) :양쪽의 공백 제거
// strings.Fields : 텍스트 내부 공백제거 후 단어 가져오기
// strings.Join : 가져온 단어를 " "를 두고 string으로 만들기

// 총 페이지의 수 가져오기 
func getPages() int {
	pages := 0
	// URL Redirection
	res, err := http.Get(baseURL)
	
	// page를 받아오는 것 error check
	checkErr(err)
	checkCode(res)

	// res.Body : 기본적으로 byte IO(입출력)이므로 이 함수가 끝났을 때 닫아야 함 
	defer res.Body.Close()

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	// page 수를 나타내는 class 찾기
	doc.Find(".pagination").Each(func(i int, s *goquery.Selection) {
		// link 개수 받아오기
		pages = s.Find("a").Length()
	})
	
	return pages
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err) // Fatalln : Println + os.Exit(1)
	}
}

func checkCode(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalln("Request failed with Status:", res.StatusCode)
	}
}

// csv 파일로 만들기
func wirteJobs(jobs []jobInfo) {
	c := make(chan []string)
	// "jobs.csv" 파일명의 I/O file 생성
	file, err := os.Create("jobs.csv") 
	checkErr(err)

	// 파일을 새로 만듦
	writer := csv.NewWriter(file) 

	// 모든 데이터를 파일에 저장  
	defer writer.Flush() // defer : 마지막에 꼭 실행함

	// Index
	headers := []string{"ID", "Title", "Company", "Location", "Summary"} 
	wErr := writer.Write(headers)
	checkErr(wErr)

	// jobs의 형태는 []jobInfo이므로, job은 jobInfo이다.
	// job을 channel로 보내서 string의 형태로 jobInfo의 element를 묶을거다!
	// 묶은 것은 하나의 string slice type이 될 것이다.
	// => string slice type으로 채널을 선언해준다.  
	for _, job := range jobs {
		go bindingJobInfo(job, c)
	}
	
	for i:=0; i<len(jobs); i++ {
		jwErr := writer.Write(<-c)
		checkErr(jwErr)
	}
}

func bindingJobInfo(job jobInfo, c chan<- []string) {
	bindedJob := []string{"https://kr.indeed.com/%EC%B7%A8%EC%97%85?q=python&vjk=" + job.id + "&advn=1729840185424618" , " " + job.title, job.company, job.location, job.summary}
	c <- bindedJob
}

/*
	채널 적용법

		1. goroutine을 실행하고자하는 함수(1)에서 함수(2)로 채널의 정보를 보낸다.
		이때 채널은 받으려고하는 type으로 선언한다.

		2. 함수(2)에서 해당 작업을하고 채널을 통해 메세지를 보낸다.

		3. 해당 goroutine의 개수만큼 함수(1)에서 메세지를 받아야한다.  
*/