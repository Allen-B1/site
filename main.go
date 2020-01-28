package main

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func listFiles(dir string) ([]string, error) {
	out := make([]string, 0)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("couldn't list files: %w", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "~") || strings.HasPrefix(file.Name(), ".") {
			continue
		}
		if file.IsDir() {
			subFiles, err := listFiles(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}
			for _, subFile := range subFiles {
				out = append(out, filepath.Join(file.Name(), subFile))
			}
		} else {
			out = append(out, file.Name())
		}
	}
	return out, nil
}

// Type FileHandler serves a file.
type FileHandler string

func (f FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadFile(string(f))
	if err != nil {
		w.WriteHeader(500)
		io.WriteString(w, "internal server error")
		log.Printf("error loading '%s': %v\n", string(f), err)
		return
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(string(f))))
	w.WriteHeader(200)
	w.Write(body)
}

var Templates = template.New("").Funcs(template.FuncMap{
	"list": func(arr ...interface{}) []interface{} {
		return arr
	},
})

type TemplateHandler string

func (t TemplateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := Templates.ExecuteTemplate(w, filepath.Base(string(t)), nil)
	if err != nil {
		log.Printf("error templating '%s': %v\n", string(t), err)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	server := &http.Server{
		ReadHeaderTimeout: time.Second * 8,
		WriteTimeout:      time.Second * 8,
		Handler:           mux,
		Addr:              ":" + port,
	}

	files, err := listFiles(".")
	if err != nil {
		log.Println("fatal: ", err)
		os.Exit(1)
	}
	fmt.Println(files)
	for _, file := range files {
		if strings.HasSuffix(file, ".html") {
			_, err = Templates.ParseFiles(file)
			if err != nil {
				log.Println("fatal: ", err)
				os.Exit(1)
			}
			index := strings.Index(file, ".")
			name := file
			if index >= 0 {
				name = file[:index]
			}
			mux.Handle("/"+name, TemplateHandler(file))
		} else {
			mux.Handle("/"+file, FileHandler(file))
		}
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			TemplateHandler("index.html").ServeHTTP(w, r)
		} else {
			w.Header().Set("Location", "/")
			w.WriteHeader(303)
		}
	})
	server.ListenAndServe()
}
