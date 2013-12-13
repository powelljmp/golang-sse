package main

import (
	"fmt"
	"net/http"
	"html/template"
	"time"
	"strconv"
	"sync"
	"io"
	"runtime"
)

//Path for static files
const PATH 	= "/static/"

//poll repeat time
const SLEEPTIME = 30

type ServerCheck struct {
	url string
	tag string
}

//domains to check
var slice_urls = []ServerCheck{
	ServerCheck{"http://www.google.co.uk/","Google"},
	ServerCheck{"http://www.yahoo.co.uk/","Yahoo"},	
	ServerCheck{"http://www.bing.co.uk/","Bing"}
	ServerCheck{"http://www.down-domain-no-loading.co.uk","Down"}, 
	}
	
//map of channels
var clientPool = make(map[chan string]bool)

func getStatus(sc ServerCheck, wg *sync.WaitGroup){
	g := "500 -|- URL: "+sc.url+" -|- TAG:"+sc.tag
	//request URL
	req, err := http.Get(sc.url)
	//catch err return		
	if err != nil {
		if err == io.EOF {
		   fmt.Println(err) 
		   fmt.Println("EOF caught")               
		}
		fmt.Println("\n GET err:", err)			
	} else {
		//grab header code from return
		g = req.Status[0:3]+"-|-"+sc.url+"-|-"+sc.tag
		//close request
		req.Close = true	
		defer req.Body.Close()
	}
	//Finish waiting group worker
	wg.Done()
	//push the response through map of channels
	for key := range clientPool {
		key <-g			
	}	
}

func loudSpeeker(w http.ResponseWriter, r *http.Request){
	fmt.Printf("\n\nNew user\n")
	var user = make(chan string)
	clientPool[user]=true

	f, ok := w.(http.Flusher)	
	if !ok {
        http.Error(w, "Streaming unsupported!",
                http.StatusInternalServerError)
        return
	}
	c, ok := w.(http.CloseNotifier)
	if !ok {
        http.Error(w, "close notification unsupported",
                http.StatusInternalServerError)
        return
	}
	closer := c.CloseNotify()
	
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {		
		select {
			case u := <-user:
				if(u[0:1]=="q"){
					t := time.Now().Format("2006-01-02 15:04:05")
					fmt.Fprintf(w, "data:%s\r\n\n", "clear")
					fmt.Fprintf(w, "data:000 COUNT: "+u[1:]+" : %v\r\n\n", t)	
				}else{
					//fmt.Printf("user channel\n")
					fmt.Fprintf(w, "data: %v\r\n\n", u)	
				}							
				f.Flush()
			case <-closer:
              	fmt.Printf("Closing connection to user\n")
              	delete(clientPool, user)
               	return          
		}		
	}
}

func theBrain(){
				
	l := 1
	//create wait group
	wg := new(sync.WaitGroup)	
	for {
		runtime.Gosched()	
		//send loop counter to each channel
		for key := range clientPool {
			key <- "q"+strconv.Itoa(l)			
		}			
		fmt.Printf("\nRunning check #"+strconv.Itoa(l)+"\n")
		for i := 0; i <len(slice_urls); i++ {
			//add a wait group worker for each url
			wg.Add(1)
			//sleep before next request. helps with frontend load
			time.Sleep(0.5 * 1e9)
			//send request to new goroutine
			go getStatus(slice_urls[i], wg)
		}
		//wait for wait group to be completed. i.e all urls returned
		wg.Wait()
		fmt.Printf("\n...sleeping..."+strconv.Itoa(SLEEPTIME)+" secs\n")							
		l++
		time.Sleep(SLEEPTIME * 1e9)	
	}
}

func homepage(w http.ResponseWriter, r *http.Request){ 
	t, err := template.ParseFiles(PATH+"/homepage7.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, "nil")	
}

func main(){
	fmt.Printf("\nSTART PROGRAM \n\n")
	
	go theBrain()
		
	//handle directory/files	
	http.Handle("/css/", http.FileServer(http.Dir(PATH)))
	http.HandleFunc("/updates",loudSpeeker)
	http.Handle("/",http.HandlerFunc(homepage))
			
	//listen on port		
	http.ListenAndServe("localhost:8080",nil)
}