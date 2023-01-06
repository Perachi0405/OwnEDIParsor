package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/jf-tech/go-corelib/jsons"
	"github.com/jf-tech/omniparser/transformctx"
	"github.com/spf13/cobra"

	// "github.com/Perachi0405/ownEDIParsor/transformctx"
	"github.com/jf-tech/omniparser"
)

//&cobra.Command is an interface
//using the command ./op server will trigger the
var (
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Launches op into HTTP server mode with its REST APIs.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			doServer()
		},
	}
	port int
)

func init() {
	fmt.Println("init inside serverCmd.go")
	serverCmd.Flags().IntVarP(&port, "port", "p", 8080, "the listening HTTP port")
}

const (
	contentTypeHeader = "Content-Type"
	contentTypeJSON   = "application/json"
)

//creating a new HTTP request
func doServer() {
	fmt.Println("Inside doServer() serverCmd.go")
	transformRouter := chi.NewRouter()
	fmt.Println("ServerCmd.go file transformRouter", transformRouter)
	transformRouter.Use(middleware.RealIP)
	transformRouter.Use(middleware.AllowContentType(contentTypeJSON))
	transformRouter.Post("/", httpPostTransform)

	samplesRouter := chi.NewRouter()
	fmt.Println("ServerCmd.go file samplesRouter", samplesRouter)
	samplesRouter.Get("/", httpGetSamples)

	versionRouter := chi.NewRouter()
	fmt.Println("ServerCmd.go file versionRouter", versionRouter)
	versionRouter.Get("/", httpGetVersion)

	rootRouter := chi.NewRouter()
	rootRouter.Get("/", func(w http.ResponseWriter, req *http.Request) {
		// http.Dir
		http.FileServer(http.Dir(filepath.Join(serverCmdDir(), "web"))).ServeHTTP(w, req)
	})
	fmt.Println("ServerCmd.go file rootRouter", rootRouter)
	rootRouter.Mount("/transform", transformRouter)
	rootRouter.Mount("/samples", samplesRouter)
	rootRouter.Mount("/version", versionRouter)

	envPort, found := os.LookupEnv("PORT")
	if found {
		var err error
		port, err = strconv.Atoi(envPort)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Listening on port %d ...", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), rootRouter))
}

func serverCmdDir() string {
	_, filename, _, _ := runtime.Caller(1)
	fmt.Println("servercmdDir filename", filename)
	absDir, _ := filepath.Abs(filepath.Dir(filename))
	fmt.Println("servercmdDir absDir", absDir)
	return absDir
}

func writeError(w http.ResponseWriter, msg string, code int) {
	http.Error(w, msg, code)
	log.Print(code)
}

func writeBadRequest(w http.ResponseWriter, msg string) {
	writeError(w, msg, http.StatusBadRequest)
}

func writeInternalServerError(w http.ResponseWriter, msg string) {
	writeError(w, msg, http.StatusInternalServerError)
}

func writeSuccessJSON(w http.ResponseWriter, jsonStr string) {
	fmt.Println("inside writeSuccessJSON serverCmd.go")
	w.Header().Set(contentTypeHeader, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(jsons.BPJ(jsonStr)))
	log.Print(http.StatusOK)
}

func writeSuccess(w http.ResponseWriter, v interface{}) {
	fmt.Println("inside writeSuccess serverCmd.go")
	writeSuccessJSON(w, jsons.BPM(v))
}

type reqTransform struct {
	Schema     string            `json:"schema"`
	Input      string            `json:"input"`
	Properties map[string]string `json:"properties"`
}

func httpPostTransform(w http.ResponseWriter, r *http.Request) {
	fmt.Println("inside httpposttransform serverCmd.go")
	log.Printf("Serving POST '/transform' request from %s ... ", r.RemoteAddr)
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("bad request: unable to read request body. err: %s", err))
		return
	}
	var req reqTransform
	err = json.Unmarshal(b, &req)
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("bad request: invalid request body. err: %s", err))
		return
	}
	s, err := omniparser.NewSchema("test-schema", strings.NewReader(req.Schema))
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("bad request: invalid schema. err: %s", err))
		return
	}
	t, err := s.NewTransform(
		"test-input", strings.NewReader(req.Input), &transformctx.Ctx{ExternalProperties: req.Properties})
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("bad request: unable to new transform. err: %s", err))
		return
	}
	var records []string
	for {
		b, err := t.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			writeBadRequest(w, fmt.Sprintf("bad request: transform failed. err: %s", err))
			return
		}
		records = append(records, string(b))
	}
	writeSuccessJSON(w, "["+strings.Join(records, ",")+"]")
	log.Print(jsons.BPM(req))
}

var (
	sampleDir                  = "../../extensions/omniv21/samples/"
	sampleFormats              = []string{"csv2", "json", "xml", "fixedlength2", "edi"}
	sampleInputFilenamePattern = regexp.MustCompile("^([0-9]+[_a-zA-Z0-9]+)\\.input\\.[a-z]+$")
)

type sample struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
	Input  string `json:"input"`
}

func httpGetSamples(w http.ResponseWriter, r *http.Request) {
	fmt.Println("inside httpGetSamples serverCmd.go")
	log.Printf("")
	log.Printf("Serving GET '/samples' request from %s ... ", r.RemoteAddr)
	samples := []sample{}
	for _, format := range sampleFormats {
		dir := filepath.Join(serverCmdDir(), sampleDir, format)
		fmt.Println("Directory httpGetSamples", dir)
		files, err := ioutil.ReadDir(dir)
		fmt.Println("Files httpGetSamples", files)
		if err != nil {
			goto getSampleFailure
		}
		for _, f := range files {
			submatch := sampleInputFilenamePattern.FindStringSubmatch(f.Name())
			fmt.Println("Submatch httpGetSamples", submatch)
			if len(submatch) < 2 {
				continue
			}
			sample := sample{
				Name: filepath.Join(format, submatch[1]),
			}
			fmt.Println("sample httpGetSamples", sample)
			schema, err := ioutil.ReadFile(filepath.Join(dir, submatch[1]+".schema.json"))
			fmt.Println("Schema httpGetsamples", schema)
			if err != nil {
				goto getSampleFailure
			}
			sample.Schema = string(schema)
			input, err := ioutil.ReadFile(filepath.Join(dir, f.Name()))
			fmt.Println("inputFiles httpGetsamples", input)
			if err != nil {
				goto getSampleFailure
			}
			sample.Input = string(input)
			samples = append(samples, sample)
		}
	}
	writeSuccess(w, samples)
	return

getSampleFailure:
	writeInternalServerError(w, "unable to get samples")
	return
}

func httpGetVersion(w http.ResponseWriter, r *http.Request) { //Invoked First
	fmt.Println("inside httpGetversion serverCmd.go")
	log.Printf("Serving GET '/version' request from %s ... ", r.RemoteAddr)
	writeSuccess(w, build)
}
