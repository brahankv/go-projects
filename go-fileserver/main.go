package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	port    = flag.String("port", "30006", "Port to run the server on")
	folders = flag.String("folders", "", "Comma-separated list of folders to serve")
)

func main() {
	flag.Parse()
	if *folders == "" {
		log.Fatal("No folders provided. Use -folders to specify folders.")
	}
	folderList := strings.Split(*folders, ",")
	for _, folder := range folderList {
		if _, err := os.Stat(folder); os.IsNotExist(err) {
			log.Fatalf("Folder does not exist: %s", folder)
		}
	}

	// Serve static files (UI)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	// API: Tree view
	http.HandleFunc("/api/tree", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" || path == "/" {
			// List root folders
			var out []map[string]string
			for _, f := range folderList {
				out = append(out, map[string]string{"name": filepath.Base(f), "type": "folder", "path": f})
			}
			json.NewEncoder(w).Encode(out)
			return
		}
		// List files/folders in given path
		fis, err := ioutil.ReadDir(path)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var out []map[string]string
		for _, fi := range fis {
			t := "file"
			if fi.IsDir() {
				t = "folder"
			}
			out = append(out, map[string]string{
				"name": fi.Name(),
				"type": t,
				"path": filepath.Join(path, fi.Name()),
			})
		}
		json.NewEncoder(w).Encode(out)
	})

	// API: File view
	http.HandleFunc("/api/file", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "Missing path", 400)
			return
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		// Check for binary content
		isBinary := false
		for i := 0; i < len(data) && i < 800; i++ {
			if data[i] == 0 {
				isBinary = true
				break
			}
			if data[i] < 0x09 || (data[i] > 0x0D && data[i] < 0x20) {
				isBinary = true
				break
			}
		}
		ext := filepath.Ext(path)
		lang := extToLang(ext)
		if isBinary {
			json.NewEncoder(w).Encode(map[string]string{
				"content": "[Binary file will not be displayed]",
				"language": "",
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"content": string(data),
			"language": lang,
		})
	})


	// API: Upload
	http.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		err := r.ParseMultipartForm(32 << 20) // 32MB
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		folder := r.FormValue("folder")
		if folder == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Missing folder"})
			return
		}
		files := r.MultipartForm.File["files"]
		for _, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
			defer file.Close()
			outPath := filepath.Join(folder, fileHeader.Filename)
			out, err := os.Create(outPath)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
			defer out.Close()
			_, err = ioutil.ReadAll(file)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
			file.Seek(0, 0)
			_, err = io.Copy(out, file)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
				return
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	})

	// API: Download
	http.HandleFunc("/api/download", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "Missing path", 400)
			return
		}
		fname := filepath.Base(path)
		w.Header().Set("Content-Disposition", "attachment; filename="+fname)
		mimeType := mime.TypeByExtension(filepath.Ext(fname))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", mimeType)
		http.ServeFile(w, r, path)
	})

	log.Printf("Serving on :%s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

// extToLang maps file extensions to highlight.js language classes
func extToLang(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".json":
		return "json"
	case ".md":
		return "markdown"
	default:
		return ""
	}
}
