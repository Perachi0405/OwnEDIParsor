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
	"github.com/spf13/cobra"

	"github.com/Perachi0405/ownediparse"
	"github.com/Perachi0405/ownediparse/transformctx"
)

var (
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Launches op into HTTP server mode with its REST APIs.",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Println("invoke the doServer()")
			doServer()
		},
	}
	port int
)

func init() {
	// fmt.Println("Init in serverCmd.go")
	serverCmd.Flags().IntVarP(&port, "port", "p", 8080, "the listening HTTP port")
}

const (
	contentTypeHeader = "Content-Type"
	contentTypeJSON   = "application/json"
)

func doServer() {
	// fmt.Println("Inside the doServer()")
	transformRouter := chi.NewRouter()
	transformRouter.Use(middleware.RealIP)
	transformRouter.Use(middleware.AllowContentType(contentTypeJSON))
	transformRouter.Post("/", httpPostTransform)

	samplesRouter := chi.NewRouter()
	samplesRouter.Get("/", httpGetSamples)

	versionRouter := chi.NewRouter()
	versionRouter.Get("/", httpGetVersion)

	rootRouter := chi.NewRouter()
	rootRouter.Get("/", func(w http.ResponseWriter, req *http.Request) {
		http.FileServer(http.Dir(filepath.Join(serverCmdDir(), "web"))).ServeHTTP(w, req)
	})
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
	absDir, _ := filepath.Abs(filepath.Dir(filename))
	// fmt.Println("absDir", absDir)
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
	w.Header().Set(contentTypeHeader, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(jsons.BPJ(jsonStr)))
	log.Print(http.StatusOK)
}

//writing the success response
func writeSuccess(w http.ResponseWriter, v interface{}) {
	writeSuccessJSON(w, jsons.BPM(v))
}

type reqTransform struct {
	Schema     string            `json:"schema"`
	Input      string            `json:"input"`
	Properties map[string]string `json:"properties"`
}

func httpPostTransform(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving POST '/transform' request from %s ... ", r.RemoteAddr)
	b, err := ioutil.ReadAll(r.Body)
	// fmt.Println("Body Request", string(b))
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("bad request: unable to read request body. err: %s", err))
		return
	}
	var req reqTransform
	// fmt.Println("Unmarshal", json.Unmarshal(b, &req))
	err = json.Unmarshal(b, &req)
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("bad request: invalid request body. err: %s", err))
		return
	}
	s, err := ownediparse.NewSchema("test-schema", strings.NewReader(req.Schema)) // sends the name and the reader
	// fmt.Println("NewSchemaOutput", string(s.Content()))                          //validated Json schema in JSON format 'without \n special characters'
	//Getting the transformed data
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("bad request: invalid schema. err: %s", err))
		return
	}
	t, err := s.NewTransform(
		"test-input", strings.NewReader(req.Input), &transformctx.Ctx{ExternalProperties: req.Properties}) // passing the name, Input file and properties with the schema s,
	fmt.Println("output of newTransform", t)
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("bad request: unable to new transform. err: %s", err))
		return
	}
	var records []string
	for {
		b, err := t.Read()              //the Read func is called for many times
		fmt.Println("Reading t for", t) //unknown data in object format. TransformInstance &{0xc000deeec0 <nil> 0xc000098080}

		if err == io.EOF {
			break
		}
		if err != nil {
			writeBadRequest(w, fmt.Sprintf("bad request: transform failed. err: %s", err))
			return
		}
		// fmt.Println("Data type", b) //transformed Data in byte type
		records = append(records, string(b))
	}
	//fmt.Println("Transformed records", records)
	writeSuccessJSON(w, "["+strings.Join(records, ",")+"]") // return the parsed file to the UI
	// log.Print(jsons.BPM(req))
}

var (
	sampleDir                  = "../../extensions/omniv21/samples/"
	sampleFormats              = []string{"csv2", "json", "xml", "fixedlength2", "edi"}
	sampleInputFilenamePattern = regexp.MustCompile("^([0-9]+[_a-zA-Z0-9]+)\\.input\\.[a-z]+$")
)

//
type sample struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
	Input  string `json:"input"`
}

//
func httpGetSamples(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving GET '/samples' request from %s ... ", r.RemoteAddr)
	samples := []sample{}
	for _, format := range sampleFormats {
		dir := filepath.Join(serverCmdDir(), sampleDir, format)
		files, err := ioutil.ReadDir(dir)
		// fmt.Println("dirName", dir)
		// fmt.Println("Filename", files) //getting the files from each folder
		if err != nil {
			goto getSampleFailure
		}
		for _, f := range files {
			submatch := sampleInputFilenamePattern.FindStringSubmatch(f.Name()) //make a filename from the schema name
			// fmt.Println("submatch", submatch)
			if len(submatch) < 2 {
				continue
			}
			sample := sample{
				Name: filepath.Join(format, submatch[1]),
			} //Schema name
			// fmt.Println("sample name", sample)
			schema, err := ioutil.ReadFile(filepath.Join(dir, submatch[1]+".schema.json")) //Reading the schema
			// fmt.Println("Schema readed", schema)
			if err != nil {
				goto getSampleFailure
			}
			sample.Schema = string(schema) // give values to the schema key
			input, err := ioutil.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				goto getSampleFailure
			}
			sample.Input = string(input)      // give values to the input key
			samples = append(samples, sample) // return the response array of names
		}
	}
	writeSuccess(w, samples) //Return the 200
	return

getSampleFailure:
	writeInternalServerError(w, "unable to get samples")
	return
}

//invoked at first
func httpGetVersion(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving GET '/version' request from %s ... ", r.RemoteAddr)
	writeSuccess(w, build)
}
