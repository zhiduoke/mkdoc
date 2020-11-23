package main

import (
	"errors"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-yaml/yaml"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	conf := getConfig()
	err := rewriteMkdocProject(conf)
	if err != nil {
		log.Fatal(err)
	}
	processMakeDoc(conf)
	notify()
	server(conf)
}

func server(conf *config) {
	registerHandler(conf)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func registerHandler(conf *config) {
	// handle static
	log.Println("server docs:")
	log.Printf("    %s\t=>\t127.0.0.1:8080", "index")
	for _, v := range conf.mkdocConfigs {
		id := v["id"].(string)
		docdir := filepath.Join("project", id, "docs", "docsify")
		log.Printf("    %s\t=>\t127.0.0.1:8080/%s", id, id)
		prefix := "/" + id + "/"
		h := http.StripPrefix(prefix, http.FileServer(http.Dir(docdir)))
		http.Handle(prefix, newBasicAuthHandler(conf.webUserName, conf.webPassword, h))
	}
	log.Println("notify url: 127.0.0.1:8080/notify")

	// handle notify
	http.HandleFunc("/notify", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			return
		}
		token := request.FormValue("token")
		if token != conf.notifyToken {
			writer.WriteHeader(http.StatusForbidden)
			return
		}
		log.Println("update document from:", request.RemoteAddr)
		delayNotify()
	})

	// handle index
	tpl, err := template.New("index.html").Parse(docIndexTemplate)
	if err != nil {
		log.Fatal(err)
	}
	type project struct {
		Link string
		Name string
	}
	var data = struct {
		Projects []*project
	}{}
	for _, v := range conf.mkdocConfigs {
		id := v["id"].(string)
		name := id
		if pname, ok := v["name"].(string); ok {
			name = pname
		}
		link := "/" + id
		data.Projects = append(data.Projects, &project{Name: name, Link: link})

	}
	handleWithBasicAuth("/", conf.webUserName, conf.webPassword, func(writer http.ResponseWriter, request *http.Request) {
		err = tpl.Execute(writer, data)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
		}
	})
}

var makeDocChan = make(chan struct{})

func notify() {
	go func() { makeDocChan <- struct{}{} }()
}

var delayNotify = debounce(notify, time.Second*3)

func processMakeDoc(conf *config) {
	go func() {
		for {
			<-makeDocChan
			err := checkoutRepo(conf)
			if err != nil {
				log.Println("checkout repo:", err)
				continue
			}
			//wg := sync.WaitGroup{}
			for _, v := range conf.mkdocConfigs {
				//wg.Add(1)
				dir := filepath.Join("project", v["id"].(string))
				//go func() {
				//defer wg.Done()
				o, err := makeDoc(dir)
				if err != nil {
					log.Println("mkdoc:", err)
				}
				if conf.debug {
					log.Println(string(o))
				}
				//}()
			}
			//wg.Wait()
			os.RemoveAll("./src")
		}
	}()
}

// call mkdoc command
func makeDoc(dir string) ([]byte, error) {
	cmd := exec.Command("mkdoc", "make")
	cmd.Dir = dir
	return cmd.Output()
}

func repoDir(url string) string {
	i := strings.LastIndex(url, "/")
	if i == -1 {
		log.Fatalf("config: invalid repo url: %s", url)
	}
	return url[i : len(url)-len(".git")]
}

// clone and checkout source repo
func checkoutRepo(conf *config) error {
	repoURL, branch := conf.repoURL, conf.branchName
	userName, password := conf.gitUserName, conf.gitPassword
	cloneDir := repoDir(repoURL)
	var auth string
	if len(userName) > 0 {
		auth = userName
	}
	if len(password) > 0 {
		auth = auth + ":" + password
	}
	if len(auth) > 0 {
		auth += "@"
		if strings.HasPrefix(repoURL, "https://") {
			repoURL = "https://" + auth + repoURL[len("https://"):]
		} else if strings.HasPrefix(repoURL, "http://") {
			repoURL = "http://" + auth + repoURL[len("http://"):]
		} else {
			return errors.New("invalid repo url: " + repoURL)
		}
	}
	err := os.RemoveAll("./src")
	if err != nil {
		return err
	}
	opt := &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
		Depth:         1,
		Progress:      nil,
	}
	if conf.debug {
		opt.Progress = os.Stdout
		log.Println("clone:", repoURL)
	}
	err = os.Mkdir("./src", 0755)
	if err != nil {
		return err
	}

	_, err = git.PlainClone(filepath.Join("./src", cloneDir), false, opt)
	return err
}

func rewriteMkdocProject(conf *config) error {
	for _, v := range conf.mkdocConfigs {
		dir := filepath.Join("project", v["id"].(string))
		os.RemoveAll(dir)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
		o, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dir, "conf.yaml"), o, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
